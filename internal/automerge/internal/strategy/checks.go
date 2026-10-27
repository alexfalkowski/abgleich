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

	completed := completedCheckNames(statuses, checks)
	return completed[completionCheck], nil
}

// A nil result means some reported work is still active or incomplete. Failed
// final signals count as completed; they never bypass GitHub's merge requirements.
func completedCheckNames(statuses []*github.CommitStatus, checks []*github.CheckRun) map[string]bool {
	completed := make(map[string]bool)

	for _, status := range statuses {
		switch status.GetState() {
		case "success", "failure", "error":
			completed[status.GetContext()] = true
		default:
			return nil
		}
	}

	for _, check := range checks {
		if check.GetStatus() != "completed" || strings.IsEmpty(check.GetConclusion()) {
			return nil
		}
		completed[check.GetName()] = true
	}

	return completed
}
