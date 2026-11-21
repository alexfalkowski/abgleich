package assessment

import (
	"fmt"

	"github.com/alexfalkowski/abgleich/internal/ai"
	"github.com/alexfalkowski/abgleich/internal/github"
	"github.com/alexfalkowski/abgleich/internal/review/internal/history"
	"github.com/alexfalkowski/abgleich/internal/review/internal/sources"
	"github.com/alexfalkowski/go-service/v3/strings"
)

type reviewReply struct {
	Snapshot       *history.Snapshot
	Completion     *ai.Completion
	Title          string
	HeadSHA        string
	BaseSHA        string
	Marker         string
	Assessment     string
	Decision       string
	Event          github.ReviewEvent
	Comments       []*github.DraftReviewComment
	InlineFindings []reviewFinding
	Findings       []reviewFinding
	LowSeverity    []reviewFinding
	Cleared        []clearedFinding
	Unverified     []string
	Sources        []string
	Missing        []string
	External       []string
	Binary         []string
	Comparison     []string
	RequestID      int64
}

// Input contains the evidence and provenance used to build one review.
type Input struct {
	// Source records which text the worker could supply to the model.
	Source sources.Report

	// Completion includes the model response and its actual provider usage.
	Completion *ai.Completion

	// Followup contains previously authenticated blockers, if any.
	Followup *history.Followup

	// Patches fingerprints the patches used in this assessment.
	Patches map[string]string

	// Title labels the review feature in GitHub.
	Title string

	// HeadSHA identifies the revision on which findings are placed.
	HeadSHA string

	// BaseSHA identifies the compared target revision.
	BaseSHA string

	// Marker identifies a completed result for these revisions and this feature.
	Marker string

	// Files contains the complete changed-file listing used for inline placement.
	Files []*github.CommitFile

	// RequestID identifies the maintainer comment that authorized the review.
	RequestID int64
}

// Draft holds a review verdict and the data needed to render the native request.
type Draft struct {
	reply reviewReply
}

// NewDraft validates the result, reconciles prior findings, and builds a draft
// using the model response and independently collected source evidence.
func NewDraft(input Input) (*Draft, error) {
	result, err := parseResult(input.Completion.Text)
	if err != nil {
		return nil, err
	}

	// Preserve provider provenance while rendering only the decoded overview.
	output := *input.Completion
	output.Text = result.Overview
	draft := &Draft{reply: reviewReply{
		Completion: &output,
		Title:      input.Title,
		Missing:    input.Source.Missing,
		Sources:    input.Source.Inspected,
		External:   input.Source.External,
		Binary:     input.Source.Binary,
		HeadSHA:    input.HeadSHA,
		BaseSHA:    input.BaseSHA,
		RequestID:  input.RequestID,
	}}
	reply := &draft.reply
	if len(input.Files) == 0 {
		reply.Missing = append(reply.Missing, "GitHub returned no changed files.")
	}

	if input.Followup != nil {
		reply.Comparison = input.Followup.ComparisonUnavailable
	}
	reply.reconcileFindings(result, input.Followup, fmt.Sprintf("%s\n%d", input.Marker, reply.RequestID))
	if err := reply.placeFindings(result.Findings, input.Files); err != nil {
		return nil, err
	}

	reply.assess(result)

	if reply.Event != github.ReviewComment {
		partial := !*result.Complete || len(reply.Missing) > 0
		reply.Marker = input.Marker
		if partial {
			reply.Marker = history.PartialMarker(reply.Marker)
		}
		reply.Snapshot = input.snapshot(result, partial)
	}

	return draft, nil
}

// Approves reports whether the draft would approve the changes.
func (d *Draft) Approves() bool {
	return d.reply.Event == github.ReviewApprove
}

// WithholdApproval changes the draft's event from approval to comment when
// another feature still requests changes. The assessment remains reusable.
func (d *Draft) WithholdApproval() {
	d.reply.Event = github.ReviewComment
	d.reply.Decision = "Comment. Approval withheld because the other review still requests changes."
}

// Request renders the finalized draft as a native GitHub review request.
func (d *Draft) Request() (*github.PullRequestReviewRequest, error) {
	body, err := d.reply.body()
	if err != nil {
		return nil, err
	}
	return &github.PullRequestReviewRequest{
		Body: new(body), Comments: d.reply.Comments, Event: new(d.reply.Event.String()),
	}, nil
}

func (r *reviewReply) body() (string, error) {
	body, err := render("review.md.tmpl", r)
	if err != nil || r.Snapshot == nil {
		return body, err
	}

	// Keep the blockers when fingerprints would exceed the review body budget.
	const maxReviewWithSnapshot = 60_000
	snapshot, err := r.Snapshot.Encode()
	if err != nil {
		return "", err
	}
	if len(body)+len(snapshot)+1 > maxReviewWithSnapshot {
		compact := *r.Snapshot
		compact.Patches, compact.Sources = nil, nil
		snapshot, err = compact.Encode()
		if err != nil {
			return "", err
		}
	}
	if len(body)+len(snapshot)+1 <= maxReviewWithSnapshot {
		return strings.Concat(body, snapshot, "\n"), nil
	}
	if r.Event == github.ReviewRequestChanges && !strings.Contains(r.Marker, ":v2:partial:") {
		// A retry must recognize that a blocker record was expected.
		marker := history.PartialMarker(r.Marker)
		body = strings.ReplaceAll(body, r.Marker, marker)
	}
	return body, nil
}

func (r *reviewReply) placeFindings(findings []reviewFinding, files []*github.CommitFile) error {
	for index := range findings {
		finding := &findings[index]
		if finding.Severity == severityP3 {
			r.LowSeverity = append(r.LowSeverity, *finding)
			continue
		}
		if finding.ReviewID > 0 {
			r.Findings = append(r.Findings, *finding)
			continue
		}

		body, err := render("finding.md.tmpl", finding)
		if err != nil {
			return err
		}

		inline := finding.canPlaceInline(files)
		finding.reference = history.Reference{ID: finding.ID, Digest: history.FindingDigest(body), Inline: inline}
		if !inline {
			r.Findings = append(r.Findings, *finding)
			continue
		}

		r.InlineFindings = append(r.InlineFindings, *finding)
		r.Comments = append(r.Comments, &github.DraftReviewComment{
			Path: new(finding.Path), Line: new(finding.Line), Side: new(finding.Side.String()),
			Body: new(body),
		})
	}
	return nil
}
