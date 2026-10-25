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
Public and private repositories accessible to the App installation are discovered on
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

### ⏸️ Hold a pull request

Apply the exact label `hold` to pause Abgleich on one PR. Maintainers apply and
remove it using their existing GitHub permissions. Matching is case-sensitive;
the label name is fixed and needs no worker configuration. Ensure the managed
repository baseline has provisioned `hold` before using it.

After observing the label, Abgleich skips auto-merge enablement, immediate merges,
rebases, and `/ab automerge` label requests for that PR, including Dependabot and
labeled fork PRs. Comment commands retain receipt and rejection reactions. Other
PRs continue normally. A current hold survives worker restarts because each run
reads GitHub state.

Removing the hold resumes normal policy. Dependabot can be enrolled again;
removing hold does not renew a labeled request revoked by auto-merge disablement
or removal of `automerge`.

The hold suppresses Abgleich requests only. It does not disable GitHub auto-merge
or undo an accepted request. Disable auto-merge in GitHub separately when it is
already enabled and must stop. The worker rechecks hold membership before acting,
but a label applied after the final read can still race a mutation.

### 💬 Comment requests

With GitHub App authentication and webhooks configured, post a new PR comment
containing exactly `/ab automerge`. Surrounding whitespace is allowed.
Mention-based commands are no longer supported; use `/ab automerge` to avoid
notifying GitHub users.
Commands are case-sensitive; extra text and unknown commands are ignored without reactions.
The worker checks the author's current write or admin permissions, re-reads the
comment and PR, and adds `automerge`. The App reacts on the original command:

| Reaction | Meaning |
| --- | --- |
| 👀 | The signed command was received and processing started. |
| 👍 | The request was accepted: the label was added or automation was already enabled or labeled. |
| 😕 | The request was ineligible, rejected, or failed. |

Acceptance confirms command processing; the PR's status shows when merging finishes.
The label remains the durable request for automation;
the next scheduled sweep applies the existing merge policy. Fork PRs are supported.
Edited comments and bot authors do not authorize requests.

An existing label or enabled auto-merge needs no label write. The App still checks
the current comment and the author's permission before acknowledging acceptance.
A delivery whose original comment is at or before a label removal or auto-merge disablement
cannot add the label. Missing history prevents action. After a revocation, remove
any remaining label and post a new command with a later timestamp to renew the
request. These checks survive restarts without local storage. GitHub label writes
do not support an expected-state guard: a simultaneous manual change can still
race the final write. This is not an exactly-once delivery guarantee.

Reactions are best effort and do not control automation. Receipt feedback gets at
most one second and a quarter of the remaining delivery budget, so slow feedback
leaves time for the label operation. Final acceptance or rejection feedback uses
the remaining delivery budget. An outage or exhausted deadline can leave feedback
incomplete; reaction failures are logged.
Reactions remain on the comment, and GitHub reuses an existing reaction on
redelivery. A later attempt can therefore leave both rejection and acceptance
reactions; consult the label and current PR state for the outcome.

### 🔎 Explain a pull request's automation status

With the same App and webhook setup, post a new comment containing exactly
`/ab status`. The requester needs current write or admin access. Surrounding
whitespace is allowed; arguments, edited comments, and bot authors are ignored.
The App replies with its observed policy decision, reason, and next step,
including the head commit, UTC assessment time, GitHub auto-merge state, and a
link to the request. Receipt and outcome reactions use the same delivery budget;
acceptance means the status reply was confirmed published.

Status explains repository opt-out, drafts, closed PRs, `hold`, squash-merge
settings, label authorization, and the applicable Dependabot, label, or rebase
strategy. For example, a label that predates auto-merge disablement reports
**Waiting for renewed authorization** and explains how to remove and reapply it.
Forks can use labeled merging, but their authors must keep their branches current.

The reply describes what the worker would attempt under the observed state.
It never labels, enables auto-merge, merges, rebases, approves, or bypasses rules.
GitHub still enforces merge requirements. Coarse mergeability does not identify
individual failing checks or missing reviews; consult GitHub's merge box for
those details. Unavailable history or comparison evidence is reported explicitly.

