package workflow

import (
	"archive/zip"
	"fmt"
	"net/url"
	"path"

	"github.com/alexfalkowski/abgleich/internal/github"
	"github.com/alexfalkowski/abgleich/internal/transport/http/client"
	"github.com/alexfalkowski/go-service/v2/context"
	"github.com/alexfalkowski/go-service/v2/errors"
	"github.com/alexfalkowski/go-service/v2/io"
	"github.com/alexfalkowski/go-service/v2/net/http"
	"github.com/alexfalkowski/go-service/v2/slices"
	"github.com/alexfalkowski/go-service/v2/strings"
	"github.com/alexfalkowski/go-service/v2/time"
)

const (
	sourceRequestTimeout = 10 * time.Second
	maxSourceRedirects   = 10
)

var errRepositorySource = errors.New("review: could not retrieve complete repository source")

// NewSources constructs a credential-free connection to the trusted Go proxy.
// Omitted configuration returns nil because no review feature needs source access.
// Archive redirects may reach the Go proxy's HTTPS storage bucket. It never
// executes Go or extracts archives to the filesystem.
func NewSources(cfg *Config, connections *client.Factory) (*Sources, error) {
	if !cfg.IsEnabled() {
		return nil, nil
	}

	if !validSourceEndpoint(cfg.GetModuleProxy()) || !validSourceEndpoint(cfg.GetGoDocs()) {
		return nil, errors.New("review: invalid public source endpoint")
	}

	connection, err := connections.NewClient(nil, sourceRequestTimeout)
	if err != nil {
		return nil, errors.Prefix("review: source client", err)
	}
	connection.CheckRedirect = sourceRedirect

	return &Sources{
		client: connection,
		proxy:  strings.TrimRight(cfg.GetModuleProxy(), "/"),
		goDocs: strings.TrimRight(cfg.GetGoDocs(), "/"),
	}, nil
}

// Sources provides public dependency files and language references without
// credentials. Dependency archives and references stay local to each review.
type Sources struct {
	client *http.Client
	proxy  string
	goDocs string
}

// fetch uses the credential-free transport and the caller's deadline for public source.
func (s *Sources) fetch(ctx context.Context, endpoint string) ([]byte, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, errors.Prefix("review: source request", err)
	}

	response, err := s.client.Do(request)
	if err != nil {
		return nil, errors.New("review: public source unavailable")
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, errors.New("review: public source unavailable")
	}

	content, _, err := io.ReadAll(response.Body)
	return content, errors.Prefix("review: read public source", err)
}

// sourceRedirect permits the public Go proxy's archive bucket in addition to
// the original origin. Checking the initial origin constrains every later hop.
func sourceRedirect(request *http.Request, via []*http.Request) error {
	if len(via) >= maxSourceRedirects {
		return errors.New("review: too many source redirects")
	}
	if request.URL.User != nil {
		return http.ErrUseLastResponse
	}

	origin := via[0].URL
	if http.SameOrigin(origin, request.URL) {
		return nil
	}

	proxy := http.SameOrigin(origin, &url.URL{Scheme: "https", Host: "proxy.golang.org"})
	archive := strings.Contains(origin.Path, "/@v/") && strings.HasSuffix(origin.Path, ".zip")
	storage := http.SameOrigin(request.URL, &url.URL{Scheme: "https", Host: "storage.googleapis.com"}) &&
		strings.HasPrefix(request.URL.Path, "/proxy-golang-org-prod/") &&
		strings.HasPrefix(path.Clean(request.URL.Path), "/proxy-golang-org-prod/")
	if proxy && archive && storage {
		return nil
	}

	return http.ErrUseLastResponse
}

type sourceSession struct {
	github     github.Service
	sources    *Sources
	repository *github.Repository
	files      map[string]string
	trees      map[string][]*github.TreeEntry
	archives   map[string]*zip.Reader
	headSHA    string
	baseSHA    string
	missing    []string
	external   []string
	inspected  []string
	binary     []string
}

func (s *sourceSession) list(ctx context.Context, filter string) ([]string, error) {
	files, err := s.treeFiles(ctx, s.headSHA)
	if err != nil {
		return nil, err
	}

	var paths []string
	for _, name := range files {
		if strings.Contains(name, filter) {
			paths = append(paths, name)
		}
	}

	return paths, nil
}

func (s *sourceSession) read(ctx context.Context, filenames []string) (string, error) {
	paths, err := s.treeFiles(ctx, s.headSHA)
	if err != nil {
		return strings.Empty, err
	}

	var selected []string
	for _, filename := range paths {
		for _, requested := range filenames {
			if filename == requested || path.Dir(filename) == requested {
				selected = append(selected, filename)
				break
			}
		}
	}

	contents, err := s.repositoryFiles(ctx, selected, false)
	if err != nil {
		return strings.Empty, err
	}

	var source strings.Builder
	for _, filename := range filenames {
		if err := ctx.Err(); err != nil {
			return strings.Empty, err
		}
		source.WriteString(s.readPath(ctx, filename, paths, contents))
		source.WriteString("\n")
	}

	return source.String(), nil
}

func (s *sourceSession) readPath(ctx context.Context, filename string, paths []string, contents map[string]string) string {
	if !io.ValidPath(filename) || strings.Contains(filename, "\\") {
		return s.unavailable("review: invalid repository path")
	}

	if !slices.Contains(paths, filename) {
		return s.directory(ctx, filename, paths, contents)
	}

	return strings.Concat("Source: ", filename, " at ", s.headSHA, "\n", contents[filename])
}

