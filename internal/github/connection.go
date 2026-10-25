package github

import (
	"github.com/alexfalkowski/abgleich/internal/transport/http/client"
	"github.com/alexfalkowski/go-service/v2/di"
	"github.com/alexfalkowski/go-service/v2/env"
	"github.com/alexfalkowski/go-service/v2/id"
	"github.com/alexfalkowski/go-service/v2/telemetry/logger"
	"github.com/alexfalkowski/go-service/v2/transport/http"
	"github.com/alexfalkowski/go-service/v2/transport/http/retry"
)

// ConnectionParams supplies shared HTTP settings and lifecycle dependencies.
type ConnectionParams struct {
	di.In

	Connections *client.Factory
	ID          id.Generator
	Config      *Config
	Logger      *logger.Logger
	UserAgent   env.UserAgent
}

// NewConnection constructs the shared transport. Only safe methods are retried;
// idle connections close after consumers stop using the client.
func NewConnection(params ConnectionParams) (*http.Client, error) {
	opts := []http.ClientOption{
		http.WithClientLogger(params.Logger),
		http.WithClientUserAgent(params.UserAgent),
		http.WithClientID(params.ID),
	}

	if cfg := params.Config.Client; cfg.IsEnabled() {
		opts = append(opts, http.WithClientRetry(retry.NewConfig(cfg.Retry), retry.SafeMethods))
	}

	return params.Connections.NewClient(params.Config.Client, params.Config.GetTimeout(), opts...)
}
