package golang

import (
	"path"

	"github.com/alexfalkowski/go-service/v3/context"
	"github.com/alexfalkowski/go-service/v3/errors"
	"github.com/alexfalkowski/go-service/v3/io"
	"github.com/alexfalkowski/go-service/v3/net/http"
	"github.com/alexfalkowski/go-service/v3/net/url"
	"github.com/alexfalkowski/go-service/v3/strings"
)

const maxSourceRedirects = 10

type source struct {
	client   *http.Client
	endpoint string
}

func (s source) read(ctx context.Context, suffix string) ([]byte, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, s.endpoint+suffix, nil)
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

// sourceRedirect rejects URL credentials and permits the public Go proxy's
// archive bucket in addition to the original origin. Checking the initial
// origin constrains every later hop.
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
