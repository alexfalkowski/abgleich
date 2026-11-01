package workflow

import (
	"fmt"

	"github.com/alexfalkowski/abgleich/internal/github"
	"github.com/alexfalkowski/go-service/v2/encoding/json"
	"github.com/alexfalkowski/go-service/v2/errors"
	"github.com/alexfalkowski/go-service/v2/strings"
	"github.com/bluekeyes/go-gitdiff/gitdiff"
)

type reviewResult struct {
	Overview string          `json:"overview"`
	Findings []reviewFinding `json:"findings"`
}

type reviewFinding struct {
	Body string `json:"body"`
	Path string `json:"path"`
	Side string `json:"side"`
	Line int    `json:"line"`
}

// canPlaceInline reports whether the finding's location is present on the requested side of GitHub's patch.
// Missing patches and uncertain locations keep the finding in the overview.
func (f *reviewFinding) canPlaceInline(files []*github.CommitFile) bool {
	if f.Line <= 0 || (f.Side != "LEFT" && f.Side != "RIGHT") {
		return false
	}

	for _, file := range files {
		if file.GetFilename() == f.Path {
			return patchContainsLine(file.GetPatch(), f.Line, f.Side)
		}
	}
	return false
}

func (f *reviewFinding) location() string {
	if f.Path == "" && f.Line == 0 && f.Side == "" {
		return ""
	}
	return fmt.Sprintf("Location unavailable on the diff: %s:%d (%s).\n", f.Path, f.Line, f.Side)
}

func patchContainsLine(patch string, lineNumber int, side string) bool {
	// GitHub supplies hunks only; the parser requires file headers.
	files, _, err := gitdiff.Parse(strings.NewReader("--- a/file\n+++ b/file\n" + patch))
	if err != nil {
		return false
	}

	line := int64(lineNumber)
	for _, file := range files {
		for _, fragment := range file.TextFragments {
			start, count := fragment.NewPosition, fragment.NewLines
			if side == "LEFT" {
				start, count = fragment.OldPosition, fragment.OldLines
			}
			if line >= start && line < start+count {
				return true
			}
		}
	}
	return false
}

func parseResult(text string) (*reviewResult, error) {
	var result reviewResult
	if err := json.Unmarshal(strings.Bytes(text), &result); err != nil {
		return nil, errors.New("review: invalid result JSON")
	}

	if strings.IsEmpty(strings.TrimSpace(result.Overview)) || result.Findings == nil {
		return nil, errors.New("review: result requires an overview and findings array")
	}

	for _, finding := range result.Findings {
		if strings.IsEmpty(strings.TrimSpace(finding.Body)) {
			return nil, errors.New("review: finding requires a body")
		}
	}

	return &result, nil
}
