package workflow

import (
	"crypto/sha256"
	"fmt"
	"regexp"
	"strconv"

	"github.com/alexfalkowski/abgleich/internal/github"
	"github.com/alexfalkowski/go-service/v3/context"
	"github.com/alexfalkowski/go-service/v3/strings"
)

// Partial reviews retain feature identity for verdicts and thread maintenance.
// Their markers do not match successful results during deduplication.
var reviewThreadMarker = regexp.MustCompile(`(?m)^<!-- abgleich:([a-z]+):v2:(?:partial:)?[a-f0-9]{64} -->$`)

type reviewHistory struct {
	reviews       []*github.PullRequestReview
	authenticated []*github.PullRequestReview
	authChecked   bool
}

func (c *Command) hasReview(ctx context.Context, task *reviewTask, comments []*github.IssueComment) (bool, error) {
	exists, _, err := c.reviewHistory(ctx, task, comments)
	return exists, err
}

func (c *Command) reviewHistory(ctx context.Context, task *reviewTask, comments []*github.IssueComment) (bool, reviewHistory, error) {
	marker := c.resultMarker(task)
	legacy := c.latestMatchingComment(comments, marker)

	reviews, err := c.github.ListReviews(ctx, task.repository, task.request)
	if err != nil {
		return false, reviewHistory{}, err
	}
	history := reviewHistory{reviews: reviews}

	if legacy == nil && !containsResult(reviews, marker) {
		return false, history, nil
	}

	authored, err := c.authenticatedAssessments(ctx, reviews)
	if err != nil {
		return false, reviewHistory{}, err
	}
	history.authenticated, history.authChecked = authored, true
	latest := lastAssessment(authored)
	if legacy != nil && (latest == nil || latest.GetSubmittedAt().Before(legacy.GetCreatedAt().Time)) {
		return true, history, nil
	}
	return strings.Contains(latest.GetBody(), "\n"+marker), history, nil
}

func (c *Command) latestMatchingComment(comments []*github.IssueComment, marker string) *github.IssueComment {
	var latest *github.IssueComment
	for _, comment := range comments {
		if c.github.IsAppComment(comment) && strings.Contains(comment.GetBody(), "\n"+marker) &&
			(latest == nil || latest.GetCreatedAt().Before(comment.GetCreatedAt().Time)) {
			latest = comment
		}
	}
	return latest
}

func (c *Command) historyAssessments(ctx context.Context, history reviewHistory) ([]*github.PullRequestReview, error) {
	if history.authChecked {
		return history.authenticated, nil
	}
	return c.authenticatedAssessments(ctx, history.reviews)
}

func containsResult(reviews []*github.PullRequestReview, marker string) bool {
	for _, review := range reviews {
		if strings.Contains(review.GetBody(), "\n"+marker) {
			return true
		}
	}
	return false
}

func (c *Command) resultMarker(task *reviewTask) string {
	return c.resultMarkerAt(task, task.request.GetHead().GetSHA(), task.request.GetBase().GetSHA(), false)
}

func (c *Command) resultMarkerAt(task *reviewTask, headSHA, baseSHA string, partial bool) string {
	identity := strings.Join("\n",
		c.profile.Name,
		strings.ToLower(task.repository.GetFullName()),
		strconv.Itoa(task.request.GetNumber()),
		task.request.GetNodeID(),
		headSHA,
		baseSHA,
	)
	var kind string
	if partial {
		kind = "partial:"
	}
	return fmt.Sprintf("<!-- abgleich:%s:v2:%s%x -->", c.profile.Name, kind, sha256.Sum256(strings.Bytes(identity)))
}
