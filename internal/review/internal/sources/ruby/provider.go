package ruby

import (
	_ "embed"
	"path"

	"github.com/alexfalkowski/abgleich/internal/ai"
	"github.com/alexfalkowski/abgleich/internal/review/internal/sources/provider"
	"github.com/alexfalkowski/abgleich/internal/transport/http/client"
	"github.com/alexfalkowski/go-service/v2/context"
	"github.com/alexfalkowski/go-service/v2/encoding/json"
	"github.com/alexfalkowski/go-service/v2/errors"
	"github.com/alexfalkowski/go-service/v2/net/http"
	"github.com/alexfalkowski/go-service/v2/strings"
	transporthttp "github.com/alexfalkowski/go-service/v2/transport/http"
	httpretry "github.com/alexfalkowski/go-service/v2/transport/http/retry"
	"github.com/alexfalkowski/go-service/v2/transport/retry"
)

//go:embed prompt.txt
var guide string

// NewProvider constructs credential-free RubyGems access without remote calls.
func NewProvider(cfg *provider.Config, connections *client.Factory) (provider.Result, error) {
	if !cfg.IsEnabled() {
		return provider.Result{}, nil
	}

	if !provider.ValidEndpoint(cfg.GetGemRegistry()) {
		return provider.Result{}, errors.New("review: invalid public source endpoint")
	}

	retries := httpretry.NewConfig(&retry.Config{Attempts: 2})
	connection, err := connections.NewClient(nil, provider.RequestTimeout, transporthttp.WithClientRetry(retries, httpretry.SafeMethods))
	if err != nil {
		return provider.Result{}, errors.Prefix("review: source client", err)
	}

	return provider.Result{Strategy: &Provider{client: connection, gemRegistry: strings.TrimRight(cfg.GetGemRegistry(), "/")}}, nil
}

// Provider shares a credential-free RubyGems HTTP client across review commands.
// Its tools keep verified compressed archives independently for each review.
type Provider struct {
	client      *http.Client
	gemRegistry string
}

// Matches selects Ruby support for Ruby source or a Bundler manifest/lockfile.
func (*Provider) Matches(paths []string) bool {
	for _, name := range paths {
		switch path.Ext(name) {
		case ".rb", ".gemspec":
			return true
		}

		switch path.Base(name) {
		case "Gemfile", "Gemfile.lock":
			return true
		}
	}

	return false
}

// Guide returns worker-owned Ruby review instructions.
func (*Provider) Guide() string { return guide }

// Tools binds Ruby lookup and an independent archive cache to one review.
func (p *Provider) Tools(repository provider.Repository) []ai.Tool {
	s := &session{provider: p, repository: repository}
	return []ai.Tool{
		{
			Name: "read_ruby_dependency", Parameters: provider.Schema("manifest", "gem", "platform", "path", "start_line", "end_line"), Call: s.dependencyTool,
			Description: "Read a public Ruby gem at the exact version in the reviewed Gemfile.lock. " +
				"manifest: repository-relative lockfile; gem: locked name; platform: locked platform, or ruby for a generic gem; " +
				"path: exact file or directory (. lists files). Optional inclusive one-based start_line/end_line select an excerpt; set both null for a whole file. " +
				"Archives are checksum verified. No arbitrary versions or URLs. Inspect Git, local, and vendored dependencies through repository source.",
		},
	}
}

type session struct {
	provider   *Provider
	repository provider.Repository
	gems       map[string][]byte
	bytesRead  int64
}

func (s *session) dependencyTool(ctx context.Context, input string) (string, error) {
	var args rubyDependencyRequest
	if err := json.Unmarshal(strings.Bytes(input), &args); err != nil {
		return strings.Empty, errors.Prefix("review: invalid Ruby dependency lookup", provider.ErrInvalidLookup)
	}

	return s.rubyDependency(ctx, args)
}
