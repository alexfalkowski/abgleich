package client

import (
	"github.com/alexfalkowski/go-service/v2/config/client"
	"github.com/alexfalkowski/go-service/v2/context"
	"github.com/alexfalkowski/go-service/v2/crypto/tls"
	"github.com/alexfalkowski/go-service/v2/di"
	"github.com/alexfalkowski/go-service/v2/net/http"
	"github.com/alexfalkowski/go-service/v2/os"
	"github.com/alexfalkowski/go-service/v2/time"
	transporthttp "github.com/alexfalkowski/go-service/v2/transport/http"
	"github.com/alexfalkowski/go-service/v2/transport/http/breaker"
)

// NewFactory binds client cleanup to the application lifecycle. Construct clients
// before registering consumers so shutdown stops those consumers first.
func NewFactory(lifecycle di.Lifecycle, fs *os.FS) *Factory {
	return &Factory{lifecycle: lifecycle, fs: fs}
}

// Factory creates independent HTTP clients whose pools close during shutdown.
// Callers select their timeout and any retry or logging middleware.
type Factory struct {
	lifecycle di.Lifecycle
	fs        *os.FS
}

// NewClient resolves optional TLS sources and circuit breaking from cfg without
// making network requests. Invalid TLS sources return an error. Retries remain
// disabled unless opts enables them; cfg.Retry is left to the caller's policy.
// Each client has its own pool and rejects redirects to a different origin.
func (f *Factory) NewClient(cfg *client.Config, timeout time.Duration, opts ...transporthttp.ClientOption) (*http.Client, error) {
	var tlsConfig *tls.Config
	if cfg.IsEnabled() && cfg.TLS.IsEnabled() {
		var err error
		tlsConfig, err = client.NewConfig(f.fs, cfg.TLS)
		if err != nil {
			return nil, err
		}
	}

	// Keep the pool because HTTP middleware does not forward CloseIdleConnections.
	pool := http.Transport(tlsConfig)
	options := []transporthttp.ClientOption{
		transporthttp.WithClientTimeout(timeout),
		transporthttp.WithClientRoundTripper(pool),
	}
	if cfg.IsEnabled() {
		options = append(options, transporthttp.WithClientBreaker(breaker.NewConfig(cfg.Breaker)))
	}
	options = append(options, opts...)

	connection, err := transporthttp.NewClient(options...)
	if err != nil {
		return nil, err
	}

	connection.CheckRedirect = http.SameOriginRedirect
	f.lifecycle.Append(di.Hook{
		OnStop: func(context.Context) error {
			pool.CloseIdleConnections()
			return nil
		},
	})

	return connection, nil
}
