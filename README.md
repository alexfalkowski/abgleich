[![CircleCI](https://circleci.com/gh/alexfalkowski/abgleich.svg?style=svg)](https://circleci.com/gh/alexfalkowski/abgleich)
[![codecov](https://codecov.io/gh/alexfalkowski/abgleich/graph/badge.svg?token=S9SPVVYQAY)](https://codecov.io/gh/alexfalkowski/abgleich)
[![Go Report Card](https://goreportcard.com/badge/github.com/alexfalkowski/abgleich)](https://goreportcard.com/report/github.com/alexfalkowski/abgleich)
[![Go Reference](https://pkg.go.dev/badge/github.com/alexfalkowski/abgleich.svg)](https://pkg.go.dev/github.com/alexfalkowski/abgleich)
[![Stability: Active](https://masterminds.github.io/stability/active.svg)](https://masterminds.github.io/stability/active.html)

# Abgleich

Abgleich is a background worker that keeps auto-merge pull requests in
[infraops](https://github.com/alexfalkowski/infraops) current with their base branches.

## Background

An open, non-draft pull request is stale when its branch is missing commits from
its base branch. Age does not affect eligibility. Only pull requests with
auto-merge enabled and both branches in the configured repository are updated.

## Server

The worker runs immediately on startup and every five minutes thereafter. It
lists open pull requests through go-github, compares each eligible branch with
its base, and asks GitHub to rebase stale branches using the GraphQL
[`updatePullRequestBranch`](https://docs.github.com/en/graphql/reference/pulls#updatepullrequestbranch)
mutation with `updateMethod: REBASE`. This is the server-side operation used by
GitHub's update-with-rebase UI and CLI. GitHub performs the branch update.

Pull requests are rechecked immediately before the mutation. The mutation
includes `expectedHeadOid`, so concurrent branch changes reject the update and
are discovered on the next run. Conflicts reported by GitHub fail the branch and
require an author to repair it. One failed branch does not prevent other
eligible branches from being processed.

Build with `make build`, then start with:

```sh
./abgleich server -config file:config.yaml
```

The process needs network access to the GitHub REST and GraphQL APIs. It does
not clone repositories, execute Git, or need a writable checkout directory.

## Configuration

For example, `config.yaml` can contain:

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

Owner, repository, and token source are required. The `scheduler`
schedule is optional; omitted or zero interval and timeout values default to
five minutes. Negative durations are invalid. An optional `github.url` overrides
the REST API base URL, defaulting to `https://api.github.com/`. GraphQL uses
`/graphql` on the same origin.

GitHub HTTP transport settings live under `github.client`: timeout, retries,
circuit breaking, and TLS. The timeout defaults to one minute when omitted or
zero; retries and circuit breaking remain disabled unless configured.
Configured retries apply only to safe HTTP methods. GraphQL rebase mutations are
never retried by the HTTP transport; a later scheduled run reads current GitHub
state before deciding whether to request another update. `github.url` continues
to select the API origin, and `github.token` supplies GitHub authentication.

The token needs pull request write access and permission to update the head
repository's contents. Repository rules can still reject a rebase. A source such as
`file:/run/secrets/github-token` supports mounted secrets. Credentials are read
once at startup; missing or empty tokens prevent startup. Restart the worker
after changing or restoring its token. Both REST and GraphQL requests use the
same authenticated client. go-github restricts authentication to the configured
API origin.

## Health

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

## Telemetry

Each completed run produces one log entry with `job`, `duration`, `interval`, and
`overrun`. Success uses `info`, successful overruns use `warn`, and failures use
`error` and include the error, even when they also overrun. Accepted rebase requests
identify the pull request and branch. GraphQL errors are failures even when the HTTP response is
200. A failed branch contributes to the run's failure outcome while other
eligible branches continue to be processed.

The default timeout cancels API requests after five minutes. Work that exceeds
its interval sets `overrun` to `true` when it finishes. An overrun preserves the
handler's success or failure result. Each tick attempts to start work; attempts while the previous run is
active are ignored. Runs never overlap, and missed attempts are not queued.

Scheduled job monitoring uses structured logs. Configure collection and alerts
for repeated failures, exceeded intervals, and missing completed runs in your log
backend. The scheduler does not publish custom job metrics. Standard go-service
HTTP and runtime telemetry remains available through its configured exporters.

## Deployment

Run as a long-lived service with one active instance. Scheduling limits apply
within a process; the expected head OID guards updates from other instances and CI.
Use liveness for restart decisions so a GitHub outage does not cause restart
loops. Shutdown cancels active work and waits for it to stop.

No local checkpoint or database is needed. Every run reconciles current GitHub
state. After a crash, timeout, or ambiguous mutation response, a fresh run
determines whether the remote branch still needs updating. GitHub can complete
an accepted update after the caller stops waiting; retries always compare current
remote state before requesting another rebase. The standard service container
image is sufficient.

## Design

[`automerge.Rebaser`](internal/automerge/rebaser.go) owns the auto-merge
eligibility rules and the decision to rebase stale branches. Its `Rebase`
operation depends on `automerge.GitRepository`, an interface defined beside
its consumer in the domain package.
[`github.Repository`](internal/github/repository.go) implements remote reads,
comparisons, and guarded rebase requests using an injected `github.Client`.
That type aliases the go-github client; the GitHub module constructs it once
through `github.NewClient`. Shared HTTP options and lifecycle dependencies
are injected, following the service client pattern. The package owns
authentication, pagination, and response validation. SDK imports stay inside
`internal/github`.

The [application wiring](internal/cmd/automerge.go) connects the repository to
the rebaser and registers `Rebase` as the `automerge` scheduled job. Domain and
GitHub packages do not depend on scheduling. The
[scheduler](internal/scheduler/job.go) gives each job its own ticker, run timeout,
cancellation, and outcome logs. The module connects `Job.Start` and `Job.Stop`
to the service lifecycle. Each job uses a nonblocking lock to admit one handler
at a time and waits for active work during shutdown. The application injects one
job directly. Additional
scheduled operations can construct their own `scheduler.Job` and register its
`Start` and `Stop` hooks in the application wiring. Handlers must observe
cancellation and return their complete outcome. Health checks are independent
of the scheduler.

The schedule uses `scheduler.interval` and `scheduler.timeout`. Existing
configuration using `scheduler.automerge` or `github.schedule` must move those
interval and timeout values directly under `scheduler`. The job is identified
as `automerge` in telemetry; execution logs use the `job` field.

GitHub API access uses [go-github v93](https://github.com/google/go-github/releases/tag/v93.0.0).

## Development

If you would like to contribute, here is how you can get started.

### Structure

The project follows the structure in [golang-standards/project-layout](https://github.com/golang-standards/project-layout).

### Dependencies

Development requires Go, Ruby, Git for repository setup, and the shared tooling
described by `make help`.

### Style

This project favours the [Uber Go Style Guide](https://github.com/uber-go/guide/blob/master/style.md)

### Setup

Initialize the shared submodule and dependencies, then run the auto-merge features:

```sh
git submodule sync && git submodule update --init
make dep
make features feature=features/automerge
```

Nonnative starts the worker and a local fake GitHub REST and GraphQL API. The
[rebasing](test/features/automerge/rebase.feature),
[recovery](test/features/automerge/recovery.feature), and
[scheduling](test/features/automerge/schedule.feature) features cover branch
eligibility, pagination, conflicts, concurrent changes, retries, timeouts, and
restart recovery. The [credential features](test/features/config/config.feature)
cover startup token validation and replacement; run them with
`make features feature=features/config/config.feature`. Failure scenarios also
verify that process probes remain healthy. Tests use
an accelerated schedule and disposable credentials; they do not modify GitHub.
See [CI](.circleci/config.yml) for the build and release workflow.

### Changes

Release history is available in [GitHub releases](https://github.com/alexfalkowski/abgleich/releases).
