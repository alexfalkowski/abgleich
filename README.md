[![CircleCI](https://circleci.com/gh/alexfalkowski/abgleich.svg?style=svg)](https://circleci.com/gh/alexfalkowski/abgleich)
[![codecov](https://codecov.io/gh/alexfalkowski/abgleich/graph/badge.svg?token=S9SPVVYQAY)](https://codecov.io/gh/alexfalkowski/abgleich)
[![Go Reference](https://pkg.go.dev/badge/github.com/alexfalkowski/abgleich.svg)](https://pkg.go.dev/github.com/alexfalkowski/abgleich)
[![Stability: Active](https://masterminds.github.io/stability/active.svg)](https://masterminds.github.io/stability/active.html)

# 🔄 Abgleich

Abgleich is a background worker that enables auto-merge or merges ready
Dependabot and labeled pull requests, and keeps auto-merge branches current
across a GitHub account's opted-in repositories.

## 🎯 Background

An open, non-draft pull request is stale when its branch is missing commits from
its base branch. Age does not affect eligibility. Only pull requests with
auto-merge enabled and both branches in the same discovered repository are rebased.
Public and private repositories visible to the owner's token are discovered on
each run. Archived, disabled, foreign, and repositories without **Allow auto-merge**
enabled are skipped entirely. That repository setting is the opt-in for the worker.

For Dependabot PRs, the worker also enables squash auto-merge while requirements
are pending, or requests a squash merge when GitHub reports the PR ready. It
identifies Dependabot by the PR author `dependabot[bot]`, independent of branch
names. This automatic Dependabot policy excludes draft, closed, and fork PRs.
Other authors can enable auto-merge themselves or opt in with a label.

### 🏷️ Labeled pull requests

Apply the exact label `automerge` to an open, non-draft PR to request squash
auto-merge. If GitHub reports the PR ready, the worker requests a guarded squash
merge immediately. The repository must allow auto-merge and squash merging.

This policy supports PRs from forks as well as branches in the repository. The
base must belong to the discovered repository. The worker never rebases fork
branches; their authors keep them current. Labeled branches in the same
repository can be rebased on a later sweep once auto-merge is enabled.

Label permissions in the target repository control who can opt a PR in. Forking
or opening a PR grants no labeling permission. Authorized bots and workflows can
also apply labels, so configure them accordingly. The label requests automation
for the PR; it is not an approval tied to a particular commit. Configure GitHub's
required reviews and stale-review rules when changes need human approval.

If GitHub disables auto-merge, including after an external contributor pushes,
the existing label cannot restore it. Remove and reapply `automerge` to renew the
request. Wait at least a second between removing and reapplying the label so
the event timestamps are unambiguous. The worker reads complete label and
disable-event history on each run, including after restart. Missing or invalid
history prevents automation; equal application and revocation timestamps defer it.

Removing the label prevents a new request; it does not cancel auto-merge already
enabled on GitHub. To stop that merge, disable auto-merge in GitHub as well.

## 🚀 Quickstart

Run these steps from the repository root.

### ✅ Prerequisites

- Go matching [go.mod](go.mod).
- Git with GitHub SSH access for the `bin/` submodule.
- Ruby with Bundler for dependency setup and the Cucumber harness.
- Make, plus `curl` for the readiness check below.

### 🏗️ Bootstrap and build

Initialize the shared tooling, install dependencies, and build the worker:

```sh
git submodule sync
git submodule update --init
make dep
make build
```

### ▶️ Configure and start

Save the example in **Configuration** below as `config.yaml`, choosing the GitHub
personal account to manage. Supply that account's token with the permissions described
under **Credentials**. The example reads it from `GITHUB_TOKEN`.

If `GITHUB_TOKEN` is already exported in your shell, start the worker directly:

```sh
./abgleich server -config file:config.yaml
```

The `github.token: env:GITHUB_TOKEN` setting tells the worker to read the exported
environment variable once at startup.

If you need to set the token first, you can use this optional hidden prompt in
Bash or Zsh, then run the startup command above:

```bash
printf 'GitHub token: '
read -r -s GITHUB_TOKEN
printf '\n'
export GITHUB_TOKEN
```

The worker begins checking the account's repositories immediately. Leave it
running, and use another terminal to verify readiness:

```sh
curl --fail --silent --show-error http://localhost:11000/abgleich/readyz
```

An HTTP 200 response with body `SERVING` confirms process readiness. Check the
worker's structured logs for GitHub check outcomes; readiness is independent of
whether a branch check succeeds.

## 🚀 Server

The worker runs immediately on startup and, by default, every fifteen minutes
thereafter. It discovers all owned repositories before processing them serially.
For each repository, it lists open pull requests through go-github, compares each eligible
branch with its base, and asks GitHub to rebase stale branches using the GraphQL
[`updatePullRequestBranch`](https://docs.github.com/en/graphql/reference/pulls#updatepullrequestbranch)
mutation with `updateMethod: REBASE`. This is the server-side operation used by
GitHub's update-with-rebase UI and CLI. GitHub performs the branch update.

Before listing PRs, the worker reads the full repository settings; GitHub's
repository list omits merge settings. Dependabot and labeled automation use GraphQL
[`enablePullRequestAutoMerge`](https://docs.github.com/en/graphql/reference/pulls#enablepullrequestautomerge)
with `mergeMethod: SQUASH` and `expectedHeadOid`, or the REST merge API with
`merge_method: squash` and the expected head SHA. Immediate merges require `mergeable: true` and
`mergeable_state: clean`; unknown or failing check states never trigger an
immediate merge. GitHub still enforces checks, reviews, and repository rules.
The worker does not approve reviews or request rule bypasses.

Each sweep attempts at most one mutation per PR. After enabling auto-merge,
a later sweep can rebase a stale branch. If squash merging is disabled, the
merge action fails with an operator-visible error instead of choosing
another merge method. Other authors' auto-merge PRs retain their selected method.

Pull requests are rechecked immediately before each mutation. Rebases and
auto-merge enablement include `expectedHeadOid`, and immediate merges include `sha`, so concurrent branch changes
reject those requests and are discovered on the next run. Conflicts reported by GitHub fail the branch and
require an author to repair it. One failed branch does not prevent other
eligible branches or repositories from being processed. After a timeout, the
next scheduled sweep starts with fresh repository discovery.

Discovery validates that the authenticated personal account matches `github.owner`.
All repository pages must succeed before processing begins. Pull request pagination
also finishes before any branches in that repository are updated.

The process needs network access to the GitHub REST and GraphQL APIs. It does
not clone repositories, execute Git, or need a writable checkout directory.

## ⚙️ Configuration

The `github` and `health` blocks are required. Both `health.duration` and
`health.timeout` must be positive durations; they have no defaults. This example
also includes shared go-service settings for identifiers, logging, metrics, and
the HTTP listener.

Save it as `config.yaml` and adjust `github.owner`:

```yaml
environment: production
github:
  client:
    timeout: 1m
    retry:
      attempts: 2
      backoff: 100ms
  owner: alexfalkowski
  token: env:GITHUB_TOKEN
scheduler:
  interval: 15m
  timeout: 5m
health:
  duration: 1s
  timeout: 1s
id:
  kind: uuid
telemetry:
  logger:
    kind: json
    level: info
  metrics:
    kind: prometheus
transport:
  http:
    address: tcp://:11000
    timeout: 5s
```

Set `github.owner` and `github.token`. The schedule and
HTTP client settings are optional:

| Setting | Default | Purpose |
| --- | --- | --- |
| `scheduler.interval` | `15m` | Time between owner-wide sweeps. |
| `scheduler.timeout` | `5m` | Deadline for a complete owner-wide sweep. |
| `github.client.timeout` | `1m` | Deadline for an individual HTTP request. |
| `github.url` | `https://api.github.com/` | REST API origin; GraphQL uses `/graphql` on the same origin. |

Omitted or zero scheduler durations and GitHub client timeouts use the defaults
above. Negative schedule durations are invalid.

### 🔌 HTTP transport

GitHub HTTP transport settings live under `github.client`: timeout, retries,
circuit breaking, and TLS. Retries and circuit breaking remain disabled unless configured.
Configured retries apply only to safe HTTP methods. Rebase, auto-merge, and merge
mutations are never retried by the HTTP transport; a later scheduled run reads
current GitHub state before deciding whether another action is needed.

### 🚦 API usage

One shared go-github client processes repositories and pull requests serially.
The fifteen-minute interval limits polling volume; there is no additional rate-limit
library or configuration. If GitHub reports throttling, the current sweep stops
instead of continuing through other repositories. go-github retains its built-in
rate-limit checks, and a later scheduled run reads current state again.

Personal access tokens share the account's API budget with other automation.
Monitor repeated rate-limit failures and increase `scheduler.interval` if needed.

### 🔐 Credentials

Use a personal access token belonging to `github.owner`, covering all repositories
to manage, including private and newly created repositories. A fine-grained token
needs **Pull requests: read and write** and **Contents: read and write**.
Enable **Allow auto-merge** on each repository to opt in, and **Allow squash
merging** for Dependabot and labeled automation. Required checks and reviews must still pass;
repository rules can reject a rebase or merge. Repositories outside the token's
access cannot be discovered.
A source such as
`file:/run/secrets/github-token` supports mounted secrets. Both REST and GraphQL
requests use the same authenticated client. go-github restricts authentication
to the configured API origin.

> [!IMPORTANT]
> Credentials are read once at startup. Missing or empty tokens prevent startup.
> Restart the worker after changing or restoring its token.

## 🩺 Health

The service exposes HTTP observability endpoints:

| Endpoint | Behavior |
| --- | --- |
| `/abgleich/healthz` | Shared external connectivity check. |
| `/abgleich/readyz` | Process readiness, independent of scheduled job outcomes. |
| `/abgleich/livez` | Process liveness, independent of GitHub failures. |
| `/abgleich/metrics` | Prometheus metrics when that exporter is configured. |

Job failures, conflicts, and timeouts do not affect these health checks. Infraops
probes the private readiness and liveness endpoints directly, including for
workers without a Service or Ingress.

`health.duration` controls how frequently health state is sampled.
`health.timeout` bounds the shared connectivity check; it is independent of the
scheduled job's run timeout.

## 📊 Telemetry

Each completed run produces one log entry with `job`, `duration`, `interval`, and
`overrun`. The auto-merge job uses `job: automerge`.

| Outcome | Log level | Details |
| --- | --- | --- |
| Success | `info` | The check completed within its interval. |
| Successful overrun | `warn` | The check succeeded but exceeded its interval. |
| Failure | `error` | Includes the error, even when the check also overran. |

Accepted rebase requests, enabled auto-merges, and confirmed merges identify the
repository, pull request, and branch. Their messages are `requested pull request rebase`,
`enabled pull request auto-merge`, and `merged pull request`, respectively. Failures
include repository names and pull request numbers where available. GraphQL errors
are failures even when the HTTP response is 200. A failed branch contributes to
the run's failure outcome while other eligible branches continue to be processed.

- **Timeouts:** the default run timeout cancels active work after five minutes.
- **Overruns:** work that exceeds its interval sets `overrun` to `true` when it
  finishes, preserving the handler's success or failure result.
- **Overlap:** each tick attempts to start work. Attempts while the previous run
  is active are ignored; missed attempts are not queued.

Scheduled job monitoring uses structured logs. Configure collection and alerts
for repeated failures, exceeded intervals, and missing completed runs in your log
backend. The scheduler does not publish custom job metrics. Standard go-service
HTTP and runtime telemetry remains available through its configured exporters.

## 📦 Deployment

- **Instances:** run as a long-lived service with one active instance. Scheduling
  limits apply within a process; the expected head OID guards updates from other instances and CI.
- **Restarts:** use liveness for restart decisions so a GitHub outage does not cause restart loops.
- **Shutdown:** active work is cancelled, and the worker waits for it to stop.

### 🧯 Recovery

No local checkpoint or database is needed. Every run reconciles current GitHub
state. After a crash, timeout, or ambiguous mutation response, a fresh run
determines whether the remote branch still needs updating. GitHub can complete
an accepted update after the caller stops waiting; retries always compare current
remote state before requesting another rebase. The standard service container
image is sufficient.

## 🧩 Design

| Package | Responsibility |
| --- | --- |
| [`automerge`](internal/automerge/reconciler.go) | Repository opt-in, strategy selection and ordering, and job registration. |
| [`automerge/internal/strategy`](internal/automerge/internal/strategy/doc.go) | Dependabot, label, and rebase policies with fresh pre-mutation checks. |
| [`github`](internal/github/service.go) | Owner-wide repository discovery, identity checks, authenticated API access, pagination, and response validation. |
| [`scheduler`](internal/scheduler/scheduler.go) | Independent schedules, timeouts, overlap protection, outcome logs, and job lifecycle. |
| [`cmd`](internal/cmd/module.go) | Composition of package modules. |

Dependencies follow the responsibilities:

```mermaid
flowchart LR
    automerge --> github
    automerge --> scheduler
```

`automerge.Reconciler.Reconcile` asks `github.Service` for all repositories, then uses
that service for repository settings, PR reads, identity checks, comparisons,
auto-merge enablement, and guarded rebase and merge requests. `github.Repository`
and pull request types alias SDK snapshots;
they contain no service behavior. The service uses an injected `github.Client`,
which aliases the go-github client. The GitHub module constructs it once through
`github.NewClient`, with shared HTTP options and lifecycle dependencies injected.
SDK imports stay inside `internal/github`.

The strategy module provides the policies in this order: Dependabot, label,
rebase. The reconciler receives the ordered slice through DI and selects the first match.
A selected strategy owns the PR for that sweep, including no-op and error
outcomes. The strategy interface, implementations, and shared PR
eligibility helpers live in `internal/automerge/internal/strategy`.
Each strategy owns its applicability and fresh pre-mutation checks.
Label authorization uses paginated GraphQL history; GitHub field aliases
decode events into SDK issue-event snapshots.

Logging is composed around operations. `github.NewService` wraps the API service
with a logging decorator to report accepted actions after response validation.
`Scheduler.Register` wraps each handler to report its outcome and overruns
within the job's timeout.

### 🗓️ Job registration

Each package registers construction in its own module. The
[auto-merge module](internal/automerge/module.go) constructs the reconciler and
registers `Reconcile` as the `automerge` job. Other feature packages can register
their own named handlers and schedules through `Scheduler.Register`. Registration
finishes before startup, and duplicate names prevent application startup.
Scheduler has no feature-package dependencies.

The [scheduler module](internal/scheduler/module.go) connects `Scheduler.Start`
and `Scheduler.Stop` to the service lifecycle. Each
[job](internal/scheduler/job.go) has its own ticker, run timeout, cancellation,
and outcome logs. Jobs run independently and use a nonblocking lock to admit
one run per job at a time. Shutdown cancels all jobs before waiting for active
work. Handlers must observe cancellation and return their complete outcome.
Health checks are independent of the scheduler.

GitHub API access uses [go-github v93](https://github.com/google/go-github/releases/tag/v93.0.0).

## 🛠️ Development

If you would like to contribute, here is how you can get started.

### 🗂️ Structure

The project follows the structure in [golang-standards/project-layout](https://github.com/golang-standards/project-layout).

### ✍️ Style

This project favours the [Uber Go Style Guide](https://github.com/uber-go/guide/blob/master/style.md)

### 🏁 Setup

After completing the quickstart's bootstrap steps, run the auto-merge features:

```sh
make features feature=features/automerge
```

Use `make help` to discover the other development commands.

Nonnative starts the worker and a local fake GitHub REST and GraphQL API:

- [Rebasing](test/features/automerge/rebase.feature) covers eligibility,
  pagination, conflicts, and concurrent changes.
- [Dependabot](test/features/automerge/dependabot.feature) covers repository opt-in,
  enabling auto-merge, guarded squash merges, exclusions, and failure recovery.
- [Labels](test/features/automerge/label.feature) covers labeled fork and repository
  PRs, authorization renewal, complete history, and concurrent changes.
- [Repositories](test/features/automerge/repositories.feature) covers owner-wide
  discovery, private repositories, failure isolation, throttling, and timeout recovery.
- [Recovery](test/features/automerge/recovery.feature) covers GitHub failures,
  retries, and ambiguous mutation responses.
- [Scheduling](test/features/automerge/schedule.feature) covers timeouts,
  overlap prevention, and restart recovery.
- [Credentials](test/features/config/config.feature) covers startup token
  validation and replacement. Run it with `make features feature=features/config/config.feature`.

Failure scenarios also verify that process probes remain healthy. Tests use an
accelerated schedule and disposable credentials; they do not modify GitHub.
See [CI](.circleci/config.yml) for the build and release workflow.

### 📝 Changes

Release history is available in [GitHub releases](https://github.com/alexfalkowski/abgleich/releases).
