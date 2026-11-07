package strategy

import "github.com/alexfalkowski/abgleich/internal/github"

// HoldLabel is the exact label that pauses merge, enablement, and rebase actions.
const HoldLabel = "hold"

// Eligible leaves closed, draft, and exactly hold-labeled PRs untouched.
func Eligible(pull *github.PullRequest) bool {
	return pull.GetState() == github.PullOpen && !pull.GetDraft() && !github.HasLabel(pull.GetLabels(), HoldLabel)
}

// readyToMerge accepts optional check failures; GitHub still enforces merge requirements.
func readyToMerge(pull *github.PullRequest) bool {
	state := pull.GetMergeableState()
	return pull.GetMergeable() && (state == github.MergeableClean || state == github.MergeableUnstable)
}
