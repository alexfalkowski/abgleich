# Accepted design for GitHub automation

Part of the [accepted-design index](../accepted-design.md).

## GitHub API boundary

GitHub performs all repository mutations. The worker operates through
`github.Service` without cloning repositories or running Git. Keep direct
go-github imports inside `internal/github`. Reuse SDK types through the
project's aliases and their pointer/getter APIs. Use go-service HTTP constants
where available.

Read PRs through REST. Use the GraphQL `updatePullRequestBranch` mutation with
`updateMethod: REBASE` for server-side rebases because REST branch updates
merge. Keep custom GraphQL payloads local and use the PR's `NodeID` for
GraphQL operations.

HTTP retries apply only to safe methods. An ambiguous mutation response is
handled by reading fresh state on a later run. Validate GraphQL errors and
response identity even on HTTP success. Reject comparisons missing
`behind_by`.

[GitHub service](../../../internal/github/service.go) ·
[GitHub client](../../../internal/github/client.go) ·
[Rebase mutation](../../../internal/github/rebase.go) ·
[Recovery scenarios](../../../test/features/automerge/recovery.feature)

## Reconciliation and fork eligibility

Each run reconciles fresh remote state. Finish owner-wide repository discovery
before mutating. Read full repository settings before listing PRs because
discovery omits merge settings. Require auto-merge opt-in and complete PR
pagination for each repository before acting on its PRs. Event checks share
the scheduled sweep's coordinator and check only the affected repository.
Event-driven rebases also require the configured final CI signal on the
current head and no active checks or pending commit statuses. Scheduled
sweeps recover missed events and failures without requiring that signal.

Eligible PRs are open and non-draft with both branches in the discovered
repository. Fork contributions lack CI in this workflow. Apply the same
boundary to Dependabot, the exact `automerge` label, `/ab automerge`, and
rebasing. Status explains the fork exclusion. Summaries, reviews, and
review-thread maintenance also exclude forks. GitHub retains control of any
existing auto-merge on a fork.

Evaluate Dependabot, label, and rebase strategies in that order. The first
matching strategy owns the result, including a no-op or error. Recheck
eligibility, identity, and branch state before acting. Guard rebases and
auto-merge enablement with `expectedHeadOid` and immediate merges with the
expected head SHA. Attempt at most one mutation per PR per sweep or event
check and continue processing other PRs after an individual failure. Leave
PR approval to reviewers and merge-rule enforcement to GitHub.

[Reconciler](../../../internal/automerge/reconciler.go) ·
[Event coordinator](../../../internal/automerge/coordinator.go) ·
[Automation guide](../../automation.md) ·
[Event scenarios](../../../test/features/automerge/events.feature)

## Label consent and holds

The exact `hold` label pauses auto-merge enablement, immediate merges, rebases,
and `/ab automerge` label requests. Recheck it before acting. Existing GitHub
auto-merge and status, summary, and review commands are independent of the
hold.

Read complete label and auto-merge-disable history before acting on labeled
authorization. The label must be strictly newer than its latest removal and
the latest auto-merge disable event. Equal timestamps defer action. Keep
this policy stateless across restarts. Removing the label revokes future
labeled actions while GitHub retains control of existing auto-merge.

[Label scenarios](../../../test/features/automerge/label.feature) ·
[Command scenarios](../../../test/features/automerge/commands.feature)

## Branch cleanup

Cleanup requires its own configuration and repository auto-merge opt-in.
Preserve default, protected, excluded, and open-PR branches. Measure inactivity
using the latest commit's committer date. Recheck eligibility before deletion
and use `updateRefs.beforeOid` to guard the branch tip atomically. Repository
settings and PR state can change after the final read.

[Cleanup reconciler](../../../internal/cleanup/reconciler.go) ·
[Branch deletion](../../../internal/github/branch_delete.go) ·
[Cleanup safety scenarios](../../../test/features/cleanup/safety.feature)
