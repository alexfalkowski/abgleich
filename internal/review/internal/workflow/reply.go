package workflow

import (
	"fmt"

	"github.com/alexfalkowski/abgleich/internal/ai"
	"github.com/alexfalkowski/abgleich/internal/github"
	"github.com/alexfalkowski/abgleich/internal/markdown"
	"github.com/alexfalkowski/go-service/v2/strings"
)

type reviewReply struct {
	Completion *ai.Completion
	Title      string
	HeadSHA    string
	BaseSHA    string
	Marker     string
	Assessment string
	Decision   string
	Event      string
	Comments   []*github.DraftReviewComment
	Sources    []string
	Missing    []string
	RequestID  int64
}

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

	return fmt.Sprintf("⏹️ %s canceled because %s. Request: [original request](#issuecomment-%d).", c.definition.Title, reason, task.commandComment.GetID())
}

func (c *Command) reply(task *reviewTask, completion *ai.Completion, review *reviewContext) (*reviewReply, error) {
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
				Body: new(markdown.Inert(finding.Body)),
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

	return &reviewReply{
		Completion: &output,
		Title:      c.definition.Title,
		Missing:    source.missing,
		Sources:    source.inspected,
		HeadSHA:    task.pullRequest.GetHead().GetSHA(),
		BaseSHA:    task.pullRequest.GetBase().GetSHA(),
		Marker:     marker,
		Assessment: assessment,
		Decision:   decision,
		Event:      event,
		Comments:   comments,
		RequestID:  task.commandComment.GetID(),
	}, nil
}
