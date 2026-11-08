package workflow

import (
	"crypto/sha256"
	"fmt"
	"regexp"
	"strconv"

	"github.com/alexfalkowski/abgleich/internal/github"
	"github.com/alexfalkowski/go-service/v2/context"
	"github.com/alexfalkowski/go-service/v2/strings"
)

var reviewThreadMarker = regexp.MustCompile(`(?m)^<!-- abgleich:([a-z]+):v2:[a-f0-9]{64} -->$`)

func (c *Command) hasReview(ctx context.Context, task *reviewTask, comments []*github.IssueComment) (bool, error) {
	marker := c.resultMarker(task)
	for _, comment := range comments {
		if c.github.IsAppComment(comment) && strings.Contains(comment.GetBody(), "\n"+marker) {
			return true, nil
		}
	}

	reviews, err := c.github.ListReviews(ctx, task.repository, task.request)
	if err != nil {
		return false, err
	}

	var candidates []*github.PullRequestReview
	for _, review := range reviews {
		switch review.GetState() {
		case github.ReviewCommented, github.ReviewApproved, github.ReviewChangesRequested, github.ReviewDismissed:
		default:
			continue
		}
		if strings.Contains(review.GetBody(), "\n"+marker) {
			candidates = append(candidates, review)
		}
	}

	return c.github.HasAppReview(ctx, candidates)
}

func (c *Command) resultMarker(task *reviewTask) string {
	identity := strings.Join("\n",
		c.profile.Name,
		strings.ToLower(task.repository.GetFullName()),
		strconv.Itoa(task.request.GetNumber()),
		task.request.GetNodeID(),
		task.request.GetHead().GetSHA(),
		task.request.GetBase().GetSHA(),
	)
	return fmt.Sprintf("<!-- abgleich:%s:v2:%x -->", c.profile.Name, sha256.Sum256(strings.Bytes(identity)))
}
