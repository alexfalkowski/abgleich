# 🛠️ Operations

[← Back to README](../README.md)

## 🩺 Health

| Endpoint | Behavior |
| --- | --- |
| `/abgleich/healthz` | Shared external connectivity check. |
| `/abgleich/readyz` | Process readiness, independent of scheduled job outcomes. |
| `/abgleich/livez` | Process liveness, independent of GitHub failures. |
| `/abgleich/metrics` | Prometheus metrics when that exporter is configured. |

Job failures, conflicts, authentication errors, and timeouts do not affect health.
Infraops probes private readiness and liveness directly, even without a Service
or Ingress.

## 📊 Telemetry

Each completed job logs `job` (`automerge`, `cleanup`, `summarise`, `codereview`, or `secreview`), `duration`,
`interval`, `overrun`, and `trigger` (`schedule` or `queue`):

| Outcome | Log level | Details |
| --- | --- | --- |
| Success | `info` | The check completed within its interval. |
| Successful overrun | `warn` | The check succeeded but exceeded its interval. |
| Failure | `error` | Includes the error, even when the check also overran. |

Timeouts cancel active work. Overruns preserve the handler's outcome and set
`overrun: true`; subsequent scheduled starts while that job is still running
are skipped without queuing.
Individual branch failures contribute to the job error while other branches
continue. Failures include repository and PR identity where available; GraphQL
errors count even with HTTP 200.

Event checks log `automerge event finished` with `repository`, `duration`, and
an error on failure. These outcomes are separate from scheduled `job finished`
records. Event failures do not affect readiness or liveness and are not
automatically requeued; later events and scheduled sweeps reconcile fresh state.
An event check can succeed without a rebase while waiting for final CI evidence.

Collect logs and alert on repeated failures, overruns, and missing completed runs.
For repeated throttling, increase the [auto-merge interval](configuration.md#-auto-merge-schedule).
There are no custom job metrics; standard go-service HTTP and runtime exporters
remain available.

Accepted actions log `requested pull request rebase`, `enabled pull request auto-merge`,
or `merged pull request` with repository, PR, and branch. `deleted branch` includes
repository, branch, and SHA. Retain these for recovery; ambiguous responses can
leave successful actions without acceptance logs.
`resolved pull request review thread` includes repository, PR, and thread identity without
comment contents.
`published pull request review` includes repository, PR, reviewed head SHA, and
the submitted `event` (`COMMENT`, `APPROVE`, or `REQUEST_CHANGES`).

Monitor `ai generation` for task (including `codereview` and `secreview`), model, duration, and successful token usage.
Success logs at info and failure at error. Provider failures add `http_status`,
`provider_code`, `provider_type`, and `provider_request_id` when available;
codes and types must be recognized, and request IDs must match the supported format.
Provider diagnostics exclude raw messages/responses, request bodies, URLs, source,
and credentials.

## 📦 Deployment

Run the standard service container as one long-lived active instance; scheduling
limits are per process, while head guards protect against concurrent updates.
Use liveness for restarts to avoid outage-driven restart loops. Shutdown cancels
all scheduled jobs, stops the automerge coordinator, then closes outbound
connection pools.

## 🧯 Recovery

### 🔄 Auto-merge recovery

No checkpoint or database is needed. After crashes, timeouts, or ambiguous
responses, the next sweep reconciles fresh GitHub state. An accepted update can
finish after the caller stops waiting; inspect remote state before acting manually.
Existing GitHub auto-merge on fork PRs is left unchanged; disable it in GitHub
separately if needed. Use a branch in the target repository for merge automation.
If event rebasing is waiting, inspect the configured completion signal on the
current head. Failed workflows that never emit their final signal and GitHub
permission failures when reading CI evidence leave repair to the scheduled sweep.

### 💬 Webhook recovery

GitHub does not automatically redeliver failures. Inspect App delivery results
and redeliver after recovery; current state is re-read. Scheduled sweeps recover
automation state after missed, rejected, or lost repository notifications, but
they do not replay comment commands. Ambiguous label responses are not
automatically retried.

### 📝 Summary recovery

Pending requests are lost on restart; post `/ab summarise` again. For a fresh
summary, delete the existing App summary first. Failure replies permit retrying.
App comments are checked before generation and publication, including after
ambiguous responses. Inference can repeat if generation finishes without publication.

### 🔬 Code-review recovery

Post a new `/ab codereview` after a restart loses pending work or an attempt fails.
Use `/ab review` to request all configured reviews again; each feature reuses
its successful results and queues only missing work.
Failures, including malformed model result JSON or a missing or null `complete`
field, carry no successful-result marker and use ordinary failure comments. These
show an emoji summary, a safe reason, and recovery advice. Collapsed **Failure details**
identify the processing stage, original request, head/base SHAs, configured model,
unavailable token usage, and available provider diagnostics described under [Telemetry](#-telemetry).
Unknown provider metadata and raw provider messages are withheld from comments.
Before generation and publication,
the worker reads complete discussion and native review history for matching App
results; human or other-App lookalikes do not count. Deduplication includes task, case-insensitive
target repository, PR identity, head, and base. A matching result reuses its
original usage; new head or base revisions permit another manual review.
Submitted approvals, requests for changes, comments with success markers, and
dismissed results all prevent duplicate inference. Incomplete comments carry no
success marker; retry with a new command after addressing missing context or
increasing the relevant [review budget](configuration.md#-code-review-settings).
Existing ordinary App results remain valid and can still be deleted before a
fresh request. Submitted native reviews are not pending reviews and cannot be
reset through these commands; use a new revision for a fresh review. Inference may repeat when
an earlier attempt generated without publishing. Ambiguous publication is never
automatically retried; a later human request first discovers any published result.
Inspect job and generation errors for incomplete API responses, provider failures,
context-size failures, and deadlines before retrying.

Outdated-thread maintenance rediscovers GitHub state on the existing review
schedule, including after restart or a lost resolution response. No human command
is needed for this cleanup. Queue work, job deadlines, or GitHub failures can delay
it; failed threads do not block other candidates. Closed PRs, fork PRs, and disabled
review features receive no maintenance. Existing overview comments and overview-only
findings remain historical output. See [review behavior](reviews.md#-ai-code-reviews).

### 🛡️ Security-review recovery

Post a new `/ab secreview` after pending work is lost on restart or an attempt
fails. The [code-review recovery rules](#-code-review-recovery) also apply here:
complete discussion lookup, authenticated App results, revision-specific reuse,
and no automatic retry of ambiguous publication. Security results use their own
`secreview` marker, so other tasks do not suppress a security review.

### 🧹 Cleanup recovery

Recreate an unintended deletion from the logged SHA while GitHub retains the
commit, or use **Restore branch** on a merged PR. Inspect GitHub first: a lost
response can mean deletion succeeded without an acceptance log.
