package cmd

import (
	"github.com/alexfalkowski/abgleich/internal/automerge"
	"github.com/alexfalkowski/abgleich/internal/config"
	"github.com/alexfalkowski/abgleich/internal/github"
	"github.com/alexfalkowski/abgleich/internal/scheduler"
	"github.com/alexfalkowski/go-service/v2/telemetry/logger"
)

func newRebaser(cfg *config.Config, repository *github.Repository, log *logger.Logger) *automerge.Rebaser {
	return automerge.NewRebaser(cfg.GitHub.Owner+"/"+cfg.GitHub.Repository, repository, log)
}

func newAutoMergeJob(cfg *config.Config, rebaser *automerge.Rebaser, log *logger.Logger) *scheduler.Job {
	return scheduler.NewJob("automerge", cfg.Scheduler, rebaser.Rebase, log)
}
