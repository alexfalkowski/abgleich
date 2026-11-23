package github

import (
	"github.com/alexfalkowski/go-service/v3/context"
	"github.com/alexfalkowski/go-service/v3/env"
	"github.com/alexfalkowski/go-service/v3/errors"
	"github.com/alexfalkowski/go-service/v3/id"
	"github.com/alexfalkowski/go-service/v3/meta"
	"github.com/alexfalkowski/go-service/v3/net/http"
	"github.com/alexfalkowski/go-service/v3/net/http/body"
	"github.com/alexfalkowski/go-service/v3/os"
	"github.com/alexfalkowski/go-service/v3/strings"
	"github.com/alexfalkowski/go-service/v3/telemetry/logger"
	transporthttp "github.com/alexfalkowski/go-service/v3/transport/http"
	"github.com/alexfalkowski/go-service/v3/transport/http/retry"
	"github.com/alexfalkowski/go-signal/v2"
	"github.com/bradleyfalzon/ghinstallation/v2"
	"github.com/google/go-github/v93/github"
)

// ClientParams supplies startup credentials and the GitHub transport settings.
type ClientParams struct {
	// In marks the injected dependencies.
	signal.In

	// Lifecycle closes the shared pool during shutdown.
	Lifecycle signal.Lifecycle

	// ID generates outbound request identifiers.
	ID id.Generator

	// Config supplies the endpoint and App identity.
	Config *Config

	// FS resolves the configured App private key at startup.
	FS *os.FS

	// Logger records outbound HTTP outcomes.
	Logger *logger.Logger

	// UserAgent identifies the worker in outbound API requests.
	UserAgent env.UserAgent
}

// NewClient reads the App key once at startup. Missing or empty sources and
// invalid RSA PEMs prevent startup; replacement requires a restart. Signing and
// installation token renewal are handled by ghinstallation. Construction makes
// no GitHub requests.
func NewClient(params ClientParams) (*Client, error) {
	cfg := params.Config
	key, err := params.FS.ReadSource(cfg.App.PrivateKey)
	if err != nil {
		return nil, errors.Prefix("github: app private key", err)
	}

	conn, endpoint, err := transporthttp.NewClientFromConfig(cfg.Client,
		transporthttp.WithClientLifecycle(params.Lifecycle),
		transporthttp.WithClientTimeout(cfg.GetTimeout()),
		transporthttp.WithClientRetry(retry.NewConfig(cfg.Client.Retry), retry.SafeMethods),
		transporthttp.WithClientLogger(params.Logger),
		transporthttp.WithClientUserAgent(params.UserAgent),
		transporthttp.WithClientID(params.ID),
	)
	if err != nil {
		return nil, errors.Prefix("github.client", err)
	}
	conn.CheckRedirect = http.SameOriginRedirect

	// Copy HTTP settings so token renewal, App requests, and installation requests
	// can use different transports while sharing the underlying connection pool.
	tokenConnection := *conn
	transport := &userAgentTransport{RoundTripper: conn.Transport}
	tokenConnection.Transport = transport

	applicationTransport, err := ghinstallation.NewAppsTransport(transport, cfg.App.ID, key)
	if err != nil {
		return nil, err
	}

	apiURL := endpoint.String()
	applicationTransport.BaseURL = apiURL

	applicationConnection := tokenConnection
	applicationConnection.Transport = applicationTransport
	application, err := github.NewClient(github.WithHTTPClient(&applicationConnection), github.WithURLs(&apiURL, &apiURL))
	if err != nil {
		return nil, err
	}

	installationAuth := ghinstallation.NewFromAppsTransport(applicationTransport, cfg.App.InstallationID)
	installationAuth.Client = &tokenConnection
	installationConnection := tokenConnection
	installationConnection.Transport = &installationTransport{RoundTripper: transport, authentication: installationAuth, requests: make(chan struct{}, 1)}

	installation, err := github.NewClient(github.WithHTTPClient(&installationConnection), github.WithURLs(&apiURL, &apiURL))
	if err != nil {
		return nil, err
	}

	return &Client{installation: installation, application: application}, nil
}

// Client supplies installation access. The application client uses JWTs to
// validate installation ownership.
type Client struct {
	installation *github.Client
	application  *github.Client
}

// GitHub rejects Hookshot user agents on API requests. Clear the inbound
// identity so the shared transport uses the SDK header or application default.
type userAgentTransport struct {
	http.RoundTripper
}

// RoundTrip clears the inbound user-agent metadata on a cloned request so the
// wrapped transport uses the SDK or worker identity when contacting GitHub.
func (t *userAgentTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	ctx := meta.WithAttributes(req.Context(), meta.WithUserAgent(meta.Blank()))
	return t.RoundTripper.RoundTrip(req.Clone(ctx))
}

// ghinstallation holds a mutex while renewing tokens; waiting for that mutex
// cannot be cancelled. Serialize only token acquisition so waiting remains
// cancellable without making webhook commands wait for background API requests.
type installationTransport struct {
	http.RoundTripper
	authentication *ghinstallation.Transport
	requests       chan struct{}
}

// RoundTrip obtains an installation token with cancellable waiting, then sends
// an authenticated request through the wrapped transport. Token failures close
// the request body without sending it.
func (t *installationTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	token, err := t.token(req.Context())
	if err != nil {
		body.Close(req.Body)
		return nil, err
	}

	request := req.Clone(req.Context())
	request.Header.Set("Authorization", "token "+token)
	if strings.IsEmpty(request.Header.Get(http.AcceptKey)) {
		request.Header.Set(http.AcceptKey, "application/vnd.github.v3+json")
	}

	return t.RoundTripper.RoundTrip(request)
}

func (t *installationTransport) token(ctx context.Context) (string, error) {
	select {
	case t.requests <- struct{}{}:
		defer func() {
			<-t.requests
		}()

		if err := ctx.Err(); err != nil {
			return "", err
		}

		return t.authentication.Token(ctx)
	case <-ctx.Done():
		return "", ctx.Err()
	}
}
