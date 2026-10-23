package strategy

import "github.com/alexfalkowski/abgleich/internal/github"

// Eligible applies the shared requirement that closed and draft PRs remain untouched.
func Eligible(pull *github.PullRequest) bool {
	return pull.GetState() == "open" && !pull.GetDraft()
}

// readyToMerge requires GitHub to confirm mergeability and a clean merge state.
func readyToMerge(pull *github.PullRequest) bool {
	return pull.GetMergeable() && pull.GetMergeableState() == "clean"
}
