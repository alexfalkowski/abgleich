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
	// In marks this struct as an injected dependency group.
	di.In

	// Connections creates the GitHub connection pool and registers its shutdown cleanup.
	Connections *client.Factory
	// ID generates outbound request identifiers for the shared transport.
	ID id.Generator
	// Config supplies GitHub HTTP settings and the effective request timeout.
	Config *Config
	// Logger receives outbound HTTP transport logs.
	Logger *logger.Logger
	// UserAgent identifies this worker in outbound API requests.
	UserAgent env.UserAgent
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
