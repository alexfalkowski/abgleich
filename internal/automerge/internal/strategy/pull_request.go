package strategy

import "github.com/alexfalkowski/abgleich/internal/github"

// Eligible leaves closed, draft, and exactly hold-labeled PRs untouched.
func Eligible(pull *github.PullRequest) bool {
	return pull.GetState() == "open" && !pull.GetDraft() && !github.HasLabel(pull.GetLabels(), "hold")
}

// readyToMerge accepts optional check failures; GitHub still enforces merge requirements.
func readyToMerge(pull *github.PullRequest) bool {
	state := pull.GetMergeableState()
	return pull.GetMergeable() && (state == "clean" || state == "unstable")
}
