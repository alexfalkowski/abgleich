package markdown

import (
	"html"
	"regexp"

	"github.com/alexfalkowski/go-service/v3/strings"
)

var bareDomain = regexp.MustCompile(`(?i)\bwww\.`)

var textEscaper = strings.NewReplacer(
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
)

// Escape quotes HTML and Markdown syntax and deactivates GitHub mentions.
// Bare URLs remain available for automatic linking.
func Escape(text string) string {
	return html.EscapeString(textEscaper.Replace(text))
}

// Inert applies Escape and deactivates URL schemes and www links in review text.
func Inert(text string) string {
	text = bareDomain.ReplaceAllString(text, "www\u200b.")
	text = strings.ReplaceAll(text, "://", ":\u200b//")

	return Escape(text)
}
