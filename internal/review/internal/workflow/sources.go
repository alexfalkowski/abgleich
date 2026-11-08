package workflow

import (
	"net/url"
	"path"

	"github.com/alexfalkowski/abgleich/internal/transport/http/client"
	"github.com/alexfalkowski/go-service/v2/context"
	"github.com/alexfalkowski/go-service/v2/errors"
	"github.com/alexfalkowski/go-service/v2/io"
	"github.com/alexfalkowski/go-service/v2/net/http"
	"github.com/alexfalkowski/go-service/v2/strings"
	"github.com/alexfalkowski/go-service/v2/time"
)

const (
	sourceRequestTimeout = 10 * time.Second
	maxSourceRedirects   = 10
)

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
