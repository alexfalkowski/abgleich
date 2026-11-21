package workflow

import (
	"github.com/alexfalkowski/abgleich/internal/github"
	"github.com/alexfalkowski/go-service/v3/context"
	"github.com/alexfalkowski/go-service/v3/slices"
	"github.com/alexfalkowski/go-service/v3/strings"
)

const (
	noFindings       = "No findings."
	advisoryFindings = "Advisory feedback."

	// Recognize the assessment labels used for approval-eligible results.
	cleanAssessment       = "\n**Assessment:** " + noFindings + "\n"
	advisoryAssessment    = "\n**Assessment:** " + advisoryFindings + "\n"
	lowSeverityAssessment = "\n**Assessment:** Low-severity findings.\n"
)

func (r *reviewReply) assess(complete bool) {
	r.Event = github.ReviewComment
	switch {
	case len(r.Comments) > 0 || len(r.Findings) > 0:
		r.Event = github.ReviewRequestChanges
		r.Assessment = "Findings."
		r.Decision = "Request changes."
	case !complete:
		r.Assessment = "Review unfinished."
		r.Decision = "Comment. Approval withheld because the reviewer did not finish. See the overview for remaining work."
	case len(r.Missing) > 0:
		r.Assessment = "Review completed with limitations."
		r.Decision = "Comment. Approval withheld because the worker recorded the limitations below."
	case len(r.Advisories) > 0:
		r.Event = github.ReviewApprove
		r.Assessment = advisoryFindings
		r.Decision = "Approve. Feedback is advisory."
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
	// An approval-eligible assessment clears its feature even if another feature blocked approval.
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
	return strings.Contains(body, cleanAssessment) || strings.Contains(body, advisoryAssessment) ||
		strings.Contains(body, lowSeverityAssessment)
}

func (c *Command) hasOtherReviewMarker(review *github.PullRequestReview) bool {
	marker := reviewThreadMarker.FindStringSubmatch(review.GetBody())
	return len(marker) == 2 && marker[1] != c.profile.Name
}
