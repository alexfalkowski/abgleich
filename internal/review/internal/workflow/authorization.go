package workflow

import (
	"github.com/alexfalkowski/abgleich/internal/commands"
	"github.com/alexfalkowski/abgleich/internal/github"
	"github.com/alexfalkowski/abgleich/internal/review/internal/history"
	"github.com/alexfalkowski/go-service/v3/context"
	"github.com/alexfalkowski/go-service/v3/errors"
	"github.com/alexfalkowski/go-service/v3/strings"
)

// consent refreshes review-specific PR eligibility and shared human authorization.
type consent struct {
	github     github.Service
	authorizer *commands.Authorizer
}

func (c consent) authorize(ctx context.Context, task *reviewTask) (*reviewTask, []*github.IssueComment, error) {
	current, err := c.refreshTask(ctx, task)
	if err != nil {
		return nil, nil, err
	}

	if current.request.GetNumber() != task.request.GetNumber() || !c.eligible(current.repository, current.request) {
		return nil, nil, nil
	}

	comment, comments, err := c.authorizer.AuthorizeWithHistory(ctx, task.command, current.repository, current.request.GetNumber(), task.comment)
	if err != nil || comment == nil {
		return nil, nil, err
	}

	current.comment = comment
	return current, comments, nil
}

// Cancellation still requires consent and PR identity, while permitting closed and draft PRs.
func (c consent) authorizeCancellation(ctx context.Context, task *reviewTask) (*reviewTask, []*github.IssueComment, error) {
	current, err := c.refreshTask(ctx, task)
	if err != nil {
		return nil, nil, err
	}

	if !c.github.IsSamePullRequestIdentity(current.repository, current.request, task.request) {
		return nil, nil, errors.New("review: pull request identity changed before cancellation")
	}

	comment, comments, err := c.authorizer.AuthorizeWithHistory(ctx, task.command, current.repository, current.request.GetNumber(), task.comment)
	if err != nil || comment == nil {
		return nil, nil, err
	}

	current.comment = comment
	return current, comments, nil
}

func (c consent) refreshTask(ctx context.Context, task *reviewTask) (*reviewTask, error) {
	if err := c.github.ValidateInstallation(ctx); err != nil {
		return nil, err
	}

	repo, err := c.github.GetRepository(ctx, task.repository)
	if err != nil {
		return nil, err
	}

	pull, err := c.github.GetPullRequest(ctx, repo, task.request.GetNumber())
	if err != nil {
		return nil, err
	}

	return &reviewTask{repository: repo, request: pull, comment: task.comment, command: task.command}, nil
}

func (c consent) eligible(repo *github.Repository, pull *github.PullRequest) bool {
	return pull.GetState() == github.PullOpen &&
		!pull.GetDraft() &&
		c.github.IsSameRepository(repo, pull) &&
		!strings.IsEmpty(pull.GetBase().GetSHA()) &&
		github.ValidatePullRequestMetadata(pull) == nil
}

func (c consent) revisionChanged(expected, current *reviewTask) bool {
	return !c.github.IsSamePullRequest(current.request, expected.request) ||
		current.request.GetBase().GetSHA() != expected.request.GetBase().GetSHA()
}

// recheck returns a refreshed task only while its authorized revisions still match.
// A nil task stops processing, including existing results and stale revisions.
func (c *Command) recheck(ctx context.Context, task *reviewTask) (*reviewTask, history.View, error) {
	current, comments, err := c.consent.authorize(ctx, task)
	if err != nil {
		return nil, history.View{}, err
	}
	if current == nil {
		return nil, history.View{}, c.publishCancellation(ctx, task)
	}

	if c.consent.revisionChanged(task, current) {
		return nil, history.View{}, c.publishStaleReview(ctx, task, staleRevisionReason)
	}

	exists, reviews, err := c.history(current).Read(ctx, comments)
	if err != nil || exists {
		return nil, history.View{}, err
	}

	return current, reviews, nil
}

func (c *Command) guard(ctx context.Context, task *reviewTask, review *reviewEvidence) error {
	current, comments, err := c.consent.authorize(ctx, task)
	if err != nil {
		return err
	}

	if current == nil || c.consent.revisionChanged(task, current) {
		return errors.New("review: authorization or revision changed")
	}

	exists, previous, err := c.history(current).Read(ctx, comments)
	if err != nil {
		return err
	}
	if exists {
		return errReviewStopped
	}

	if review != nil && review.followup.HasBlockerEvidence() {
		reason, err := c.history(current).CheckPrevious(ctx, previous, review.previous, review.followup)
		if err != nil {
			return err
		}
		if reason != "" {
			return staleReviewError(reason)
		}
	}

	return ctx.Err()
}