func (s *sourceSession) directory(ctx context.Context, directory string, paths []string, contents map[string]string) string {
	var source strings.Builder
	for _, filename := range paths {
		if path.Dir(filename) != directory {
			continue
		}

		if err := ctx.Err(); err != nil {
			break
		}

		fmt.Fprintf(&source, "\nSource: %s at %s\n", filename, s.headSHA)
		source.WriteString(contents[filename])
		source.WriteString("\n")
	}

	if source.Len() == 0 {
		return s.missingPath(directory)
	}

	return source.String()
}

// missingPath distinguishes absent paths from excluded source using the complete tree.
func (s *sourceSession) missingPath(filename string) string {
	if filename == "." {
		return s.unavailable(filename)
	}

	for _, entry := range s.trees[s.headSHA] {
		name := entry.GetPath()
		if name == filename || strings.HasPrefix(name, filename+"/") ||
			(entry.GetType() != github.TreeDirectory && !github.IsRegularSourceFile(entry) && strings.HasPrefix(filename, name+"/")) {
			return s.unavailable(filename)
		}
	}

	return strings.Concat("Source not found: ", filename)
}

func (s *sourceSession) repositoryFile(ctx context.Context, filename string, base bool) (string, error) {
	contents, err := s.repositoryFiles(ctx, []string{filename}, base)
	return contents[filename], err
}

func (s *sourceSession) repositoryFiles(ctx context.Context, filenames []string, base bool) (map[string]string, error) {
	revision := s.headSHA
	if base {
		revision = s.baseSHA
	}

	contents, missing := s.uncachedFiles(revision, filenames)
	if len(missing) == 0 {
		return contents, nil
	}

	files, err := s.regularFiles(ctx, revision, missing, contents)
	if err != nil || len(files) == 0 {
		return contents, err
	}

	current, err := s.github.ReadSourceFiles(ctx, s.repository, revision, files)
	if err != nil {
		return nil, errors.Join(errRepositorySource, err)
	}

	for _, file := range files {
		filename := file.GetPath()
		key := revision + ":" + filename
		if content, ok := current[filename]; ok {
			s.inspected = append(s.inspected, key)
			s.files[key] = content
			contents[filename] = content
			continue
		}

		content := "Source excluded: " + key + " (binary file)"
		s.binary = append(s.binary, key)
		s.files[key] = content
		contents[filename] = content
	}

	return contents, nil
}

func (s *sourceSession) uncachedFiles(revision string, filenames []string) (map[string]string, []string) {
	contents := make(map[string]string, len(filenames))
	seen := make(map[string]bool, len(filenames))

	var missing []string
	for _, filename := range filenames {
		if seen[filename] {
			continue
		}
		seen[filename] = true
		if !io.ValidPath(filename) || strings.Contains(filename, "\\") {
			contents[filename] = s.unavailable("invalid repository path")
			continue
		}
		if content, ok := s.files[revision+":"+filename]; ok {
			contents[filename] = content
			continue
		}
		missing = append(missing, filename)
	}

	return contents, missing
}

func (s *sourceSession) regularFiles(ctx context.Context, revision string, filenames []string, contents map[string]string) ([]*github.TreeEntry, error) {
	entries, err := s.treeEntries(ctx, revision)
	if err != nil {
		return nil, err
	}

	byPath := make(map[string]*github.TreeEntry, len(entries))
	for _, entry := range entries {
		byPath[entry.GetPath()] = entry
	}

	var files []*github.TreeEntry
	for _, filename := range filenames {
		if entry := byPath[filename]; github.IsRegularSourceFile(entry) {
			files = append(files, entry)
		} else {
			contents[filename] = s.unavailable(strings.Concat(revision, ":", filename, " (regular file unavailable)"))
		}
	}

	return files, nil
}

func (s *sourceSession) treeFiles(ctx context.Context, revision string) ([]string, error) {
	entries, err := s.treeEntries(ctx, revision)
	if err != nil {
		return nil, err
	}

	paths := make([]string, 0, len(entries))
	for _, entry := range entries {
		if github.IsRegularSourceFile(entry) {
			paths = append(paths, entry.GetPath())
		}
	}

	return paths, nil
}

func (s *sourceSession) treeEntries(ctx context.Context, revision string) ([]*github.TreeEntry, error) {
	if entries, ok := s.trees[revision]; ok {
		return entries, nil
	}

	entries, err := s.github.ListSourceFiles(ctx, s.repository, revision)
	if err != nil {
		return nil, errors.Join(errRepositorySource, err)
	}

	s.trees[revision] = entries
	return entries, nil
}

func (s *sourceSession) unavailable(label string) string {
	message := "Source unavailable: " + label
	if !slices.Contains(s.missing, message) {
		s.missing = append(s.missing, message)
	}

	return message
}

func (s *sourceSession) externalUnavailable(label string) string {
	message := "External source unavailable: " + label
	if !slices.Contains(s.external, message) {
		s.external = append(s.external, message)
	}

	return message
}

func validSourceEndpoint(raw string) bool {
	endpoint, err := url.Parse(raw)
	if err != nil {
		return false
	}
	if strings.IsEmpty(endpoint.Host) || endpoint.User != nil {
		return false
	}
	if !strings.IsEmpty(endpoint.RawQuery) || !strings.IsEmpty(endpoint.Fragment) {
		return false
	}

	return endpoint.Scheme == "https" || endpoint.Scheme == "http"
}