GitHub reads are separate observations, not an atomic snapshot. A detected
change to the revision or policy state rejects the request without a reply
(HTTP 409); post a new `/ab status` request to retry. State can still change after
the final check. Publication is best effort within the configured command
deadline and is never retried automatically. A failed or timed-out response can
already have published a reply, and webhook redelivery can produce another
comment with the same request link. No exactly-once publication is promised.

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

Save the example in **Configuration** below as `config.yaml`. Set the owner,
App ID, installation ID, and path to the App's RSA private key. The key uses a
go-service source such as `file:/etc/secrets/abgleich/github-app-private-key.pem`
and is read once at startup. App registration and permissions are described
under **GitHub App setup**.

Start the worker:

```sh
./abgleich server -config file:config.yaml
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
For each repository, it lists open pull requests through go-github and processes
them oldest first by creation time. It compares each eligible branch with its base
and asks GitHub to rebase stale branches using the GraphQL
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

Discovery validates that the App installation belongs to `github.owner`.
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
  app:
    id: 5427450
    installation_id: 174152088
    private_key: file:/etc/secrets/abgleich/github-app-private-key.pem
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
    timeout: 9s
    max_receive_size: 1048576B
```

Set `github.owner` and `github.app` for all GitHub access. Add
`github.webhook` for comment commands. The schedule and HTTP client settings
are optional:

| Setting | Default | Purpose |
| --- | --- | --- |
| `scheduler.interval` | `15m` | Time between owner-wide sweeps. |
| `scheduler.timeout` | `5m` | Deadline for a complete owner-wide sweep. |
| `github.client.timeout` | `1m` | Deadline for an individual HTTP request. |
| `github.url` | `https://api.github.com/` | REST API origin; GraphQL uses `/graphql` on the same origin. |
| `github.webhook.timeout` | `5s` | Complete synchronous delivery budget; must be positive and below `10s`. |
| `transport.http.max_receive_size` | `4MB` | Incoming request body limit for all HTTP routes; the example sets 1 MiB. |

Omitted or zero scheduler durations and GitHub client timeouts use the defaults
above. Negative schedule durations are invalid.

### 🔌 HTTP transport

GitHub HTTP transport settings live under `github.client`: timeout, retries,
circuit breaking, and TLS. Retries and circuit breaking remain disabled unless configured.
Configured retries apply only to safe HTTP methods. Rebase, auto-merge, and merge
mutations are never retried by the HTTP transport; a later scheduled run reads
current GitHub state before deciding whether another action is needed.

### 🚦 API usage

The shared go-github client processes repositories and pull requests serially.
The fifteen-minute interval limits polling volume; there is no additional rate-limit
library or configuration. If GitHub reports throttling, the current sweep stops
instead of continuing through other repositories. go-github retains its built-in
rate-limit checks, and a later scheduled run reads current state again.

Polling and comment commands share the installation's API budget. Monitor repeated
rate-limit failures and increase `scheduler.interval` if needed.

### 🔐 Credentials

The worker and comment handler share one App installation. Configure
`github.app.id`, `github.app.installation_id`, and `github.app.private_key`;
`github.owner` must match the installation's account. A personal access token
is not used. `ghinstallation` signs App requests and caches and renews the
installation access token. REST and GraphQL use the same installation credentials;
App clients reject redirects to another API origin.

The worker validates the installation's App ID, owner, and suspension state once
per sweep and once per comment command, before repository access. It discovers
only repositories granted to that installation, including private repositories.
Install the App on all owned repositories to include new repositories
automatically, or select repositories explicitly.
Enable **Allow auto-merge** on each repository to opt in, and **Allow squash
merging** for Dependabot and labeled automation. Required reviews and checks still
apply; give the App no ruleset bypass.

### 🤖 GitHub App setup

