package workflow

import (
	"github.com/alexfalkowski/abgleich/internal/github"
	"github.com/alexfalkowski/go-service/v3/encoding/json"
	"github.com/alexfalkowski/go-service/v3/errors"
	"github.com/alexfalkowski/go-service/v3/strings"
	"github.com/bluekeyes/go-gitdiff/gitdiff"
)

type severity string

const (
	severityP0 severity = "P0"
	severityP1 severity = "P1"
	severityP2 severity = "P2"
	severityP3 severity = "P3"
)

// String returns the severity identifier used in review output.
func (s severity) String() string {
	return string(s)
}

type reviewResult struct {
	Complete *bool           `json:"complete"`
	Overview string          `json:"overview"`
	Findings []reviewFinding `json:"findings"`
}

type reviewFinding struct {
	Title    string          `json:"title"`
	Severity severity        `json:"severity"`
	Trigger  string          `json:"trigger"`
	Impact   string          `json:"impact"`
	Fix      string          `json:"fix"`
	Path     string          `json:"path"`
	Side     github.DiffSide `json:"side"`
	Line     int             `json:"line"`
}

// canPlaceInline reports whether the finding's location is present on the requested side of GitHub's patch.
// Missing patches and uncertain locations keep the finding in the overview.
func (f *reviewFinding) canPlaceInline(files []*github.CommitFile) bool {
	if f.Line <= 0 || (f.Side != github.DiffLeft && f.Side != github.DiffRight) {
		return false
	}

	for _, file := range files {
		if file.GetFilename() == f.Path {
			return patchContainsLine(file.GetPatch(), f.Line, f.Side)
		}
	}
	return false
}

func patchContainsLine(patch string, lineNumber int, side github.DiffSide) bool {
	// GitHub supplies hunks only; the parser requires file headers.
	files, _, err := gitdiff.Parse(strings.NewReader("--- a/file\n+++ b/file\n" + patch))
	if err != nil {
		return false
	}

	line := int64(lineNumber)
	for _, file := range files {
		for _, fragment := range file.TextFragments {
			start, count := fragment.NewPosition, fragment.NewLines
			if side == github.DiffLeft {
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

	if strings.IsEmpty(strings.TrimSpace(result.Overview)) || result.Findings == nil || result.Complete == nil {
		return nil, errors.New("review: result requires an overview, findings array, and completion boolean")
	}

	for _, finding := range result.Findings {
		switch finding.Severity {
		case severityP0, severityP1, severityP2, severityP3:
		default:
			return nil, errors.New("review: finding requires a severity from P0 to P3")
		}
		for _, field := range []string{finding.Title, finding.Trigger, finding.Impact, finding.Fix} {
			if strings.IsEmpty(strings.TrimSpace(field)) {
				return nil, errors.New("review: finding requires a title, trigger, impact, and fix")
			}
		}
	}

	return &result, nil
}
