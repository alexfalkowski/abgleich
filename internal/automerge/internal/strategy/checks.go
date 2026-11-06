package strategy

import (
	"github.com/alexfalkowski/abgleich/internal/github"
	"github.com/alexfalkowski/go-service/v2/context"
	"github.com/alexfalkowski/go-service/v2/strings"
)

// A push or an individual job completing does not prove the CI workflow has
// reached its sync stage. Require the configured final signal, then leave
// a missing signal and active work to later events or scheduled recovery.
func (s *Rebase) checksCompleted(ctx context.Context, repository *github.Repository, pull *github.PullRequest, completionCheck string) (bool, error) {
	sha := pull.GetHead().GetSHA()

	statuses, err := s.github.ListCommitStatuses(ctx, repository, sha)
	if err != nil {
		return false, err
	}

	checks, err := s.github.ListCheckRuns(ctx, repository, sha)
	if err != nil {
		return false, err
	}

	return checkCompleted(statuses, checks, completionCheck), nil
}

// Failed final signals count as completed; any unfinished work prevents rebasing.
func checkCompleted(statuses []*github.CommitStatus, checks []*github.CheckRun, completionCheck string) bool {
	completed := false

	for _, status := range statuses {
		switch status.GetState() {
		case "success", "failure", "error":
			completed = completed || status.GetContext() == completionCheck
		default:
			return false
		}
	}

	for _, check := range checks {
		if check.GetStatus() != "completed" || strings.IsEmpty(check.GetConclusion()) {
			return false
		}

		completed = completed || check.GetName() == completionCheck
	}

	return completed
}