Register the App with **Contents: read and write**, **Pull requests: read and
write**, and **Issues: read and write**. Metadata read access is automatic. The worker
uses server-side rebases and squash merges; the comment handler adds the existing
`automerge` label and reactions. Issues write access is required for comment
reactions. When upgrading an existing App, update its permissions and approve
the installation's permission change. Use your own App and installation IDs in
the configuration.

Omit `github.webhook` to run the worker without exposing a webhook endpoint.
To receive comments, add this block to the existing `github` configuration:

```yaml
github:
  webhook:
    secret: file:/etc/secrets/abgleich/github-webhook-secret
    timeout: 8s
```

The webhook secret uses go-service sources such as `file:` or `env:`. Keep HTTP
request, webhook, job, and health timeouts independent; the server HTTP timeout
must accommodate the webhook budget. The explicit `8s` budget gives permission
checks, labeling, and final feedback room to complete within the example's `9s`
HTTP timeout and GitHub's ten-second delivery deadline.

To enable comment commands, configure the App's **Issue comment** subscription,
a random webhook secret matching `github.webhook.secret`, and an HTTPS webhook
URL ending in `/webhooks/github`. The proposed deployment URL is
`https://abgleich.lean-thoughts.com/webhooks/github`; DNS and ingress must be
provisioned before activation. The handler requires SHA-256 signatures. The shared
HTTP transport enforces `transport.http.max_receive_size`; set `1048576B` for a
1 MiB limit as shown above. It acknowledges ignored and completed deliveries with
HTTP 204, rejects invalid signatures with 401 and unauthorized commands with 403,
and returns non-success for API failures, conflicts, or processing timeouts.

Automerge and status processing stays inside the request. Summary commands queue
in-memory work for their own scheduled job. Each process accepts one command at a
time, with concurrent webhook commands receiving 503; inference does not hold that slot.
GitHub does not automatically redeliver failed webhooks. Inspect delivery results
in the App settings and redeliver failures after recovery. Ambiguous label
responses are not retried automatically; redelivery re-reads current state.
Polling continues when webhook delivery is unavailable. App API failures can
prevent both polling and comment operations; they do not affect process health.
Installation requests share a cancellable request slot so a webhook deadline
also bounds waiting for another request or token renewal.

For rollout, replace `github.token` in runtime configuration with the `github.app`
block above. The existing deployed secret can retain its old token entry while
the new version is deployed; remove that unused entry after a successful rollout.

> [!IMPORTANT]
> App private keys and webhook secrets are read once at startup.
> Missing or empty configured sources and invalid RSA PEM files prevent startup.
> Restart the worker after changing these sources. Installation access tokens
> renew automatically. GitHub authentication failures occur during requests and
> do not make worker health fail.

### 📝 AI pull request summaries

Post a new PR comment containing exactly `/ab summarise`; surrounding whitespace
is allowed. The PR must be open and non-draft, and the unedited human comment's
author must currently have write or admin access.

Summaries are disabled by default. Configure the provider and task below, then add
the exact topic **`abgleich-ai`** to each repository you want to enable:

```yaml
ai:
  client:
    timeout: 1m
  provider: openai
  openai:
    api_key: file:/etc/secrets/abgleich/openai-api-key
summarise:
  model: gpt-6.1-sol
  interval: 30s
  timeout: 2m
```

The model is required, with no default. Omit `summarise` to disable the command's
work and its scheduled job. The independent job defaults to a `30s` interval and
`2m` timeout; configured durations must be positive. It processes at most four
queued requests serially. Inference uses at most one minute and half the remaining
job budget. Status and automerge commands remain usable while inference runs.

The App reacts 👀 on receipt and 👍 when the request is queued or a summary already
exists. Acceptance precedes model completion. The later App reply identifies the
analyzed head SHA and original request. **An existing App summary prevents another
model call or summary comment, including after restart or a head change.** To
request a fresh summary, delete the existing App summary and post `/ab summarise`
again. Failure replies do not prevent retrying.

