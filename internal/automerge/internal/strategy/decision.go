package strategy

import (
	"github.com/alexfalkowski/abgleich/internal/github"
	"github.com/alexfalkowski/go-service/v2/errors"
)

// Reason identifies the evidence or limit behind a strategy decision.
type Reason string

const (
	// ReasonReadyToMerge permits optional check failures when GitHub reports mergeability.
	ReasonReadyToMerge Reason = "ready_to_merge"
	// ReasonPendingRequirements leaves merge requirements to GitHub.
	ReasonPendingRequirements Reason = "pending_requirements"
	// ReasonSquashMergeDisabled prevents choosing another merge method.
	ReasonSquashMergeDisabled Reason = "squash_merge_disabled"
	// ReasonLabelAuthorizationRevoked includes absent or ambiguous applications.
	ReasonLabelAuthorizationRevoked Reason = "label_authorization_revoked"
	// ReasonHistoryUnavailable prevents inferring label authorization.
	ReasonHistoryUnavailable Reason = "history_unavailable"
	// ReasonComparisonUnavailable prevents inferring branch freshness.
	ReasonComparisonUnavailable Reason = "comparison_unavailable"
	// ReasonBranchCurrent means the comparison found no missing base commits.
	ReasonBranchCurrent Reason = "branch_current"
	// ReasonBranchBehind means the comparison found missing base commits.
	ReasonBranchBehind Reason = "branch_behind"
)

// Decision records a selected policy and its evidence without report wording.
// Action is merge, enable, rebase, or empty when no operation is justified.
type Decision struct {
	Policy string
	Action string
	Reason Reason
}

// IsMerge reports whether the decision calls for an immediate merge.
func (d Decision) IsMerge() bool {
	return d.Action == "merge"
}

// IsEnable reports whether the decision calls for enabling squash auto-merge.
func (d Decision) IsEnable() bool {
	return d.Action == "enable"
}

// IsRebase reports whether the decision calls for a server-side rebase.
func (d Decision) IsRebase() bool {
	return d.Action == "rebase"
}

func decideMerge(repository *github.Repository, pull *github.PullRequest) (Decision, error) {
	if !repository.GetAllowSquashMerge() {
		return Decision{Reason: ReasonSquashMergeDisabled}, errors.New("strategy: repository does not allow squash merging")
	}

	if readyToMerge(pull) {
		return Decision{Action: "merge", Reason: ReasonReadyToMerge}, nil
	}

	return Decision{Action: "enable", Reason: ReasonPendingRequirements}, nil
}
