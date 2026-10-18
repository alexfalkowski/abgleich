package github

import (
	"github.com/alexfalkowski/go-service/v2/context"
	"github.com/alexfalkowski/go-service/v2/telemetry/logger"
)

type loggingService struct {
	Service
	logger *logger.Logger
}

func (s *loggingService) Rebase(ctx context.Context, pull *PullRequest) error {
	if err := s.Service.Rebase(ctx, pull); err != nil {
		return err
	}

	s.logger.Info("requested pull request rebase", "repository", pull.GetBase().GetRepo().GetFullName(),
		"pull_request", pull.GetNumber(), "branch", pull.GetHead().GetRef())

	return nil
}
