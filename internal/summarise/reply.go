package summarise

import (
	"html"
	"strings"
)

func reply(task *summaryTask, text string) (string, error) {
	return render("summary.md.tmpl", summaryComment{
		HeadSHA:          task.pullRequest.GetHead().GetSHA(),
		CommandCommentID: task.commandComment.GetID(),
		Text:             text,
	})
}

// Markdown syntax and GitHub mentions need escaping in addition to HTML text.
func markdownText(text string) string {
	return html.EscapeString(strings.NewReplacer(
		"@", "@\u200b",
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
