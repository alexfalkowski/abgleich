package workflow

import (
	"fmt"

	"github.com/alexfalkowski/go-service/v2/errors"
	"github.com/alexfalkowski/go-service/v2/slices"
	"github.com/alexfalkowski/go-service/v2/strings"
)

func (r sourceRange) validate() error {
	if (r.StartLine == 0 && r.EndLine == 0) || (r.StartLine > 0 && r.EndLine >= r.StartLine) {
		return nil
	}

	return errors.New("review: invalid source line range")
}

// sourceText records only delivered text, rather than the whole downloaded file.
func (s *sourceSession) sourceText(label, content string, lines sourceRange) string {
	if lines.StartLine > 0 {
		var excerpt strings.Builder
		last, number := 0, 0
		input := strings.TrimSuffix(content, "\n")

		for line := range strings.SplitSeq(input, "\n") {
			number++

			if number > lines.EndLine || strings.IsEmpty(content) {
				break
			}

			if number >= lines.StartLine {
				fmt.Fprintf(&excerpt, "%d: %s\n", number, line)
				last = number
			}
		}

		if last == 0 {
			return "No lines in the requested range."
		}

		content = fmt.Sprintf("Excerpt lines %d-%d:\n%s", lines.StartLine, last, excerpt.String())
		label += fmt.Sprintf(" (lines %d-%d)", lines.StartLine, last)
	}

	if !slices.Contains(s.inspected, label) {
		s.inspected = append(s.inspected, label)
	}

	return content
}
