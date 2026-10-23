package github

import (
	"github.com/alexfalkowski/go-service/v2/errors"
	"github.com/alexfalkowski/go-service/v2/meta"
	"github.com/alexfalkowski/go-service/v2/net/http"
	"github.com/alexfalkowski/go-service/v2/net/http/body"
	"github.com/alexfalkowski/go-service/v2/os"
	"github.com/bradleyfalzon/ghinstallation/v2"
	"github.com/google/go-github/v93/github"
)

// Client supplies installation access to the worker and webhook. The private
// application client uses JWT authentication to validate installation ownership.
type Client struct {
	installation *github.Client
	application  *github.Client
}

// NewClient reads the App key once at startup. Missing or empty sources and
// invalid RSA PEMs prevent startup; replacement requires a restart. Signing and
// installation token renewal are handled by ghinstallation. Construction makes
// no GitHub requests.
func NewClient(cfg *Config, fs *os.FS, conn *http.Client) (*Client, error) {
	key, err := fs.ReadSource(cfg.App.PrivateKey)
	if err != nil {
		return nil, errors.Prefix("github app private key", err)
	}

	transport := &userAgentTransport{RoundTripper: conn.Transport}
	application, err := ghinstallation.NewAppsTransport(transport, cfg.App.ID, key)
	if err != nil {
		return nil, err
	}

	endpoint := cfg.GetURL()
	application.BaseURL = endpoint

	tokenConnection := *conn
	tokenConnection.Transport = transport
	tokenConnection.CheckRedirect = http.SameOriginRedirect

	installation := ghinstallation.NewFromAppsTransport(application, cfg.App.InstallationID)
	installation.Client = &tokenConnection

	appConnection, installationConnection := tokenConnection, tokenConnection
	appConnection.Transport = application
	installationConnection.Transport = &installationTransport{RoundTripper: installation, requests: make(chan struct{}, 1)}

	app, err := github.NewClient(github.WithHTTPClient(&appConnection), github.WithURLs(&endpoint, &endpoint))
	if err != nil {
		return nil, err
	}

	client, err := github.NewClient(github.WithHTTPClient(&installationConnection), github.WithURLs(&endpoint, &endpoint))
	if err != nil {
		return nil, err
	}

	return &Client{installation: client, application: app}, nil
}

// GitHub rejects Hookshot user agents on API requests. Clear the inbound
// identity so the shared transport uses the SDK header or application default.
type userAgentTransport struct {
	http.RoundTripper
}

func (t *userAgentTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	ctx := meta.WithAttributes(req.Context(), meta.WithUserAgent(meta.Blank()))
	return t.RoundTripper.RoundTrip(req.Clone(ctx))
}

// ghinstallation holds a mutex while renewing tokens; waiting for that mutex
// cannot be cancelled. A request slot bounds the wait by each caller's context.
type installationTransport struct {
	http.RoundTripper
	requests chan struct{}
}

func (t *installationTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	select {
	case t.requests <- struct{}{}:
		defer func() { <-t.requests }()
		return t.RoundTripper.RoundTrip(req)
	case <-req.Context().Done():
		body.Close(req.Body)
		return nil, req.Context().Err()
	}
}
