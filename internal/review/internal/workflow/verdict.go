package workflow

import (
	"github.com/alexfalkowski/abgleich/internal/github"
	"github.com/alexfalkowski/go-service/v3/context"
	"github.com/alexfalkowski/go-service/v3/slices"
	"github.com/alexfalkowski/go-service/v3/strings"
)

const (
	noFindings          = "No findings."
	lowSeverityFindings = "Low-severity findings."

	// Keep authenticated history checks aligned with rendered approval-eligible assessments.
	cleanAssessment       = "\n**Assessment:** " + noFindings + "\n"
	lowSeverityAssessment = "\n**Assessment:** " + lowSeverityFindings + "\n"
)

func (r *reviewReply) assess(result *reviewResult) {
	requiresChanges := slices.ContainsFunc(result.Findings, func(finding reviewFinding) bool {
		return finding.Severity != severityP3
	})
	r.Event = github.ReviewComment
	switch {
	case requiresChanges:
		r.Event = github.ReviewRequestChanges
		r.Assessment = "Findings."
		r.Decision = "Request changes."
	case !*result.Complete:
		r.Assessment = "Review unfinished."
		r.Decision = "Comment. Approval withheld because the reviewer did not finish. See the overview for remaining work."
	case len(r.Missing) > 0:
		r.Assessment = "Review completed with limitations."
		r.Decision = "Comment. Approval withheld because the worker recorded the limitations below."
	case len(result.Findings) > 0:
		r.Event = github.ReviewApprove
		r.Assessment = lowSeverityFindings
		r.Decision = "Approve. Only low-severity findings."
	default:
		r.Event = github.ReviewApprove
		r.Assessment = noFindings
		r.Decision = "Approve."
	}
}

func (c *Command) otherReviewRequestsChanges(ctx context.Context, task *reviewTask) (bool, error) {
	reviews, err := c.github.ListReviews(ctx, task.repository, task.request)
	if err != nil {
		return false, err
	}

	var candidates []*github.PullRequestReview
	for _, review := range reviews {
		if c.hasOtherReviewMarker(review) {
			candidates = append(candidates, review)
		}
	}

	reviews, err = c.github.AppReviews(ctx, candidates)
	if err != nil {
		return false, err
	}

	// Resolve each feature independently, including features no longer enabled.
	// Incomplete comments do not clear prior findings, including across revisions.
	// A complete assessment with at most P3 findings clears its feature even if another feature blocks approval.
	assessed := make(map[string]bool)
	for _, review := range slices.Backward(reviews) {
		name := reviewThreadMarker.FindStringSubmatch(review.GetBody())[1]
		if assessed[name] {
			continue
		}

		switch github.ReviewState(review.GetState()) {
		case github.ReviewChangesRequested:
			return true, nil
		case github.ReviewApproved, github.ReviewDismissed:
			assessed[name] = true
		case github.ReviewCommented:
			assessed[name] = isApprovalAssessment(review.GetBody())
		}
	}
	return false, nil
}

func isApprovalAssessment(body string) bool {
	return strings.Contains(body, cleanAssessment) || strings.Contains(body, lowSeverityAssessment)
}

func (c *Command) hasOtherReviewMarker(review *github.PullRequestReview) bool {
	marker := reviewThreadMarker.FindStringSubmatch(review.GetBody())
	return len(marker) == 2 && marker[1] != c.profile.Name
}
