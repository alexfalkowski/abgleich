package workflow

import (
	"github.com/alexfalkowski/go-service/v2/errors"
	"github.com/alexfalkowski/go-service/v2/io"
	"github.com/alexfalkowski/go-service/v2/strings"
)

// listSourceFilesRequest decodes the model's literal path filter for repository discovery.
type listSourceFilesRequest struct {
	Filter string `json:"filter"`
}

// readSourcesRequest decodes one or more repository paths for one authorized source lookup.
type readSourcesRequest struct {
	Paths []string `json:"paths"`
}

// submoduleRequest selects a direct mount and a path within its pinned repository.
type submoduleRequest struct {
	Submodule string `json:"submodule"`
	Path      string `json:"path"`
}

func (r submoduleRequest) validate() error {
	validMount := io.ValidPath(r.Submodule) && r.Submodule != "." && !strings.Contains(r.Submodule, "\\")
	validPath := io.ValidPath(r.Path) && !strings.Contains(r.Path, "\\")

	if !validMount || !validPath {
		return errors.New("review: invalid submodule path")
	}

	return nil
}
