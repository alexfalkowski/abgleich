package workflow

import (
	"html"
	"regexp"

	"github.com/alexfalkowski/abgleich/internal/ai"
	"github.com/alexfalkowski/abgleich/internal/github"
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

func (c *Command) reply(task *reviewTask, completion *ai.Completion, review *reviewContext) (*github.PullRequestReviewRequest, error) {
	result, err := parseResult(completion.Text)
	if err != nil {
		return nil, err
	}

	var text strings.Builder
	text.WriteString(result.Overview)
	var comments []*github.DraftReviewComment
	for _, finding := range result.Findings {
		if finding.canPlaceInline(review.Files) {
			comments = append(comments, &github.DraftReviewComment{
				Path: new(finding.Path), Line: new(finding.Line), Side: new(finding.Side),
				Body: new(markdownText(finding.Body)),
			})
			continue
		}

		text.WriteString("\n\n")
		text.WriteString(finding.location())
		text.WriteString(finding.Body)
	}

	// Preserve provider provenance while rendering only the decoded overview.
	output := *completion
	output.Text = text.String()
	source := review.source
	body, err := render("review.md.tmpl", reviewComment{
		Completion: &output,
		Title:      c.definition.Title,
		Missing:    source.missing,
		Sources:    source.inspected,
		HeadSHA:    task.pullRequest.GetHead().GetSHA(),
		BaseSHA:    task.pullRequest.GetBase().GetSHA(),
		Marker:     c.resultMarker(task),
		RequestID:  task.commandComment.GetID(),
	})
	if err != nil {
		return nil, err
	}

	return &github.PullRequestReviewRequest{
		Body: new(body), Comments: comments,
	}, nil
}

// markdownText makes model markup, links, and mentions inert.
func markdownText(text string) string {
	text = bareDomain.ReplaceAllString(text, "www\u200b.")
	return html.EscapeString(markdownEscaper.Replace(text))
}
