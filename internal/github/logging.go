package github

import (
	"github.com/alexfalkowski/go-service/v2/context"
	"github.com/alexfalkowski/go-service/v2/telemetry/logger"
)

type loggingService struct {
	Service
	logger *logger.Logger
}

func (s *loggingService) AddCommentReaction(ctx context.Context, repo *Repository, comment *IssueComment, content string) error {
	if err := s.Service.AddCommentReaction(ctx, repo, comment, content); err != nil {
		s.logger.Warn("github comment reaction failed", "repository", repo.GetFullName(), "comment", comment.GetID(),
			"reaction", content, "error", err)
		return err
	}

	s.logger.Info("added GitHub comment reaction", "repository", repo.GetFullName(), "comment", comment.GetID(), "reaction", content)

	return nil
}

func (s *loggingService) AddLabel(ctx context.Context, repo *Repository, number int, label string) error {
	if err := s.Service.AddLabel(ctx, repo, number, label); err != nil {
		return err
	}

	s.logger.Info("added pull request label", "repository", repo.GetFullName(), "pull_request", number, "label", label)

	return nil
}

func (s *loggingService) EnableAutoMerge(ctx context.Context, pull *PullRequest) error {
	if err := s.Service.EnableAutoMerge(ctx, pull); err != nil {
		return err
	}

	s.logger.Info("enabled pull request auto-merge", "repository", pull.GetBase().GetRepo().GetFullName(),
		"pull_request", pull.GetNumber(), "branch", pull.GetHead().GetRef())

	return nil
}

func (s *loggingService) Merge(ctx context.Context, repo *Repository, pull *PullRequest) error {
	if err := s.Service.Merge(ctx, repo, pull); err != nil {
		return err
	}

	s.logger.Info("merged pull request", "repository", repo.GetFullName(),
		"pull_request", pull.GetNumber(), "branch", pull.GetHead().GetRef())

	return nil
}

func (s *loggingService) Rebase(ctx context.Context, pull *PullRequest) error {
	if err := s.Service.Rebase(ctx, pull); err != nil {
		return err
	}

	s.logger.Info("requested pull request rebase", "repository", pull.GetBase().GetRepo().GetFullName(),
		"pull_request", pull.GetNumber(), "branch", pull.GetHead().GetRef())

	return nil
}
