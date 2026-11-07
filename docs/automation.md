# 🔄 Merge automation

[← Back to README](../README.md)

Each sweep discovers public and private repositories accessible to the App
installation, excluding archived, disabled, and foreign repositories. Merge
automation requires **Allow auto-merge**. Only open, non-draft PRs with both
branches in the discovered repository are eligible. Fork PRs are ignored because
fork contributions do not run CI.

For PRs authored by `dependabot[bot]` with both branches in the discovered
repository, Abgleich by default enables squash auto-merge while requirements are
pending or requests a squash merge when ready. Other authors can enable auto-merge
themselves or use the label below.

Configure [Dependabot repository exclusions](configuration.md#-dependabot-settings) to require
explicit opt-in in selected repositories. Excluded repositories still accept
the `automerge` label and `/ab automerge`, including their normal consent checks.
Exclusion does not cancel existing GitHub auto-merge; those branches remain
eligible for rebasing. Disable auto-merge in GitHub separately to stop an existing
request.

Auto-merge PRs are rebased when their branch lacks base-branch commits, regardless
of age, and only when both branches belong to the discovered repository. AI
summaries are enabled separately through [configuration](configuration.md#-summary-settings).

## ⚡ Event-triggered checks

With [webhooks configured](configuration.md#-comment-webhooks), GitHub notifications also prompt
checks of the affected repository. PR changes and branch pushes trigger a check;
terminal commit statuses and completed check runs prompt another assessment after
CI advances. Auto-merge enablement and ready merges use the existing policies.
GitHub [completes enabled auto-merge](https://docs.github.com/en/pull-requests/how-tos/merge-and-close-pull-requests/automatically-merging-a-pull-request)
when required checks and reviews pass.

Event-driven rebasing requires a [configured final CI signal](configuration.md#-event-driven-rebase-settings)
on the current head and no reported active checks or pending commit statuses.
This lets CI reach its own sync-and-push stage first. An individual job finishing
does not establish workflow completion. A missing signal, unreadable or incomplete
observations, and new heads without completed CI defer event rebasing. A failed
final signal still establishes completion; GitHub enforces merge requirements.

A later base update can prompt repair once the head's CI has finished. If CI
already synchronized the branch, the fresh comparison needs no rebase. Startup
and scheduled recovery retain their existing rebase policy and do not require
the final CI signal, including when a failed workflow never reaches its sync job.

Notifications are hints: every check completes fresh installation discovery,
reads current repository settings, and finishes its PR pagination before applying
the same policies as a scheduled sweep. Only the affected repository's PRs are
checked. Bot events are supported; label consent and `hold` still apply.
CI can restart after the final observation; head guards reject concurrent pushes,
but the worker does not lock CI or promise to eliminate competing runs.

One automerge coordinator serializes event checks and scheduled sweeps per
process. Repeated notifications for a queued repository combine into one check.
Events received during its check request one later pass. Pending notifications
are held in memory without a fixed repository limit. An event may wait behind a
running sweep.

Startup and [scheduled sweeps](configuration.md#-auto-merge-schedule) still reconcile all opted-in
repositories, recovering missed events, failures, and pending work lost on restart.
Cleanup, summary, code-review, and security-review jobs retain independent schedules.

## 🏷️ Labeled pull requests

Apply the exact `automerge` label to request squash auto-merge or an immediate
guarded squash merge when ready. The repository must allow both auto-merge and
squash merging. Both branches must belong to the discovered repository;
the label does not permit fork PRs. Eligible branches can be rebased on a later
sweep.

Target-repository label permissions control opt-in, including labels from bots
and workflows. The label authorizes the PR across commits; configure required
reviews and stale-review rules for human approval.

After GitHub disables auto-merge, remove and reapply `automerge` to renew consent,
waiting at least a second between events. Complete label and disable history is
checked on every automation check, including after restart. Missing or invalid
history blocks automation; equal application and revocation timestamps defer it. Removing the
label prevents new requests but does not cancel existing GitHub auto-merge;
disable that separately to stop it.

## ⏸️ Hold a pull request

Provision and apply the exact, case-sensitive `hold` label to pause auto-merge
enablement, immediate merges, rebases, and `/ab automerge` requests on that PR.
It covers Dependabot and labeled PRs, survives restarts, and leaves status, summary,
code-review, and security-review commands available. Rejected commands still
receive reaction feedback.

Removing `hold` resumes normal policy without renewing revoked label consent.
A hold does not cancel existing GitHub auto-merge or accepted requests; disable
auto-merge separately to stop it. Hold is rechecked before acting, but a label
applied after the final read can still race a mutation.

## 🧹 Branch cleanup

When [enabled](configuration.md#-cleanup-settings), cleanup removes merged-PR leftovers and inactive
branches in repositories allowing auto-merge. It preserves default, protected,
excluded, and open-PR head and base branches, including drafts. Merged branches
qualify immediately only while their SHA matches the merged PR; newer commits
follow the inactivity policy. Fork heads outside the repository are untouched.

> [!CAUTION]
> Inactive cleanup can delete unmerged work, including branches without a PR or
> with a closed, unmerged PR. Inactivity uses the latest commit's committer date,
> not the last push time: recently pushing an old commit does not reset its age.
> Protect or exclude long-lived branches before enabling cleanup.

Discovery completes before deletion. The worker rechecks opt-in, protection, and
open PRs, and guards deletion with GraphQL `updateRefs.beforeOid` against concurrent
pushes. Settings or PR changes after the final read can still race deletion.
Failed branches do not block others. See [cleanup recovery](operations.md#-cleanup-recovery).

## 🚀 Server

Each [scheduled sweep](configuration.md#-auto-merge-schedule) starts with fresh discovery and
processes repositories serially, with PRs oldest first. All repository pages must
succeed before processing; full repository settings and all PR pages are read
before mutating that repository. Discovery omits merge settings.

Rebases use GraphQL `updatePullRequestBranch` with `updateMethod: REBASE`;
auto-merge uses `enablePullRequestAutoMerge` with `mergeMethod: SQUASH`; immediate
merges use REST with `merge_method: squash`. Immediate readiness requires
`mergeable: true` and `mergeable_state: clean` or `unstable`: optional check
failures permit merging, while unknown readiness or blocking requirements defer
it. GitHub enforces required checks, reviews, and rules; merge automation never
approves reviews or bypasses rules. Disabled squash merging produces an operator-visible error
rather than a fallback method. Other authors' auto-merge PRs retain their method.

Each PR gets at most one mutation attempt per scheduled or event check.
Eligibility is rechecked immediately before acting; `expectedHeadOid` guards rebases and auto-merge, and
`sha` guards immediate merges. Concurrent changes are reconsidered next run.
Conflicts require author repair; individual failures do not block other PRs or
repositories. The worker needs REST and GraphQL access, with no writable checkout.

### 🚦 API usage

Polling, event checks, and commands share the installation's API budget. The
auto-merge interval controls polling volume; event frequency also affects usage.
Scheduled review maintenance also discovers repositories and open PRs on each
feature's interval. Manual review notifications perform only request-specific reads
and inference, leaving that maintenance cadence unchanged.
Event checks still discover all repository pages, but list PRs only in the
affected repository. go-github supplies built-in rate-limit checks without
additional configuration. Throttling stops the current check; a later event or
scheduled run reads fresh state.
