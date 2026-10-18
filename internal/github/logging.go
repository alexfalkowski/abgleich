package github

import (
	"github.com/alexfalkowski/go-service/v2/context"
	"github.com/alexfalkowski/go-service/v2/telemetry/logger"
)

type loggingRepository struct {
	Repository
	logger *logger.Logger
}

func (r *loggingRepository) Rebase(ctx context.Context, pull *PullRequest) error {
	if err := r.Repository.Rebase(ctx, pull); err != nil {
		return err
	}

	r.logger.Info("requested pull request rebase", "pull_request", pull.GetNumber(), "branch", pull.GetHead().GetRef())

	return nil
}
