package workflow

import (
	"github.com/alexfalkowski/abgleich/internal/ai"
	"github.com/alexfalkowski/abgleich/internal/github"
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
	External   []string
	Binary     []string
	RequestID  int64
}

func (c *Command) reply(task *reviewTask, completion *ai.Completion, review *reviewContext) (*reviewReply, error) {
	result, err := parseResult(completion.Text)
	if err != nil {
		return nil, err
	}

	// Preserve provider provenance while rendering only the decoded overview.
	output := *completion
	output.Text = result.Overview
	reply := &reviewReply{
		Completion: &output,
		Title:      c.profile.Title,
		Missing:    review.source.missing,
		Sources:    review.source.inspected,
		External:   review.source.external,
		Binary:     review.source.binary,
		HeadSHA:    task.request.GetHead().GetSHA(),
		BaseSHA:    task.request.GetBase().GetSHA(),
		RequestID:  task.comment.GetID(),
	}
	if len(review.Files) == 0 {
		reply.Missing = append(reply.Missing, "GitHub returned no changed files.")
	}

	for _, finding := range result.Findings {
		if !finding.canPlaceInline(review.Files) {
			reply.Findings = append(reply.Findings, finding)
			continue
		}

		body, err := render("finding.md.tmpl", finding)
		if err != nil {
			return nil, err
		}

		reply.Comments = append(reply.Comments, &github.DraftReviewComment{
			Path: new(finding.Path), Line: new(finding.Line), Side: new(finding.Side),
			Body: new(body),
		})
	}

	reply.assess(result)

	if reply.Event != github.ReviewComment {
		reply.Marker = c.resultMarker(task)
	}

	return reply, nil
}
