package strategy

import (
	"github.com/alexfalkowski/abgleich/internal/github"
	"github.com/alexfalkowski/go-service/v3/errors"
	"github.com/alexfalkowski/go-service/v3/strings"
)

// Action identifies a requested mutation; its zero value requests no mutation.
type Action string

const (
	actionMerge  Action = "merge"
	actionEnable Action = "enable"
	actionRebase Action = "rebase"
)

// String returns the mutation identifier, or empty when no mutation is requested.
func (a Action) String() string {
	return string(a)
}

// IsDefined reports whether an action value is present.
func (a Action) IsDefined() bool {
	return !strings.IsEmpty(a.String())
}

// Reason identifies the evidence or limit behind a strategy decision.
type Reason string

// ReasonDependabotExcluded requires explicit opt-in instead of author-based automation.
const ReasonDependabotExcluded Reason = "dependabot_excluded"

// ReasonReadyToMerge permits optional check failures when GitHub reports mergeability.
const ReasonReadyToMerge Reason = "ready_to_merge"

// ReasonPendingRequirements leaves merge requirements to GitHub.
const ReasonPendingRequirements Reason = "pending_requirements"

// ReasonSquashMergeDisabled prevents choosing another merge method.
const ReasonSquashMergeDisabled Reason = "squash_merge_disabled"

// ReasonLabelAuthorizationRevoked includes absent or ambiguous applications.
const ReasonLabelAuthorizationRevoked Reason = "label_authorization_revoked"

// ReasonHistoryUnavailable prevents inferring label authorization.
const ReasonHistoryUnavailable Reason = "history_unavailable"

// ReasonComparisonUnavailable prevents inferring branch freshness.
const ReasonComparisonUnavailable Reason = "comparison_unavailable"

// ReasonBranchCurrent means the comparison found no missing base commits.
const ReasonBranchCurrent Reason = "branch_current"

// ReasonBranchBehind means the comparison found missing base commits.
const ReasonBranchBehind Reason = "branch_behind"

// Decision records a selected policy and its evidence without report wording.
// Action is merge, enable, rebase, or empty when no operation is justified.
type Decision struct {
	// Policy identifies the strategy that selected the outcome.
	Policy string

	// Action is merge, enable, rebase, or empty when no mutation is justified.
	Action Action

	// Reason records the evidence or limitation for feature-owned explanations.
	Reason Reason
}

// IsMerge reports whether the decision calls for an immediate merge.
func (d Decision) IsMerge() bool {
	return d.Action == actionMerge
}

// IsEnable reports whether the decision calls for enabling squash auto-merge.
func (d Decision) IsEnable() bool {
	return d.Action == actionEnable
}

// IsRebase reports whether the decision calls for a server-side rebase.
func (d Decision) IsRebase() bool {
	return d.Action == actionRebase
}

func decideMerge(repository *github.Repository, pull *github.PullRequest) (Decision, error) {
	if !repository.GetAllowSquashMerge() {
		return Decision{Reason: ReasonSquashMergeDisabled}, errors.New("strategy: repository does not allow squash merging")
	}

	if readyToMerge(pull) {
		return Decision{Action: actionMerge, Reason: ReasonReadyToMerge}, nil
	}

	return Decision{Action: actionEnable, Reason: ReasonPendingRequirements}, nil
}
