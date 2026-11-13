package provider

import (
	"fmt"

	"github.com/alexfalkowski/go-service/v3/errors"
	"github.com/alexfalkowski/go-service/v3/strings"
)

// Range optionally selects inclusive one-based source lines; zero reads the whole file.
type Range struct {
	StartLine int `json:"start_line"`
	EndLine   int `json:"end_line"`
}

// Validate rejects partial or reversed line bounds before source reads.
func (r Range) Validate() error {
	if (r.StartLine == 0 && r.EndLine == 0) || (r.StartLine > 0 && r.EndLine >= r.StartLine) {
		return nil
	}

	return errors.New("review: invalid source line range")
}

// Text renders a complete file or numbered excerpt and records only supplied text.
// An empty excerpt does not record inspection.
func Text(repository Repository, label, content string, lines Range) string {
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

	repository.Inspected(label)

	return content
}
