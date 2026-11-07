package workflow

import (
	"fmt"

	"github.com/alexfalkowski/abgleich/internal/ai"
	"github.com/alexfalkowski/abgleich/internal/github"
	"github.com/alexfalkowski/go-service/v2/strings"
)

const (
	noFindings = "No findings."

	// Keep authenticated history checks aligned with the rendered clean assessment.
	cleanAssessment = "\n**Assessment:** " + noFindings + "\n"
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

func (c *Command) staleMessage() string {
	return strings.Concat("🔄 The pull request revision changed. Post a new /ab ", c.profile.Name, " request.")
}

func (c *Command) cancellationMessage(task *reviewTask) string {
	pull := task.request

	var reason string
	switch {
	case pull.GetState() == github.PullClosed && pull.GetMerged():
		reason = "the pull request was merged"
	case pull.GetState() == github.PullClosed:
		reason = "the pull request was closed"
	case pull.GetState() == github.PullOpen && pull.GetDraft():
		reason = "the pull request became a draft"
	default:
		return strings.Empty
	}

	return fmt.Sprintf("⏹️ %s canceled because %s. Request: [original request](#issuecomment-%d).", c.profile.Title, reason, task.comment.GetID())
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
	event, assessment, decision := github.ReviewComment, "Review unfinished.", "Comment. Approval withheld because the reviewer did not finish. See the overview for remaining work."
	switch {
	case len(result.Findings) > 0:
		event, assessment, decision = github.ReviewRequestChanges, "Findings.", "Request changes."
		marker = c.resultMarker(task)
	case *result.Complete && len(missing) == 0:
		event, assessment, decision = github.ReviewApprove, noFindings, "Approve."
		marker = c.resultMarker(task)
	case *result.Complete:
		assessment, decision = "Review completed with limitations.", "Comment. Approval withheld because the worker recorded the limitations below."
	}

	return &reviewReply{
		Completion: &output,
		Title:      c.profile.Title,
		Missing:    missing,
		Sources:    review.source.inspected,
		Binary:     review.source.binary,
		HeadSHA:    task.request.GetHead().GetSHA(),
		BaseSHA:    task.request.GetBase().GetSHA(),
		Marker:     marker,
		Assessment: assessment,
		Decision:   decision,
		Event:      event,
		Comments:   comments,
		Findings:   findings,
		RequestID:  task.comment.GetID(),
	}, nil
}
