[![CircleCI](https://circleci.com/gh/alexfalkowski/abgleich.svg?style=svg)](https://circleci.com/gh/alexfalkowski/abgleich)
[![codecov](https://codecov.io/gh/alexfalkowski/abgleich/graph/badge.svg?token=S9SPVVYQAY)](https://codecov.io/gh/alexfalkowski/abgleich)
[![Go Reference](https://pkg.go.dev/badge/github.com/alexfalkowski/abgleich.svg)](https://pkg.go.dev/github.com/alexfalkowski/abgleich)
[![Stability: Active](https://masterminds.github.io/stability/active.svg)](https://masterminds.github.io/stability/active.html)

# 🔄 Abgleich

Abgleich is a background worker that keeps auto-merge pull requests in
[infraops](https://github.com/alexfalkowski/infraops) current with their base branches.

## 🎯 Background

An open, non-draft pull request is stale when its branch is missing commits from
its base branch. Age does not affect eligibility. Only pull requests with
auto-merge enabled and both branches in the configured repository are updated.

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
owner and repository to manage. Supply a token with the permissions described
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

The worker begins checking the configured repository immediately. Leave it
running, and use another terminal to verify readiness:

```sh
curl --fail --silent --show-error http://localhost:11000/abgleich/readyz
```

An HTTP 200 response with body `SERVING` confirms process readiness. Check the
worker's structured logs for GitHub check outcomes; readiness is independent of
whether a branch check succeeds.

## 🚀 Server

The worker runs immediately on startup and, by default, every five minutes
thereafter. It lists open pull requests through go-github, compares each eligible
branch with its base, and asks GitHub to rebase stale branches using the GraphQL
[`updatePullRequestBranch`](https://docs.github.com/en/graphql/reference/pulls#updatepullrequestbranch)
mutation with `updateMethod: REBASE`. This is the server-side operation used by
GitHub's update-with-rebase UI and CLI. GitHub performs the branch update.

Pull requests are rechecked immediately before the mutation. The mutation
includes `expectedHeadOid`, so concurrent branch changes reject the update and
are discovered on the next run. Conflicts reported by GitHub fail the branch and
require an author to repair it. One failed branch does not prevent other
eligible branches from being processed.

The process needs network access to the GitHub REST and GraphQL APIs. It does
not clone repositories, execute Git, or need a writable checkout directory.

## ⚙️ Configuration

The `github` and `health` blocks are required. Both `health.duration` and
`health.timeout` must be positive durations; they have no defaults. This example
also includes shared go-service settings for identifiers, logging, metrics, and
the HTTP listener.

Save it as `config.yaml` and adjust `github.owner` and `github.repository`:

```yaml
environment: production
github:
  client:
    timeout: 1m
    retry:
      attempts: 2
      backoff: 100ms
  owner: alexfalkowski
  repository: infraops
  token: env:GITHUB_TOKEN
scheduler:
  interval: 5m
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

Set `github.owner`, `github.repository`, and `github.token`. The schedule and
HTTP client settings are optional:

| Setting | Default | Purpose |
| --- | --- | --- |
| `scheduler.interval` | `5m` | Time between scheduled checks. |
| `scheduler.timeout` | `5m` | Deadline for a complete check. |
| `github.client.timeout` | `1m` | Deadline for an individual HTTP request. |
| `github.url` | `https://api.github.com/` | REST API origin; GraphQL uses `/graphql` on the same origin. |

Omitted or zero scheduler durations and GitHub client timeouts use the defaults
above. Negative schedule durations are invalid.

### 🔌 HTTP transport

GitHub HTTP transport settings live under `github.client`: timeout, retries,
circuit breaking, and TLS. Retries and circuit breaking remain disabled unless configured.
Configured retries apply only to safe HTTP methods. GraphQL rebase mutations are
never retried by the HTTP transport; a later scheduled run reads current GitHub
state before deciding whether to request another update.

### 🔐 Credentials

The token needs pull request write access and permission to update the head
repository's contents. Repository rules can still reject a rebase. A source such as
`file:/run/secrets/github-token` supports mounted secrets. Both REST and GraphQL
requests use the same authenticated client. go-github restricts authentication
to the configured API origin.

> [!IMPORTANT]
> Credentials are read once at startup. Missing or empty tokens prevent startup.
> Restart the worker after changing or restoring its token.

### ♻️ Schedule configuration

Existing configuration using `scheduler.automerge` or `github.schedule` must move
the interval and timeout values directly under `scheduler`.

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

Accepted rebase requests identify the pull request and branch. GraphQL errors
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
| [`automerge`](internal/automerge/rebaser.go) | Eligibility rules, rebase decisions, and registration of the auto-merge job. |
| [`github`](internal/github/repository.go) | Pull request snapshots, authenticated API access, pagination, and response validation. |
| [`scheduler`](internal/scheduler/scheduler.go) | Independent schedules, timeouts, overlap protection, outcome logs, and job lifecycle. |
| [`cmd`](internal/cmd/module.go) | Composition of package modules. |

Dependencies follow the responsibilities:

```mermaid
flowchart LR
    automerge --> github
    automerge --> scheduler
```

`automerge.Rebaser.Rebase` uses `github.Repository` for remote reads, comparisons,
and guarded rebase requests. The repository uses an injected `github.Client`,
which aliases the go-github client. The GitHub module constructs it once through
`github.NewClient`, with shared HTTP options and lifecycle dependencies injected.
SDK imports stay inside `internal/github`.

### 🗓️ Job registration

Each package registers construction in its own module. The
[auto-merge module](internal/automerge/module.go) constructs the rebaser and
registers `Rebase` as the `automerge` job. Other feature packages can register
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
