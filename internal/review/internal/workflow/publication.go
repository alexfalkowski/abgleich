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

func (c *Command) submitReview(ctx context.Context, task *reviewTask, reply *reviewReply) error {
	task, err := c.recheck(ctx, task)
	if err != nil || task == nil {
		return err
	}

	if reply.Event == github.ReviewApprove {
		blocked, err := c.otherReviewRequestsChanges(ctx, task)
		if err != nil {
			return err
		}
		if blocked {
			reply.Event = github.ReviewComment
			reply.Decision = "Comment. Approval withheld because the other review still requests changes."
		}
	}

	body, err := render("review.md.tmpl", reply)
	if err != nil {
		return c.reportFailure(ctx, task, stageResult, err)
	}

	if err := ctx.Err(); err != nil {
		return err
	}

	return c.github.AddReview(ctx, task.repository, task.request, &github.PullRequestReviewRequest{
		Body: new(body), Comments: reply.Comments, Event: new(reply.Event),
	})
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
