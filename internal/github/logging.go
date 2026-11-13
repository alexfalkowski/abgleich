package github

import (
	"github.com/alexfalkowski/go-service/v3/context"
	"github.com/alexfalkowski/go-service/v3/telemetry/logger"
)

type loggingService struct {
	Service
	logger *logger.Logger
}

// ResolveReviewThread delegates resolution to Service and logs only confirmed
// success with the repository, pull request, and thread identity.
func (s *loggingService) ResolveReviewThread(ctx context.Context, repo *Repository, pull *PullRequest, thread *ReviewThread) error {
	if err := s.Service.ResolveReviewThread(ctx, repo, pull, thread); err != nil {
		return err
	}

	s.logger.Info("resolved pull request review thread",
		"repository", repo.GetFullName(),
		"pull_request", pull.GetNumber(),
		"thread", thread.ID)

	return nil
}

// AddReview delegates publication to Service and logs the accepted verdict and
// reviewed head SHA without recording the review body.
func (s *loggingService) AddReview(ctx context.Context, repo *Repository, pull *PullRequest, review *PullRequestReviewRequest) error {
	if err := s.Service.AddReview(ctx, repo, pull, review); err != nil {
		return err
	}

	s.logger.Info("published pull request review",
		"repository", repo.GetFullName(),
		"pull_request", pull.GetNumber(),
		"event", review.GetEvent(),
		"sha", pull.GetHead().GetSHA())

	return nil
}

// DeleteBranch delegates deletion to Service and logs the branch and expected
// head SHA only after the operation succeeds.
func (s *loggingService) DeleteBranch(ctx context.Context, repo *Repository, branch *Branch) error {
	if err := s.Service.DeleteBranch(ctx, repo, branch); err != nil {
		return err
	}

	s.logger.Info("deleted branch", "repository", repo.GetFullName(), "branch", branch.GetName(), "sha", branch.GetCommit().GetSHA())

	return nil
}

// AddComment delegates publication to Service and logs the target pull request
// only after success, without recording the comment body.
func (s *loggingService) AddComment(ctx context.Context, repo *Repository, number int, body string) error {
	if err := s.Service.AddComment(ctx, repo, number, body); err != nil {
		return err
	}

	s.logger.Info("published pull request comment",
		"repository", repo.GetFullName(),
		"pull_request", number)

	return nil
}

// AddCommentReaction delegates to Service and logs both failures and successful
// reactions with their repository and comment identity.
func (s *loggingService) AddCommentReaction(ctx context.Context, repo *Repository, comment *IssueComment, content Reaction) error {
	if err := s.Service.AddCommentReaction(ctx, repo, comment, content); err != nil {
		s.logger.Error("github comment reaction failed",
			"repository", repo.GetFullName(),
			"comment", comment.GetID(),
			"reaction", content,
			"error", err)
		return err
	}

	s.logger.Info("added GitHub comment reaction",
		"repository", repo.GetFullName(),
		"comment", comment.GetID(),
		"reaction", content)

	return nil
}

// AddLabel delegates to Service and logs the requested label only after success.
func (s *loggingService) AddLabel(ctx context.Context, repo *Repository, number int, label string) error {
	if err := s.Service.AddLabel(ctx, repo, number, label); err != nil {
		return err
	}

	s.logger.Info("added pull request label",
		"repository", repo.GetFullName(),
		"pull_request", number,
		"label", label)

	return nil
}

// EnableAutoMerge delegates to Service and logs confirmed enablement with the
// pull request's target repository and head branch.
func (s *loggingService) EnableAutoMerge(ctx context.Context, pull *PullRequest) error {
	if err := s.Service.EnableAutoMerge(ctx, pull); err != nil {
		return err
	}

	s.logger.Info("enabled pull request auto-merge",
		"repository", pull.GetBase().GetRepo().GetFullName(),
		"pull_request", pull.GetNumber(),
		"branch", pull.GetHead().GetRef())

	return nil
}

// Merge delegates to Service and logs the pull request and branch only after a
// confirmed merge.
func (s *loggingService) Merge(ctx context.Context, repo *Repository, pull *PullRequest) error {
	if err := s.Service.Merge(ctx, repo, pull); err != nil {
		return err
	}

	s.logger.Info("merged pull request",
		"repository", repo.GetFullName(),
		"pull_request", pull.GetNumber(),
		"branch", pull.GetHead().GetRef())

	return nil
}

// Rebase delegates to Service and logs acceptance of the rebase request, which
// does not confirm that GitHub has completed the rebase.
func (s *loggingService) Rebase(ctx context.Context, pull *PullRequest) error {
	if err := s.Service.Rebase(ctx, pull); err != nil {
		return err
	}

	s.logger.Info("requested pull request rebase",
		"repository", pull.GetBase().GetRepo().GetFullName(),
		"pull_request", pull.GetNumber(),
		"branch", pull.GetHead().GetRef())

	return nil
}
