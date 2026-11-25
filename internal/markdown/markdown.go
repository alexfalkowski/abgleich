package markdown

import (
	"html"
	"regexp"

	"github.com/alexfalkowski/go-service/v3/strings"
)

var bareDomain = regexp.MustCompile(`(?i)\bwww\.`)

var bareURL = regexp.MustCompile(`(?i)(^|[[:space:]*_~(])((?:https?://|www\.)[^[:space:]<>]+)`)

var leadingIndent = regexp.MustCompile(`(?m)^[ \t]`)

var trailingEntity = regexp.MustCompile(`&[[:alnum:]]+;$`)

var textEscaper = strings.NewReplacer(
	"\r\n", "\n",
	"\r", "\n",
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

// Escape quotes HTML and Markdown syntax, isolates dollars from GitHub math,
// and deactivates GitHub mentions.
// Leading indentation stays plain text instead of forming code blocks.
// Bare URLs remain available for automatic linking.
// Dollar-bearing HTTP(S) and www URLs use explicit links to preserve destinations.
func Escape(text string) string {
	var escaped strings.Builder
	last := 0
	for _, match := range bareURL.FindAllStringSubmatchIndex(text, -1) {
		start := match[4]
		target := trimURLPunctuation(text[start:match[5]])
		if !strings.Contains(target, "$") {
			continue
		}

		escaped.WriteString(escape(text[last:start]))
		escaped.WriteString("[")
		escaped.WriteString(escape(target))
		escaped.WriteString("](<")
		if strings.HasPrefix(strings.ToLower(target), "www.") {
			escaped.WriteString("http://")
		}
		escaped.WriteString(html.EscapeString(target))
		escaped.WriteString(">)")
		last = start + len(target)
	}
	escaped.WriteString(escape(text[last:]))

	return escaped.String()
}

// Inert applies Escape and deactivates URL schemes and www links in review text.
func Inert(text string) string {
	text = bareDomain.ReplaceAllString(text, "www\u200b.")
	text = strings.ReplaceAll(text, "://", ":\u200b//")

	return Escape(text)
}

func escape(text string) string {
	text = html.EscapeString(textEscaper.Replace(text))
	// Code blocks would display the dollar-isolating spans verbatim.
	text = leadingIndent.ReplaceAllStringFunc(text, func(indent string) string {
		if indent == "\t" {
			return "&#9;"
		}

		return "&#32;"
	})
	// GitHub recognizes dollar pairs even when Markdown-escaped.
	return strings.ReplaceAll(text, "$", "<span>$</span>")
}

// Match GFM autolink boundaries before generating a destination, keeping sentence
// punctuation and unmatched closing parentheses outside the link.
// https://github.github.com/gfm/#autolinks-extension-
func trimURLPunctuation(text string) string {
	text = strings.TrimRight(text, "?!.,:*_~")
	excess := strings.Count(text, ")") - strings.Count(text, "(")
	for excess > 0 && strings.HasSuffix(text, ")") {
		text = strings.TrimSuffix(text, ")")
		excess--
	}

	return trailingEntity.ReplaceAllString(text, strings.Empty)
}
