package provider

import (
	"github.com/alexfalkowski/go-service/v3/config/client"
	"github.com/alexfalkowski/go-service/v3/errors"
	"github.com/alexfalkowski/go-service/v3/net/url"
	"github.com/alexfalkowski/go-service/v3/os"
	"github.com/alexfalkowski/go-service/v3/strings"
	"github.com/alexfalkowski/go-service/v3/time"
	"github.com/alexfalkowski/go-service/v3/transport/http"
	"github.com/alexfalkowski/go-service/v3/transport/http/retry"
	"github.com/alexfalkowski/go-signal/v2"
)

const defaultRequestTimeout = 10 * time.Second

// ClientOptions applies the public source transport policy. It uses fs to load
// trusted TLS roots, rejects client credentials, defaults the request timeout
// to ten seconds, and restricts any configured retry policy to safe methods.
func ClientOptions(cfg *client.Config, lifecycle signal.Lifecycle, fs *os.FS) ([]http.ClientOption, error) {
	if cfg.TLS.HasKeyMaterial() || cfg.Token != nil {
		return nil, errors.New("public source client requires credential-free access")
	}

	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = defaultRequestTimeout
	}

	return []http.ClientOption{
		http.WithClientLifecycle(lifecycle),
		http.WithClientFS(fs),
		http.WithClientTimeout(timeout),
		http.WithClientRetry(retry.NewConfig(cfg.Retry), retry.SafeMethods),
	}, nil
}

// ValidateEndpoint rejects credentials, queries, and fragments in a source endpoint.
func ValidateEndpoint(endpoint *url.SecureURL) error {
	parsed := endpoint.URL()
	if parsed.User != nil || parsed.ForceQuery || !strings.IsEmpty(parsed.RawQuery) || !strings.IsEmpty(parsed.Fragment) {
		return errors.New("endpoint must be free of credentials, queries, and fragments")
	}
	return nil
}