Pending work stays in memory. After a worker restart, reissue the command. The
worker checks current App comments before generation and publication, so an
existing summary also prevents duplication after an ambiguous GitHub response.
Exactly-once inference is not guaranteed if generation finishes without a
published comment. Run one active service instance; a full queue returns HTTP 503.

> [!IMPORTANT]
> The repository topic authorizes sending selected PR metadata and changed-file
> patches to OpenAI, including private repositories and fork contributions.
> Approve the provider's data handling and billing before opting in. AI opt-in is
> independent of auto-merge settings and labels. Summaries are advisory and never
> label, approve, merge, rebase, or execute code.

Context includes at most 20 changed files, 8 KiB per patch and 16 KiB of patches in
total. PR titles, descriptions, and filenames are passed through without
truncation. Missing or oversized patches and omitted files are disclosed to the
model. Complete files and repository
trees are not retrieved; code and tests are not executed. The model returns plain
text with a 2,048-token output budget. The prompt asks for fewer than 150 words:
a brief behaviour summary and up to three changed-file pointers explaining where
to start reading. Specific context omissions get a short closing note. This word
target is prompted, rather than enforced by truncation. Per-line
citations are not generated or validated. Inspect the diff before relying on an
explanation. Source text is untrusted input; the model receives no tools or GitHub
credentials.

The command, permission, repository AI topic, and both branch revisions are checked
before source submission and publication. Revocation prevents the next observed
submission or reply. Revision changes discard the explanation and ask for a new
request. Already transmitted source cannot be recalled, and a change after the
final read can still race publication. Provider refusals, failures, incomplete or
empty responses, and timeouts produce failure feedback when still authorized;
health and auto-merge remain independent.

