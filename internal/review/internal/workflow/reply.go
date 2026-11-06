package workflow

import (
	"fmt"

	"github.com/alexfalkowski/abgleich/internal/ai"
	"github.com/alexfalkowski/abgleich/internal/github"
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
	Findings   []reviewFinding
	Sources    []string
	Missing    []string
	Binary     []string
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

	var comments []*github.DraftReviewComment
	var findings []reviewFinding
	for _, finding := range result.Findings {
		if finding.canPlaceInline(review.Files) {
			body, err := render("finding.md.tmpl", finding)
			if err != nil {
				return nil, err
			}

			comments = append(comments, &github.DraftReviewComment{
				Path: new(finding.Path), Line: new(finding.Line), Side: new(finding.Side),
				Body: new(body),
			})

			continue
		}

		findings = append(findings, finding)
	}

	// Preserve provider provenance while rendering only the decoded overview.
	output := *completion
	output.Text = result.Overview
	missing := review.source.missing
	if len(review.Files) == 0 {
		missing = append(missing, "GitHub returned no changed files.")
	}
	marker := strings.Empty
	event, assessment, decision := "COMMENT", "Review unfinished.", "Comment. Approval withheld because the reviewer did not finish. See the overview for remaining work."
	switch {
	case len(result.Findings) > 0:
		event, assessment, decision = "REQUEST_CHANGES", "Findings.", "Request changes."
		marker = c.resultMarker(task)
	case *result.Complete && len(missing) == 0:
		event, assessment, decision = "APPROVE", "No findings.", "Approve."
		marker = c.resultMarker(task)
	case *result.Complete:
		assessment, decision = "Review completed with limitations.", "Comment. Approval withheld because the worker recorded the limitations below."
	}

	return &reviewReply{
		Completion: &output,
		Title:      c.definition.Title,
		Missing:    missing,
		Sources:    review.source.inspected,
		Binary:     review.source.binary,
		HeadSHA:    task.pullRequest.GetHead().GetSHA(),
		BaseSHA:    task.pullRequest.GetBase().GetSHA(),
		Marker:     marker,
		Assessment: assessment,
		Decision:   decision,
		Event:      event,
		Comments:   comments,
		Findings:   findings,
		RequestID:  task.commandComment.GetID(),
	}, nil
}
