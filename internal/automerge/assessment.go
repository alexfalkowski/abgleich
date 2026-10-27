package automerge

import (
	"github.com/alexfalkowski/abgleich/internal/automerge/internal/strategy"
	"github.com/alexfalkowski/abgleich/internal/github"
	"github.com/alexfalkowski/go-service/v2/context"
)

// Assessment explains an observed automation decision and its evidence limits.
// Action describes a possible request, not a promise that reconciliation or
// GitHub will perform it. Creating an assessment never requests automation.
type Assessment struct {
	Policy   string
	Decision string
	Reason   string
	NextStep string
	Action   string
}

// Assess explains the first applicable policy for the supplied repository and
// PR snapshots without requesting automation. It may read complete history or
// branch comparisons. Remote evidence failures produce an unavailable assessment;
// callers must observe cancellation and recheck changing snapshots before use.
func (r *Reconciler) Assess(ctx context.Context, repo *github.Repository, pull *github.PullRequest) Assessment {
	if !eligibleRepository(repo) {
		return Assessment{
			Policy: "repository opt-in", Decision: "Repository automation disabled",
			Reason:   "The repository is archived, disabled, or does not allow auto-merge. Scheduled automation skips it.",
			NextStep: "Ask a repository administrator to review the repository's state and Allow auto-merge setting.",
		}
	}

	if !strategy.Eligible(pull) {
		return explainIneligible(pull)
	}

	if err := github.ValidatePullRequestMetadata(pull); err != nil {
		return Assessment{
			Policy: "pull request metadata", Decision: "Assessment unavailable",
			Reason:   "Pull request branch metadata is incomplete; the worker cannot establish an automation decision.",
			NextStep: "Retry /ab status after GitHub supplies complete pull request metadata.",
		}
	}

	for _, s := range r.strategies {
		if s.Matches(repo, pull) {
			// Decision reasons retain unavailable evidence alongside remote errors.
			decision, _ := s.Assess(ctx, repo, pull)
			return explainDecision(decision)
		}
	}

	if pull.GetAutoMerge() != nil && !r.github.IsSameRepository(repo, pull) {
		return Assessment{
			Policy: "rebase", Decision: "Fork rebase unsupported",
			Reason:   "GitHub auto-merge is enabled, but the head branch belongs to a fork. Abgleich never rebases fork branches.",
			NextStep: "Ask the contributor to keep the fork branch current. GitHub controls the existing auto-merge request.",
		}
	}

	return Assessment{
		Policy: "none", Decision: "No automation requested",
		Reason:   "No Dependabot, automerge-label, or same-repository auto-merge strategy applies to this pull request.",
		NextStep: "An authorized maintainer can apply automerge or post /ab automerge to request labeled automation.",
	}
}

// SamePolicyState compares revision, repository settings, and PR eligibility
// and merge state across snapshots. It does not establish atomicity of history
// or comparison reads, nor validate command authorization.
func (r *Reconciler) SamePolicyState(currentRepo, expectedRepo *github.Repository, current, expected *github.PullRequest) bool {
	samePullRequest := r.github.IsSamePullRequest(current, expected)
	sameBaseCommit := current.GetBase().GetSHA() == expected.GetBase().GetSHA()
	sameUpdatedAt := current.GetUpdatedAt().Equal(expected.GetUpdatedAt())
	sameRevision := samePullRequest && sameBaseCommit && sameUpdatedAt
	sameSettings := sameRepositorySettings(currentRepo, expectedRepo)
	sameEligibility := sameStatusEligibility(current, expected)
	sameMergeState := sameStatusMergeState(current, expected)
	return sameSettings && sameEligibility && sameMergeState && sameRevision
}