Keys use startup-resolved go-service sources such as `file:` or `env:`. Configured
missing or empty keys prevent startup; restart after replacement. The adapter uses
the official [OpenAI Go SDK](https://developers.openai.com/api/docs/libraries),
explicit credentials, `store: false`, and no automatic request retries.
AI HTTP settings live under `ai.client`: request timeout, TLS, and circuit
breaking use the same shared settings as `github.client`. The request timeout
defaults to `1m` when omitted or zero; negative values prevent startup. TLS
credentials and trusted roots are resolved at startup, and invalid sources
prevent startup. Circuit breaking is disabled unless configured. Inference POSTs
never retry, even when `ai.client.retry` is configured. The HTTP timeout cannot
extend the independent one-minute inference limit or the remaining job budget.
Idle connections close during shutdown after scheduled work stops.

`ai.openai.url` optionally overrides `https://api.openai.com/v1/`; use a trusted
endpoint because it receives the key and source. Existing GitHub App permissions
cover the diff reads and comment publication.

Monitor `job: summarise` outcomes and `ai generation` logs for task, model,
duration, and token usage. Generation logs use info for success and error for
failure; token usage is included on success. Provider logging is a decorator;
source, responses, and credentials are not logged. The local Cucumber harness uses fake HTTP
responses and disposable credentials. Evaluate the selected model on approved PRs
before widening opt-in; bounded diff context cannot establish whole-PR accuracy.

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
| [`automerge`](internal/automerge/reconciler.go) | Repository opt-in, strategy selection and ordering, automerge command authorization and action, and job and command registration. |
| [`automerge/internal/strategy`](internal/automerge/internal/strategy/doc.go) | Dependabot, label, and rebase policies with fresh pre-mutation checks. |
| [`summarise`](internal/summarise/doc.go) | Authorized queued summaries, bounded PR diffs, and summary job registration. |
| [`ai`](internal/ai/doc.go) | Optional credential resolution and text-only Responses API adapter. |
| [`commands`](internal/commands/doc.go) | Comment parsing, named handler dispatch, installation validation, overlap protection, background queues, and reaction feedback. |
| [`github`](internal/github/service.go) | Owner-wide repository discovery, identity checks, authenticated API access, pagination, and response validation. |
| [`scheduler`](internal/scheduler/scheduler.go) | Independent schedules, timeouts, overlap protection, outcome logs, and job lifecycle. |
| [`transport/http`](internal/transport/http/doc.go) | Webhook route registration, request deadlines, and HTTP responses. |
| [`transport/http/client`](internal/transport/http/client/doc.go) | Outbound client construction, TLS, circuit breaking, redirect protection, and connection-pool cleanup. |
| [`cmd`](internal/cmd/module.go) | Composition of package modules. |

Dependencies follow the responsibilities:

```mermaid
flowchart LR
    transport[transport/http] --> commands
    transport --> github
    commands --> github
    automerge --> commands
    automerge --> github
    automerge --> scheduler
```

`automerge.Reconciler.Reconcile` asks `github.Service` for all repositories, then uses
that service for repository settings, PR reads, identity checks, comparisons,
auto-merge enablement, and guarded rebase and merge requests. `github.Repository`
and pull request types alias SDK snapshots;
they contain no service behavior. The service uses an injected `github.Client`,
which holds the installation SDK client and a private JWT client for installation
identity checks. The GitHub module constructs it once through `github.NewClient`.
The reconciler, strategies, command handler, and dispatcher receive the same `github.Service`.
`transport/http` verifies deliveries through `github.Webhook`, then calls
`commands.Dispatcher.Handle` for command recognition, installation validation,
overlap protection, and feedback. The dispatcher selects a handler by exact
command name; `automerge.Command.Handle` owns current-comment validation,
permissions, and the label request. The
transport owns HTTP responses and delivery deadlines; go-service enforces the
configured body limit. `cmd.Module` composes the transport with the domain modules.
GitHub and AI use `transport/http/client.Factory` to construct separate outbound
clients with their own settings and connection pools. The factory resolves TLS,
configures circuit breaking, restricts redirects to the same origin, and closes
the underlying pools after consumers stop. `github.NewConnection` adds request
logging and safe-method retries; AI leaves transport retries disabled.
SDK imports stay inside `internal/github`.

Feature modules register their comment handlers through `commands.Dispatcher.Register`
before startup. The automerge module registers `automerge`; new commands live in
the feature packages that own their behavior. Duplicate names prevent startup.
The dispatcher stays independent of feature packages. Commands select an explicit
action, while reconciliation strategies select the policy applicable to a PR.

The strategy module provides the policies in this order: Dependabot, label,
rebase. The reconciler receives the ordered slice through DI and selects the first match.
A selected strategy owns the PR for that sweep, including no-op and error
outcomes. The strategy interface, implementations, and shared PR
eligibility helpers live in `internal/automerge/internal/strategy`.
Each strategy owns its applicability and fresh pre-mutation checks.
Label authorization uses paginated GraphQL history; GitHub field aliases
decode events into SDK issue-event snapshots. The GitHub package owns snapshot
completeness checks, exact label lookup, and extraction of the latest event
timestamps. Automerge applies its authorization and revocation rules to those facts.

Logging is composed around operations. `github.NewService` wraps the API service
with a logging decorator to report accepted actions after response validation.
`Scheduler.Register` wraps each handler to report its outcome and overruns
within the job's timeout.

Failed operations log at error level. Successful operations log at info level;
successful jobs that exceed their interval log at warn level. Package-owned errors
start with `package:` so wrapped failures identify their source.

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

Run a CI feature group locally with `make features-merges`,
`make features-commands`, or `make features-worker`. These select `@merges`,
`@commands`, and all remaining features, respectively. New features belong to
the worker group unless tagged for one of the other groups. Run groups one at
a time locally because the harness shares ports and configuration files.

Nonnative starts the worker and independent fake GitHub REST/GraphQL and OpenAI
Responses APIs on one local HTTP listener:

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
The harness reuses committed test credentials in `test/secrets/`, including RSA
keys created for the local fake GitHub API. Failure scenarios select
separate invalid or missing fixture paths; tests leave these files unchanged.
Reports contain logs and test results.
See [CI](.circleci/config.yml) for the build and release workflow.

### 📝 Changes

Release history is available in [GitHub releases](https://github.com/alexfalkowski/abgleich/releases).
