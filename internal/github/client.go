package github

import (
	"github.com/alexfalkowski/go-service/v2/bytes"
	"github.com/alexfalkowski/go-service/v2/context"
	"github.com/alexfalkowski/go-service/v2/di"
	"github.com/alexfalkowski/go-service/v2/env"
	"github.com/alexfalkowski/go-service/v2/errors"
	"github.com/alexfalkowski/go-service/v2/id"
	"github.com/alexfalkowski/go-service/v2/os"
	"github.com/alexfalkowski/go-service/v2/telemetry/logger"
	transporthttp "github.com/alexfalkowski/go-service/v2/transport/http"
	"github.com/alexfalkowski/go-service/v2/transport/http/breaker"
	"github.com/alexfalkowski/go-service/v2/transport/http/retry"
	"github.com/google/go-github/v93/github"
)

// Client aliases [github.Client] so consumers use the project-owned package.
type Client = github.Client

// Params supplies the GitHub endpoint, shared HTTP configuration, and injected
// transport dependencies. Credentials are resolved through Filesystem at startup.
type Params struct {
	di.In

	Lifecycle  di.Lifecycle
	ID         id.Generator
	Config     *Config
	Filesystem *os.FS
	Logger     *logger.Logger
	UserAgent  env.UserAgent
}

// NewClient constructs a reusable API client without contacting GitHub. It reads
// the token source once at startup; a missing or empty token prevents startup.
// Credential changes require a process restart.
func NewClient(params Params) (*Client, error) {
	token, err := params.Filesystem.ReadSource(params.Config.Token)
	if err != nil {
		return nil, errors.Prefix("github token", err)
	}

	endpoint := params.Config.GetURL()
	opts := []transporthttp.ClientOption{
		transporthttp.WithClientLogger(params.Logger),
		transporthttp.WithClientUserAgent(params.UserAgent),
		transporthttp.WithClientID(params.ID),
		transporthttp.WithClientTimeout(params.Config.GetTimeout()),
	}

	if cfg := params.Config.Client; cfg.IsEnabled() {
		opts = append(opts,
			transporthttp.WithClientBreaker(breaker.NewConfig(cfg.Breaker)),
			transporthttp.WithClientRetry(retry.NewConfig(cfg.Retry), retry.SafeMethods),
			transporthttp.WithClientTLS(cfg.TLS),
		)
	}

	connection, err := transporthttp.NewClient(opts...)
	if err != nil {
		return nil, err
	}

	params.Lifecycle.Append(di.Hook{
		OnStop: func(context.Context) error {
			connection.CloseIdleConnections()
			return nil
		},
	})

	return github.NewClient(
		github.WithHTTPClient(connection),
		github.WithURLs(&endpoint, &endpoint),
		github.WithAuthToken(bytes.String(token)),
	)
}