func explainDecision(decision strategy.Decision) Assessment {
	assessment := Assessment{Policy: decision.Policy, Action: decision.Action}
	switch decision.Reason {
	case strategy.ReasonDependabotExcluded:
		assessment.Decision = "Dependabot automation disabled"
		assessment.Reason = "This repository is excluded from automatic Dependabot merging by worker configuration."
		assessment.NextStep = "An authorized maintainer can apply automerge or post /ab automerge to request labeled automation."
	case strategy.ReasonReadyToMerge:
		assessment.Decision = "Would squash-merge"
		assessment.Reason = "GitHub reports this pull request ready to merge. Optional check failures do not block merging."
		assessment.NextStep = "No new request is needed. GitHub still enforces repository rules when a sweep attempts the merge."
	case strategy.ReasonPendingRequirements:
		assessment.Decision = "Would enable squash auto-merge"
		assessment.Reason = "GitHub has not confirmed this pull request ready to merge. The worker would request squash auto-merge " +
			"and let GitHub enforce checks, reviews, and repository rules."
		assessment.NextStep = "Inspect GitHub's merge box for outstanding requirements; this assessment does not identify individual checks or reviews."
	case strategy.ReasonSquashMergeDisabled:
		assessment.Decision = "Squash merging unavailable"
		assessment.Reason = "The selected merge policy requires squash merging, which this repository has disabled."
		assessment.NextStep = "Ask a repository administrator to enable squash merging if this automation is wanted."
	case strategy.ReasonLabelAuthorizationRevoked:
		assessment.Decision = "Waiting for renewed authorization"
		assessment.Reason = "The automerge label has no application strictly newer than its latest removal and the latest auto-merge disablement. " +
			"Missing application events and equal timestamps do not authorize automation."
		assessment.NextStep = "Remove and reapply automerge to renew the request. Wait at least a second before reapplying it."
	case strategy.ReasonHistoryUnavailable:
		assessment.Decision = "Assessment unavailable"
		assessment.Reason = "Complete automation history is unavailable; label authorization cannot be established."
		assessment.NextStep = "Retry /ab status when GitHub history is available. No authorization is inferred from missing evidence."
	case strategy.ReasonComparisonUnavailable:
		assessment.Decision = "Assessment unavailable"
		assessment.Reason = "Branch comparison evidence is unavailable; branch freshness cannot be established."
		assessment.NextStep = "Retry /ab status when GitHub can compare the branches."
	case strategy.ReasonBranchCurrent:
		assessment.Decision = "Branch is current"
		assessment.Reason = "The comparison reports no missing base commits at the observed revision."
		assessment.NextStep = "No rebase is needed. GitHub controls the existing auto-merge request and its requirements."
	case strategy.ReasonBranchBehind:
		assessment.Decision = "Would request a rebase"
		assessment.Reason = "Auto-merge is enabled and the same-repository branch is missing base commits."
		assessment.NextStep = "No new request is needed. A sweep would request a guarded server-side rebase; GitHub may still reject it."
	}
	return assessment
}

func sameRepositorySettings(current, expected *github.Repository) bool {
	sameArchived := current.GetArchived() == expected.GetArchived()
	sameDisabled := current.GetDisabled() == expected.GetDisabled()
	sameAutoMerge := current.GetAllowAutoMerge() == expected.GetAllowAutoMerge()
	sameSquashMerge := current.GetAllowSquashMerge() == expected.GetAllowSquashMerge()
	return sameArchived && sameDisabled && sameAutoMerge && sameSquashMerge
}

func sameStatusEligibility(current, expected *github.PullRequest) bool {
	sameState := current.GetState() == expected.GetState()
	sameDraft := current.GetDraft() == expected.GetDraft()
	sameAuthor := current.GetUser().GetLogin() == expected.GetUser().GetLogin()
	sameHold := github.HasLabel(current.GetLabels(), "hold") == github.HasLabel(expected.GetLabels(), "hold")
	sameAutoMergeLabel := github.HasLabel(current.GetLabels(), strategy.AutoMergeLabel) == github.HasLabel(expected.GetLabels(), strategy.AutoMergeLabel)
	return sameState && sameDraft && sameAuthor && sameHold && sameAutoMergeLabel
}

func sameStatusMergeState(current, expected *github.PullRequest) bool {
	sameMergeable := current.GetMergeable() == expected.GetMergeable()
	sameMergeableState := current.GetMergeableState() == expected.GetMergeableState()
	sameAutoMerge := (current.GetAutoMerge() == nil) == (expected.GetAutoMerge() == nil)
	sameMergeMethod := current.GetAutoMerge().GetMergeMethod() == expected.GetAutoMerge().GetMergeMethod()
	return sameMergeable && sameMergeableState && sameAutoMerge && sameMergeMethod
}

func explainIneligible(pull *github.PullRequest) Assessment {
	if pull.GetState() != "open" {
		return Assessment{
			Policy: "PR eligibility", Decision: "Pull request is closed",
			Reason:   "Only open pull requests are eligible for Abgleich automation.",
			NextStep: "No worker action is needed for a closed pull request.",
		}
	}

	if pull.GetDraft() {
		return Assessment{
			Policy: "PR eligibility", Decision: "Draft pull request",
			Reason:   "Abgleich leaves draft pull requests untouched.",
			NextStep: "Mark the pull request ready for review when it is ready for automation.",
		}
	}

	return Assessment{
		Policy: "hold", Decision: "On hold",
		Reason:   "The exact hold label pauses Abgleich. It does not disable auto-merge already enabled on GitHub.",
		NextStep: "Remove hold to resume normal policy. To stop existing GitHub auto-merge, disable it in GitHub separately.",
	}
}
