# AGENTS.md

## Shared guidance

Use `bin/AGENTS.md` for shared skills and cross-repository defaults.

## Project purpose

Abgleich is a background worker that enables auto-merge or merges ready Dependabot
pull requests and keeps eligible auto-merge branches current. Repositories opt in
through their auto-merge setting. GitHub performs all mutations; the worker does
not clone repositories or execute Git. Each run reconciles fresh remote state.

## Package ownership

| Package | Owns |
| --- | --- |
| `internal/cmd` | Command registration and composition of package modules. |
| `internal/github` | Service, API client, SDK aliases, owner-wide discovery, repository identity, remote access, and response validation. |
| `internal/automerge` | Repository opt-in, PR eligibility, merge and rebase decisions, and registration of its own job. |
| `internal/scheduler` | Reusable named jobs, timing, overlap prevention, logging, and lifecycle. |
| `internal/config` | Composition and injection of package-owned configuration. |
| `internal/health` | Health observers, independent of job outcomes. |

- Constructors belong in the packages owning their types; `cmd` composes their
  modules. Keep dependencies directed from `automerge` to `github` and
  `scheduler`. Neither dependency imports `automerge`.
- Keep owner-wide discovery and case-insensitive repository identity checks in
  `github.Service`. The reconciler receives the service and uses SDK repository
  snapshots, not GitHub configuration or a separately cached repository name.
- Keep accepted-action logging in the service's logging decorator, separate from
  remote access and reconciliation decisions.

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
- Use the pull request's `NodeID` for GraphQL mutations, not its numeric `ID`.
  Recheck eligibility, identity, and branch state before each mutation. Guard
  rebases with `expectedHeadOid` and immediate merges with the expected head SHA.
- Read full repository settings before listing PRs; discovery omits merge
  settings. Skip repositories without auto-merge enabled. Only open, non-draft
  PRs with both branches in the same discovered repository are eligible. Rebase
  PRs with auto-merge enabled; enable squash auto-merge or squash-merge ready PRs
  only for author `dependabot[bot]`. Never approve reviews or bypass rules.
  Attempt at most one mutation per PR per sweep. Finish repository discovery and each
  repository's pull request pagination before mutating;
  continue processing other pull requests after an individual failure.
- HTTP retries apply only to safe methods. Do not automatically retry mutation
  requests; the next run reads current state after failures or ambiguous
  responses. Validate GraphQL errors and matching response identity even on HTTP
  success, and reject comparisons missing `behind_by`.

## Configuration and documentation

- GitHub owner, credentials, and transport settings belong under `github`;
  the current job's interval and timeout are directly under `scheduler`.
  Keep HTTP request, complete job, and health-check timeouts independent.
- Credentials are resolved once at startup through go-service sources. Missing
  or empty credentials prevent startup; replacement requires a process restart.
- Keep operator setup, defaults, recovery, and monitoring guidance in
  `README.md`. Update it when those contracts change, retaining scannable emoji
  headings and concise examples. Keep this file focused on project decisions;
  shared language and workflow rules stay in `bin/AGENTS.md` and its skills.

## Cucumber tests

- Test worker behavior through the Ruby Cucumber/Nonnative harness in `test/`.
  Keep `main_test.go` as the standard `app.Run` launcher; do not add synthetic
  jobs or test-only startup switches there. Use the local fake GitHub API and
  disposable test credentials; tests must not mutate real GitHub repositories.
- Write features in natural domain language with one Given/When/Then
  progression. Avoid interleaving setup, actions, and assertions as a procedural
  script. A Given may establish relevant history, such as a successful check
  followed by repeated failures, and wait for that history to occur.
- Make asynchronous expectations explain the required behavior. For repeated
  failures, wait for at least two failed checks. Use a captured baseline only
  when the behavior requires another event after a specific point; name counts
  and wait targets clearly.
- Reuse the check and health assertions in `test/lib/abgleich/assertions.rb`.
  Add helpers when they clarify a shared domain expectation, and include useful
  branch names or expected outcomes in assertion failures.
- Preserve the states covered by assertions when simplifying steps. Health
  during an outage and after recovery are separate checks; retain both. Keep
  overlap, mutation-count, cancellation, and concurrent-change assertions.

## Validation

- Run commands from the repository root. Use `make help` for discovery and
  `make dep` when dependency setup is needed; CI expectations live in
  `.circleci/config.yml`.
- For feature changes, use the affected selector, such as
  `make features feature=features/automerge/schedule.feature`. The root target
  builds the feature binary before running Cucumber.
- Use `make go-lint package=internal/github` for scoped Go lint,
  `make ruby-lint` for step changes, and `make lint` when both apply. Use the
  relevant repository security checks for security-sensitive changes. CI owns
  complete-suite coverage.
