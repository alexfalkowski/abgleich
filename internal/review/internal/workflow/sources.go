package workflow

import (
	"archive/zip"
	"fmt"
	"io"
	"io/fs"
	"net/url"
	"path"
	"slices"

	"github.com/alexfalkowski/abgleich/internal/github"
	"github.com/alexfalkowski/abgleich/internal/transport/http/client"
	"github.com/alexfalkowski/go-service/v2/context"
	"github.com/alexfalkowski/go-service/v2/errors"
	"github.com/alexfalkowski/go-service/v2/net/http"
	"github.com/alexfalkowski/go-service/v2/strings"
	"github.com/alexfalkowski/go-service/v2/time"
)

const (
	sourceRequestTimeout = 10 * time.Second
	maxSourceRedirects   = 10
)

// NewSources constructs a credential-free connection to the trusted Go proxy.
// Archive redirects may reach the Go proxy's HTTPS storage bucket. It never
// executes Go or extracts archives to the filesystem.
func NewSources(cfg *Config, connections *client.Factory) (*Sources, error) {
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
// credentials. Each review session owns its pinned repository source and caches.
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

	content, err := io.ReadAll(response.Body)
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
	trees      map[string][]string
	archives   map[string]*zip.Reader
	headSHA    string
	baseSHA    string
	missing    []string
	inspected  []string
}

func (s *sourceSession) list(ctx context.Context, filter string) ([]string, error) {
	files, err := s.treeFiles(ctx, s.headSHA)
	if err != nil {
		return nil, errors.New("review: repository tree")
	}

	var paths []string
	for _, name := range files {
		if strings.Contains(name, filter) {
			paths = append(paths, name)
		}
	}
	return paths, nil
}

func (s *sourceSession) read(ctx context.Context, filename string) (string, error) {
	if !fs.ValidPath(filename) || strings.Contains(filename, "\\") {
		return strings.Empty, errors.New("review: invalid repository path")
	}

	paths, err := s.treeFiles(ctx, s.headSHA)
	if err != nil {
		return strings.Empty, errors.New("review: repository tree")
	}

	if !slices.Contains(paths, filename) {
		return s.directory(ctx, filename, paths), nil
	}
	return "Source: " + filename + " at " + s.headSHA + "\n" + s.repositoryFile(ctx, filename, false), nil
}

func (s *sourceSession) directory(ctx context.Context, directory string, paths []string) string {
	var source strings.Builder
	for _, filename := range paths {
		if path.Dir(filename) != directory {
			continue
		}

		if err := ctx.Err(); err != nil {
			break
		}

		fmt.Fprintf(&source, "\nSource: %s at %s\n", filename, s.headSHA)
		source.WriteString(s.repositoryFile(ctx, filename, false))
		source.WriteString("\n")
	}

	if source.Len() == 0 {
		return s.unavailable(directory)
	}
	return source.String()
}

func (s *sourceSession) repositoryFile(ctx context.Context, filename string, base bool) string {
	if !fs.ValidPath(filename) || strings.Contains(filename, "\\") {
		return s.unavailable("invalid repository path")
	}

	revision := s.headSHA
	if base {
		revision = s.baseSHA
	}

	key := revision + ":" + filename
	if content, ok := s.files[key]; ok {
		return content
	}

	paths, err := s.treeFiles(ctx, revision)
	if err != nil || !slices.Contains(paths, filename) {
		return s.unavailable(key + " (regular file unavailable)")
	}

	content, err := s.github.ReadSourceFile(ctx, s.repository, revision, filename)
	if err != nil {
		return s.unavailable(key + " (" + err.Error() + ")")
	}

	s.inspected = append(s.inspected, key)
	s.files[key] = content
	return content
}

func (s *sourceSession) treeFiles(ctx context.Context, revision string) ([]string, error) {
	if paths, ok := s.trees[revision]; ok {
		return paths, nil
	}

	paths, err := s.github.ListSourceFiles(ctx, s.repository, revision)
	if err != nil {
		return nil, err
	}

	s.trees[revision] = paths
	return paths, nil
}

func (s *sourceSession) unavailable(label string) string {
	message := "Source unavailable: " + label
	if !slices.Contains(s.missing, message) {
		s.missing = append(s.missing, message)
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
