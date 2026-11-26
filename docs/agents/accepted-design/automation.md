# GitHub automation and jobs

Part of the [accepted-design index](../accepted-design.md).

## Comment deliveries

Each worker serializes comment handlers so overlapping deliveries observe
the results of earlier actions, including an automerge label added for a
duplicate delivery. Check fresh write/admin permission before a delivery can
wait for a busy slot. Bound admission and waiting with a child deadline that
reserves processing time. Keep the handler's authorization checks before
acting. Locally budgeted admission calls and best-effort reactions stay outside
the shared GitHub circuit breaker.
Bound busy-slot permission checks per worker by both rate and concurrency.
Return 503 before a GitHub lookup when either limit is exhausted.
Summary and review jobs process accepted work independently.
[Comment webhook behavior](../../commands.md#-webhook-delivery)

## Background jobs

- Feature packages register their own named handlers through `Scheduler.Register`
  in their modules before startup. Names must be unique; the scheduler stays
  independent of feature packages.
- Jobs run immediately and then on independent schedules. Preserve per-job
  timeouts and overlap protection; missed starts are skipped rather than queued.
  Handlers must observe cancellation and return their complete outcome.
- Use `go-sync` for synchronization and error aggregation. Shutdown cancels all
  jobs before waiting for them and preserves failures with their job names.
  Keep lifecycle ordering such that jobs stop before client transports close.
- Job failures, conflicts, and timeouts must not make readiness or liveness
  fail. Preserve structured job outcome and overrun logs for monitoring.

## GitHub integration

- Keep direct go-github imports inside `internal/github`. Reuse SDK types through
  project-owned aliases and their pointer/getter APIs instead of duplicating
  models and conversion code. Use go-service HTTP constants where available.
- Read pull requests through REST and request server-side rebases through the
  GraphQL `updatePullRequestBranch` mutation with `updateMethod: REBASE`. The REST
  branch-update operation merges; the custom GraphQL payload types remain local.
- Use the pull request's `NodeID` for GraphQL operations, not its numeric `ID`.
  Recheck eligibility, identity, and branch state before each mutation. Guard
  rebases and auto-merge enablement with `expectedHeadOid`, and immediate merges
  with the expected head SHA.
- Read full repository settings before listing PRs; discovery omits merge
  settings. The auto-merge job skips repositories without auto-merge enabled.
  Only open, non-draft PRs targeting the discovered repository are eligible.
  Evaluate strategies in Dependabot, label, rebase order; stop after the first
  match, including errors and no-op outcomes. Automatic Dependabot merging and
  rebasing require both branches in the discovered repository. The exact
  `automerge` label and `/ab automerge` have the same requirement: ignore fork PRs
  because fork contributions do not run CI. Status explains this exclusion;
  summaries, reviews, and review-thread maintenance also ignore forks. Existing GitHub
  auto-merge on forks is left unchanged. Merge automation never approves reviews
  or bypasses rules.
  Attempt at most one mutation per PR per sweep. Finish repository discovery and each
  repository's pull request pagination before mutating;
  continue processing other pull requests after an individual failure.
- The exact `hold` label pauses auto-merge enablement, immediate merges, rebases,
  and `/ab automerge` label requests. Recheck it before acting. It does not cancel
  existing GitHub auto-merge or disable status, summary, and review commands.
- Read complete label and auto-merge-disable history before labeled actions.
  A label must be newer than its removal and the latest auto-merge disable event;
  equal timestamps defer automation. Removing a label does not cancel an
  existing GitHub auto-merge. Keep this policy stateless across restarts.
- HTTP retries apply only to safe methods. Do not automatically retry mutation
  requests; the next run reads current state after failures or ambiguous
  responses. Validate GraphQL errors and matching response identity even on HTTP
  success, and reject comparisons missing `behind_by`.
