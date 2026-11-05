package workflow

import (
	"fmt"
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
	Assessment string
	Decision   string
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

func (c *Command) cancellationMessage(task *reviewTask) string {
	pull := task.pullRequest

	var reason string
	switch {
	case pull.GetState() == "closed" && pull.GetMerged():
		reason = "the pull request was merged"
	case pull.GetState() == "closed":
		reason = "the pull request was closed"
	case pull.GetState() == "open" && pull.GetDraft():
		reason = "the pull request became a draft"
	default:
		return strings.Empty
	}

	return fmt.Sprintf("%s canceled because %s. Request: [original request](#issuecomment-%d).", c.definition.Title, reason, task.commandComment.GetID())
}

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
	marker := c.resultMarker(task)

	event, assessment, decision := "COMMENT", "Incomplete review.", "Comment. Review completion was not confirmed or source was unavailable."
	switch {
	case len(result.Findings) > 0:
		event, assessment, decision = "REQUEST_CHANGES", "Findings.", "Request changes."
	case result.Complete && len(source.missing) == 0 && len(review.Files) > 0:
		event, assessment, decision = "APPROVE", "No findings.", "Approve."
	default:
		marker = strings.Empty
	}

	body, err := render("review.md.tmpl", reviewComment{
		Completion: &output,
		Title:      c.definition.Title,
		Missing:    source.missing,
		Sources:    source.inspected,
		HeadSHA:    task.pullRequest.GetHead().GetSHA(),
		BaseSHA:    task.pullRequest.GetBase().GetSHA(),
		Marker:     marker,
		Assessment: assessment,
		Decision:   decision,
		RequestID:  task.commandComment.GetID(),
	})
	if err != nil {
		return nil, err
	}

	return &github.PullRequestReviewRequest{
		Body: new(body), Comments: comments, Event: new(event),
	}, nil
}

// markdownText makes model markup, links, and mentions inert.
func markdownText(text string) string {
	text = bareDomain.ReplaceAllString(text, "www\u200b.")
	return html.EscapeString(markdownEscaper.Replace(text))
}
