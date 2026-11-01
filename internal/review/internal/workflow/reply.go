package workflow

import (
	"html"
	"regexp"

	"github.com/alexfalkowski/abgleich/internal/ai"
	"github.com/alexfalkowski/go-service/v2/strings"
)

type reviewComment struct {
	Completion *ai.Completion
	Title      string
	HeadSHA    string
	BaseSHA    string
	Marker     string
	Sources    []string
	Missing    []string
	RequestID  int64
}

var credentialAssignment = regexp.MustCompile(
	`(?i)["']?(?:api[_-]?key|password|secret|token)["']?\s*[=:]\s*("(?:\\.|[^"\\])*"|'[^']*(?:''[^']*)*'|[^\s"',;{}]+)`,
)

var credentialPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?s)-----BEGIN [A-Z ]*PRIVATE KEY-----.*?-----END [A-Z ]*PRIVATE KEY-----`),
	regexp.MustCompile(`\b(?:gh[pousr]_[A-Za-z0-9_]+|github_pat_[A-Za-z0-9_]+|sk-[A-Za-z0-9_-]+)\b`),
	regexp.MustCompile(`(?i)\b(?:bearer|basic)\s+[A-Za-z0-9._~+/=-]+`),
	credentialAssignment,
}

var bareDomain = regexp.MustCompile(`(?i)\bwww\.`)

var markdownEscaper = strings.NewReplacer(
	"@", "@\u200b",
	"://", ":\u200b//",
	"\\", "\\\\",
	"`", "\\`",
	"*", "\\*",
	"_", "\\_",
	"[", "\\[",
	"]", "\\]",
	"(", "\\(",
	")", "\\)",
	"#", "\\#",
	"!", "\\!",
	"|", "\\|",
	"~", "\\~",
	"-", "\\-",
	"+", "\\+",
	"=", "\\=",
)

func (c *Command) reply(task *reviewTask, completion *ai.Completion, source *sourceSession) (string, error) {
	return render("review.md.tmpl", reviewComment{
		Completion: completion,
		Title:      c.definition.Title,
		Missing:    source.missing,
		Sources:    source.inspected,
		HeadSHA:    task.pullRequest.GetHead().GetSHA(),
		BaseSHA:    task.pullRequest.GetBase().GetSHA(),
		Marker:     c.resultMarker(task),
		RequestID:  task.commandComment.GetID(),
	})
}

// markdownText makes model text inert and redacts common credential forms.
// Provider-known credentials are redacted by ai.Provider before rendering.
func markdownText(text string) string {
	for _, pattern := range credentialPatterns {
		text = pattern.ReplaceAllString(text, "[REDACTED]")
	}

	text = bareDomain.ReplaceAllString(text, "www\u200b.")
	return html.EscapeString(markdownEscaper.Replace(text))
}
