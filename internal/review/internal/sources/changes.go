package sources

import (
	"fmt"

	"github.com/alexfalkowski/abgleich/internal/github"
	"github.com/alexfalkowski/go-service/v3/context"
	"github.com/alexfalkowski/go-service/v3/slices"
	"github.com/alexfalkowski/go-service/v3/strings"
)

// ReadChanges reads required changed-file text and records missing patches.
// ReadDesign collects accepted-design text separately. Deleted files are read at the base revision.
func (s *Session) ReadChanges(ctx context.Context, files []*github.CommitFile) (string, error) {
	var headFiles, baseFiles []string
	for _, file := range files {
		if isAcceptedDesignPath(file.GetFilename()) {
			continue
		}
		if file.GetStatus() == github.FileRemoved {
			baseFiles = append(baseFiles, file.GetFilename())
		} else {
			headFiles = append(headFiles, file.GetFilename())
		}
	}

	headSource, err := s.repositoryFiles(ctx, s.headSHA, headFiles)
	if err != nil {
		return strings.Empty, err
	}

	baseSource, err := s.repositoryFiles(ctx, s.baseSHA, baseFiles)
	if err != nil {
		return strings.Empty, err
	}

	return s.renderChangedSource(files, headSource, baseSource), nil
}

func (s *Session) renderChangedSource(files []*github.CommitFile, headSource, baseSource map[string]string) string {
	var source strings.Builder
	for _, file := range files {
		var revision string
		var contents map[string]string
		if file.GetStatus() == github.FileRemoved {
			revision = s.baseSHA
			contents = baseSource
		} else {
			revision = s.headSHA
			contents = headSource
		}

		if strings.IsEmpty(file.GetPatch()) && !slices.Contains(s.binary, strings.Concat(revision, ":", file.GetFilename())) {
			s.unavailable(strings.Concat("patch for ", file.GetFilename()))
		}
		if isAcceptedDesignPath(file.GetFilename()) {
			continue
		}

		fmt.Fprintf(&source, "\nSource: %s at %s\n", file.GetFilename(), revision)
		source.WriteString(contents[file.GetFilename()])
		source.WriteString("\n")
	}

	return source.String()
}
