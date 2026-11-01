package workflow

import (
	"fmt"
	"regexp"

	"github.com/alexfalkowski/abgleich/internal/github"
	"github.com/alexfalkowski/go-service/v2/context"
	"github.com/alexfalkowski/go-service/v2/errors"
)

var reviewThreadMarker = regexp.MustCompile(`(?m)^<!-- abgleich:([a-z]+):v2:[a-f0-9]{64} -->$`)

func (c *Command) reconcileThreads(ctx context.Context) error {
	repositories, err := c.github.ListRepositories(ctx)
	if err != nil {
		return err
	}

	var failures error
	for _, repository := range repositories {
		if err := ctx.Err(); err != nil {
			return errors.Join(failures, err)
		}

		if err := c.reconcileRepositoryThreads(ctx, repository); err != nil {
			failures = errors.Join(failures, fmt.Errorf("review: repository %s: %w", repository.GetFullName(), err))
			if github.IsRateLimited(err) || ctx.Err() != nil {
				return failures
			}
		}
	}
	return failures
}

func (c *Command) reconcileRepositoryThreads(ctx context.Context, repository *github.Repository) error {
	repository, err := c.github.GetRepository(ctx, repository)
	if err != nil || repository.GetArchived() || repository.GetDisabled() {
		return err
	}
	pulls, err := c.github.ListOpenPullRequests(ctx, repository)
	if err != nil {
		return err
	}

	var failures error
	for _, pull := range pulls {
		if err := ctx.Err(); err != nil {
			return errors.Join(failures, err)
		}
		if err := c.reconcilePullThreads(ctx, repository, pull); err != nil {
			failures = errors.Join(failures, fmt.Errorf("review: pull request #%d: %w", pull.GetNumber(), err))
			if github.IsRateLimited(err) || ctx.Err() != nil {
				return failures
			}
		}
	}
	return failures
}

func (c *Command) reconcilePullThreads(ctx context.Context, repository *github.Repository, pull *github.PullRequest) error {
	if !c.github.IsBaseRepository(repository, pull) {
		return errors.New("review: thread maintenance found a mismatched pull request")
	}
	if pull.GetState() != "open" {
		return nil
	}

	threads, err := c.github.ListReviewThreads(ctx, repository, pull)
	if err != nil {
		return err
	}

	var failures error
	for _, thread := range threads {
		if err := ctx.Err(); err != nil {
			return errors.Join(failures, err)
		}
		if !c.outdatedThread(thread) {
			continue
		}
		if err := c.resolveThread(ctx, repository, pull, thread); err != nil {
			failures = errors.Join(failures, fmt.Errorf("review: thread %s: %w", thread.ID, err))
			if github.IsRateLimited(err) || ctx.Err() != nil {
				return failures
			}
		}
	}
	return failures
}

func (c *Command) resolveThread(ctx context.Context, repository *github.Repository, pull *github.PullRequest, thread *github.ReviewThread) error {
	current, err := c.github.GetPullRequest(ctx, repository, pull.GetNumber())
	if err != nil {
		return err
	}

	sameNumber := current.GetNumber() == pull.GetNumber()
	sameNodeID := current.GetNodeID() == pull.GetNodeID()
	sameRepository := c.github.IsBaseRepository(repository, current)
	if !sameNumber || !sameNodeID || !sameRepository {
		return errors.New("review: pull request identity changed before thread resolution")
	}
	if current.GetState() != "open" {
		return nil
	}

	fresh, err := c.github.GetReviewThread(ctx, repository, current, thread.ID)
	canResolve := err == nil && fresh != nil && c.outdatedThread(fresh)
	if !canResolve {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	// GitHub has no expected-state guard for resolution. Changes after this
	// final read can still race the mutation; outdated does not mean fixed.
	return c.github.ResolveReviewThread(ctx, repository, current, fresh)
}

func (c *Command) outdatedThread(thread *github.ReviewThread) bool {
	if !thread.IsAppAuthor || !thread.IsOutdated || thread.IsResolved || !thread.ViewerCanResolve {
		return false
	}

	marker := reviewThreadMarker.FindStringSubmatch(thread.ReviewBody)
	return len(marker) == 2 && marker[1] == c.definition.Name
}
