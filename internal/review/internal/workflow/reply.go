package workflow

import (
	"fmt"

	"github.com/alexfalkowski/abgleich/internal/ai"
	"github.com/alexfalkowski/abgleich/internal/github"
	"github.com/alexfalkowski/go-service/v3/strings"
)

type reviewReply struct {
	Snapshot   *reviewSnapshot
	Completion *ai.Completion
	Title      string
	HeadSHA    string
	BaseSHA    string
	Marker     string
	Assessment string
	Decision   string
	Event      github.ReviewEvent
	Comments   []*github.DraftReviewComment
	Findings   []reviewFinding
	Cleared    []clearedFinding
	Unverified []string
	Sources    []string
	Missing    []string
	External   []string
	Binary     []string
	Comparison []string
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

	marker := c.resultMarker(task)
	if review.followup != nil {
		reply.Comparison = review.followup.ComparisonUnavailable
	}
	reply.reconcileFindings(result, review.followup, fmt.Sprintf("%s\n%d", marker, reply.RequestID))
	if err := reply.placeFindings(result.Findings, review.Files); err != nil {
		return nil, err
	}

	reply.assess(result)

	if reply.Event != github.ReviewComment {
		partial := !*result.Complete || len(reply.Missing) > 0
		reply.Marker = c.resultMarkerAt(task, reply.HeadSHA, reply.BaseSHA, partial)
		reply.Snapshot = review.snapshot(result, partial)
	}

	return reply, nil
}

func (r *reviewReply) body() (string, error) {
	body, err := render("review.md.tmpl", r)
	if err != nil || r.Snapshot == nil {
		return body, err
	}

	// Keep the blockers when fingerprints would exceed the review body budget.
	const maxReviewWithSnapshot = 60_000
	snapshot, err := r.Snapshot.encode()
	if err != nil {
		return "", err
	}
	if len(body)+len(snapshot)+1 > maxReviewWithSnapshot {
		compact := *r.Snapshot
		compact.Patches, compact.Sources = nil, nil
		snapshot, err = compact.encode()
		if err != nil {
			return "", err
		}
	}
	if len(body)+len(snapshot)+1 <= maxReviewWithSnapshot {
		return strings.Concat(body, snapshot, "\n"), nil
	}
	if r.Event == github.ReviewRequestChanges && !strings.Contains(r.Marker, ":v2:partial:") {
		// A retry must recognize that a blocker record was expected.
		marker := strings.ReplaceAll(r.Marker, ":v2:", ":v2:partial:")
		body = strings.ReplaceAll(body, r.Marker, marker)
	}
	return body, nil
}

func (r *reviewReply) placeFindings(findings []reviewFinding, files []*github.CommitFile) error {
	for index := range findings {
		finding := &findings[index]
		if finding.ReviewID > 0 || finding.Severity == severityP3 {
			r.Findings = append(r.Findings, *finding)
			continue
		}

		body, err := render("finding.md.tmpl", finding)
		if err != nil {
			return err
		}

		inline := finding.canPlaceInline(files)
		finding.reference = blockerReference{ID: finding.ID, Digest: findingDigest(body), Inline: inline}
		if !inline {
			r.Findings = append(r.Findings, *finding)
			continue
		}

		r.Comments = append(r.Comments, &github.DraftReviewComment{
			Path: new(finding.Path), Line: new(finding.Line), Side: new(finding.Side.String()),
			Body: new(body),
		})
	}
	return nil
}
