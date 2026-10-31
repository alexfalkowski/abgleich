package codereview

import (
	"html"
	"regexp"

	"github.com/alexfalkowski/abgleich/internal/ai"
	"github.com/alexfalkowski/go-service/v2/strings"
)

type reviewComment struct {
	Completion *ai.Completion
	HeadSHA    string
	BaseSHA    string
	Marker     string
	RequestID  int64
}

func reply(task *reviewTask, completion *ai.Completion) (string, error) {
	return render("review.md.tmpl", reviewComment{
		Completion: completion,
		HeadSHA:    task.pullRequest.GetHead().GetSHA(),
		BaseSHA:    task.pullRequest.GetBase().GetSHA(),
		Marker:     resultMarker(task),
		RequestID:  task.commandComment.GetID(),
	})
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

// Model text remains inert, with common credential forms redacted.
func markdownText(text string) string {
	for _, pattern := range credentialPatterns {
		text = pattern.ReplaceAllString(text, "[REDACTED]")
	}

	text = bareDomain.ReplaceAllString(text, "www\u200b.")

	return html.EscapeString(strings.NewReplacer(
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
	).Replace(text))
}
