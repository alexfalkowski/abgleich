package workflow

import (
	"github.com/alexfalkowski/abgleich/internal/github"
	"github.com/alexfalkowski/go-service/v2/context"
	"github.com/alexfalkowski/go-service/v2/slices"
	"github.com/alexfalkowski/go-service/v2/strings"
)

const (
	codeNamespace     = "codereview"
	securityNamespace = "secreview"
)

const (
	noFindings = "No findings."

	// Keep authenticated history checks aligned with the rendered clean assessment.
	cleanAssessment = "\n**Assessment:** " + noFindings + "\n"
)

func (r *reviewReply) assess(result *reviewResult) {
	r.Event = github.ReviewComment
	switch {
	case len(result.Findings) > 0:
		r.Event = github.ReviewRequestChanges
		r.Assessment = "Findings."
		r.Decision = "Request changes."
	case !*result.Complete:
		r.Assessment = "Review unfinished."
		r.Decision = "Comment. Approval withheld because the reviewer did not finish. See the overview for remaining work."
	case len(r.Missing) > 0:
		r.Assessment = "Review completed with limitations."
		r.Decision = "Comment. Approval withheld because the worker recorded the limitations below."
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

	// History is chronological. Incomplete comments do not clear prior findings,
	// including across revisions. A completed clean peer assessment can clear them
	// even when its own approval was withheld to preserve this feature's findings.
	for _, review := range slices.Backward(reviews) {
		switch review.GetState() {
		case github.ReviewChangesRequested:
			return true, nil
		case github.ReviewApproved, github.ReviewDismissed:
			return false, nil
		case github.ReviewCommented:
			if strings.Contains(review.GetBody(), cleanAssessment) {
				return false, nil
			}
		}
	}
	return false, nil
}

func (c *Command) hasOtherReviewMarker(review *github.PullRequestReview) bool {
	marker := reviewThreadMarker.FindStringSubmatch(review.GetBody())
	return len(marker) == 2 && marker[1] != c.profile.Name && (marker[1] == codeNamespace || marker[1] == securityNamespace)
}
