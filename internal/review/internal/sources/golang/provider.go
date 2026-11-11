package golang

import (
	"archive/zip"
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
)

//go:embed prompt.txt
var guide string

// NewProvider constructs credential-free Go source access without remote calls.
func NewProvider(cfg *provider.Config, connections *client.Factory) (provider.Result, error) {
	if !cfg.IsEnabled() {
		return provider.Result{}, nil
	}

	if !provider.ValidEndpoint(cfg.GetModuleProxy()) || !provider.ValidEndpoint(cfg.GetGoDocs()) {
		return provider.Result{}, errors.New("review: invalid public source endpoint")
	}

	connection, err := connections.NewClient(nil, provider.RequestTimeout)
	if err != nil {
		return provider.Result{}, errors.Prefix("review: source client", err)
	}
	connection.CheckRedirect = sourceRedirect

	return provider.Result{Strategy: &Provider{client: connection, proxy: strings.TrimRight(cfg.GetModuleProxy(), "/"), goDocs: strings.TrimRight(cfg.GetGoDocs(), "/")}}, nil
}

// Provider owns Go proxy and documentation access shared by review commands.
type Provider struct {
	client *http.Client
	proxy  string
	goDocs string
}

// Matches selects Go support for Go source or a module manifest.
func (*Provider) Matches(paths []string) bool {
	for _, name := range paths {
		if path.Ext(name) == ".go" || path.Base(name) == goModFile {
			return true
		}
	}

	return false
}

// Guide returns worker-owned Go review instructions.
func (*Provider) Guide() string { return guide }

// Tools binds Go lookups and independent archive/reference caches to one review.
func (p *Provider) Tools(repository provider.Repository) []ai.Tool {
	s := &session{provider: p, repository: repository, archives: make(map[string]*zip.Reader), files: make(map[string]string)}
	return []ai.Tool{
		{
			Name: "read_go_reference", Parameters: provider.Schema("manifest"), Call: s.referenceTool,
			Description: "Read official language release notes for the Go version declared in a repository go.mod. " +
				"Use before making language-version/compiler compatibility claims. manifest is the repository-relative go.mod path; " +
				"the worker chooses the version and trusted documentation endpoint. No arbitrary URLs or versions.",
		},
		{
			Name: "read_go_dependency", Parameters: provider.Schema("manifest", "module", "path", "start_line", "end_line"), Call: s.dependencyTool,
			Description: "Read a public Go module using the reviewed go.mod version and go.sum checksum. " +
				"manifest: repository-relative go.mod; module: required module path; path: exact file or directory (. lists the module root). " +
				"Directories return sorted paths and byte sizes without source. " +
				"For an exact file excerpt, set inclusive one-based start_line and end_line; set both null for the complete file. " +
				"No arbitrary versions or URLs. Inspect local replacements and vendored code with read_sources.",
		},
	}
}

type session struct {
	provider   *Provider
	repository provider.Repository
	archives   map[string]*zip.Reader
	files      map[string]string
}

func (s *session) dependencyTool(ctx context.Context, input string) (string, error) {
	var args dependencyRequest
	if err := json.Unmarshal(strings.Bytes(input), &args); err != nil {
		return strings.Empty, errors.Prefix("review: invalid dependency lookup", provider.ErrInvalidLookup)
	}

	if err := args.Validate(); err != nil {
		return strings.Empty, err
	}

	return s.dependency(ctx, args)
}

func (s *session) referenceTool(ctx context.Context, input string) (string, error) {
	var args referenceRequest
	if err := json.Unmarshal(strings.Bytes(input), &args); err != nil {
		return strings.Empty, errors.Prefix("review: invalid Go reference lookup", provider.ErrInvalidLookup)
	}

	return s.goReference(ctx, args.Manifest)
}
