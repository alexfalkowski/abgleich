package workflow

import (
	"github.com/alexfalkowski/abgleich/internal/github"
	"github.com/alexfalkowski/go-service/v3/encoding/json"
	"github.com/alexfalkowski/go-service/v3/errors"
	"github.com/alexfalkowski/go-service/v3/slices"
	"github.com/alexfalkowski/go-service/v3/strings"
	"github.com/bluekeyes/go-gitdiff/gitdiff"
)

type severity string

// String returns the severity identifier used in review output.
func (s severity) String() string {
	return string(s)
}

const (
	severityP0 severity = "P0"
	severityP1 severity = "P1"
	severityP2 severity = "P2"
	severityP3 severity = "P3"
)

type findingKind string

const (
	findingDefect           findingKind = "defect"
	findingTestRegression   findingKind = "test_regression"
	findingRequiredCoverage findingKind = "required_coverage"
	findingCoverage         findingKind = "coverage"
)

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
	Kind     findingKind     `json:"kind"`
	Evidence string          `json:"evidence"`
	Line     int             `json:"line"`
}

func (f *reviewFinding) canBlock(review *reviewContext) bool {
	if f.Severity == severityP3 {
		return false
	}

	switch f.Kind {
	case findingDefect:
		return true
	case findingTestRegression:
		return strings.TrimSpace(f.Evidence) != "" && f.onPatch(review.Files, true)
	case findingRequiredCoverage:
		return slices.Contains(review.requiredChecks, strings.TrimSpace(f.Evidence))
	case findingCoverage:
		return false
	}
	return false
}

// canPlaceInline reports whether the finding's location is present on the requested side of GitHub's patch.
// Missing patches and uncertain locations keep the finding in the overview.
func (f *reviewFinding) canPlaceInline(files []*github.CommitFile) bool {
	return f.onPatch(files, false)
}

func (f *reviewFinding) onPatch(files []*github.CommitFile, changed bool) bool {
	if f.Line <= 0 || (f.Side != github.DiffLeft && f.Side != github.DiffRight) {
		return false
	}

	for _, file := range files {
		if file.GetFilename() == f.Path {
			return patchContainsLine(file.GetPatch(), f.Line, f.Side, changed)
		}
	}
	return false
}

func patchContainsLine(patch string, lineNumber int, side github.DiffSide, changed bool) bool {
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
				return !changed || changedPatchLine(fragment.Lines, line-start, side)
			}
		}
	}
	return false
}

func changedPatchLine(lines []gitdiff.Line, offset int64, side github.DiffSide) bool {
	for _, line := range lines {
		onSide := (side == github.DiffLeft && line.Old()) || (side == github.DiffRight && line.New())
		if onSide {
			if offset == 0 {
				return line.Op != gitdiff.OpContext
			}
			offset--
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
		if err := finding.validate(); err != nil {
			return nil, err
		}
	}

	return &result, nil
}

func (f *reviewFinding) validate() error {
	switch f.Kind {
	case findingDefect, findingTestRegression, findingRequiredCoverage, findingCoverage:
	default:
		return errors.New("review: finding requires a recognized kind")
	}
	switch f.Severity {
	case severityP0, severityP1, severityP2, severityP3:
	default:
		return errors.New("review: finding requires a severity from P0 to P3")
	}

	for _, field := range []string{f.Title, f.Trigger, f.Impact, f.Fix} {
		if strings.IsEmpty(strings.TrimSpace(field)) {
			return errors.New("review: finding requires a title, trigger, impact, and fix")
		}
	}
	return nil
}
