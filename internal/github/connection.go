package github

import (
	"github.com/alexfalkowski/go-service/v2/context"
	"github.com/alexfalkowski/go-service/v2/di"
	"github.com/alexfalkowski/go-service/v2/env"
	"github.com/alexfalkowski/go-service/v2/id"
	"github.com/alexfalkowski/go-service/v2/telemetry/logger"
	"github.com/alexfalkowski/go-service/v2/transport/http"
	"github.com/alexfalkowski/go-service/v2/transport/http/breaker"
	"github.com/alexfalkowski/go-service/v2/transport/http/retry"
)

// ConnectionParams supplies shared HTTP settings and lifecycle dependencies.
type ConnectionParams struct {
	di.In

	Lifecycle di.Lifecycle
	ID        id.Generator
	Config    *Config
	Logger    *logger.Logger
	UserAgent env.UserAgent
}

// NewConnection constructs the shared transport. Only safe methods are retried;
// idle connections close after consumers stop using the client.
func NewConnection(params ConnectionParams) (*http.Client, error) {
	opts := []http.ClientOption{
		http.WithClientLogger(params.Logger),
		http.WithClientUserAgent(params.UserAgent),
		http.WithClientID(params.ID),
		http.WithClientTimeout(params.Config.GetTimeout()),
	}

	if cfg := params.Config.Client; cfg.IsEnabled() {
		opts = append(opts,
			http.WithClientBreaker(breaker.NewConfig(cfg.Breaker)),
			http.WithClientRetry(retry.NewConfig(cfg.Retry), retry.SafeMethods),
			http.WithClientTLS(cfg.TLS),
		)
	}

	connection, err := http.NewClient(opts...)
	if err != nil {
		return nil, err
	}

	params.Lifecycle.Append(di.Hook{
		OnStop: func(context.Context) error {
			connection.CloseIdleConnections()
			return nil
		},
	})

	return connection, nil
}
