# ⚙️ Configuration

[← Back to README](../README.md)

Start with the base configuration and merge optional blocks into the same file.
Feature sections explain safeguards; [Operations](operations.md#-operations) covers monitoring
and recovery.

## 📄 Base configuration

Save as `config.yaml`, replacing the owner, App IDs, and private-key source.
`github` and `health` are required. `health.duration` sets sampling frequency and
`health.timeout` bounds connectivity checks independently of jobs; both must be
positive and have no defaults. Other blocks configure shared go-service facilities.

```yaml
environment: production
github:
  owner: alexfalkowski
  app:
    id: 5427450
    installation_id: 174152088
    private_key: file:/etc/secrets/abgleich/github-app-private-key.pem
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

This runs merge automation; optional features and [defaults](#-defaults-and-validation)
follow below.

## 🤖 GitHub App setup

Register one App with **Contents**, **Pull requests**, and **Issues** permissions
set to **read and write**; metadata read is automatic. These cover commands,
scheduled automation, cleanup, summaries, and both review commands, including reactions,
which require Issues write. For event-driven rebasing, also grant **Commit statuses: read** and
**Checks: read** to inspect completion evidence.

Install on all owned repositories to include future repositories automatically,
or select them explicitly. Enable **Allow auto-merge** for merge automation and
cleanup, and **Allow squash merging** for Dependabot and labeled automation.
Give the App no ruleset bypass.

`github.owner` must match the installation account. App ID, owner, and suspension
are validated before repository access on each sweep, event check, and command.
REST and GraphQL share installation credentials; `ghinstallation` signs requests and
renews tokens. Redirects to other API origins are rejected; personal tokens are
not used.

## 💬 Comment webhooks

Add to `github` to enable commands and event-triggered checks; omitting
`github.webhook` disables the endpoint while scheduled sweeps continue.

```yaml
github:
  webhook:
    secret: file:/etc/secrets/abgleich/github-webhook-secret
    timeout: 8s
```

Configure a matching random secret and an HTTPS URL ending in `/webhooks/github`.
Select the App event subscriptions needed below:

| Subscription | Processed activity |
| --- | --- |
| **Issue comment** | New PR comments for `/ab` commands. |
| **Pull request** | Opened, reopened, synchronized, ready for review, converted to draft, edited, labeled, unlabeled, auto-merge enabled or disabled, and closed. |
| **Push** | Branch pushes, including target-branch updates; tag pushes are ignored. |
| **Status** | Terminal commit statuses (`success`, `failure`, or `error`); required to react to the existing CircleCI final status. |
| **Check run** | Completed check runs; use when the final CI signal or remaining work is reported as check runs. |

Other events and actions are acknowledged without work. Existing comment-only
subscriptions keep working; add subscriptions to enable event checks. Provision
DNS and ingress before activating the proposed URL,
`https://abgleich.lean-thoughts.com/webhooks/github`.

Keep request, webhook, job, and health timeouts independent. The HTTP timeout must
accommodate webhook work: the example allows `8s` within the server's `9s` and
GitHub's ten-second deadline. `transport.http.max_receive_size` also applies.
See [webhook delivery](commands.md#-webhook-delivery) for responses and concurrency.

## 🤖 Dependabot settings

To require explicit opt-in for Dependabot PRs in selected repositories, merge
this block into the worker configuration:

```yaml
automerge:
  dependabot:
    exclude_repositories:
      - alexfalkowski/docker
```

Entries match complete `owner/repository` names case-insensitively, without
wildcards or partial-name matching. Restart the worker after changing exclusions.
See [merge automation](automation.md) for explicit opt-in and existing auto-merge behavior.

## 🧪 Event-driven rebase settings

To allow event checks to rebase after CI has finished, configure its final
completion signal. For the existing CircleCI workflows:

```yaml
automerge:
  completion_check: "ci/circleci: wait-all"
```

The value is an exact, case-sensitive commit-status context or check-run name.
Choose a signal emitted only after all CI work, including branch synchronization,
has finished. The configured signal must be terminal on the current head, and all
reported check runs and commit statuses must also be terminal. Completion can be
successful or failed.

Omitting `automerge` or leaving `completion_check` empty leaves rebasing to scheduled
recovery; events still apply the configured merge policies.
Configure the [App permissions](#-github-app-setup) and
[webhook subscriptions](#-comment-webhooks) to read evidence and receive later
results. No CircleCI API credentials are used.

The current `wait-all` job runs after successful upstream jobs. If an earlier job
fails, its final status may never appear; the next scheduled sweep remains the
recovery path. A PR status assessment describes scheduled policy, so it can still
report a possible rebase while event rebasing waits for CI evidence.

## 🧹 Cleanup settings

Add this block to enable branch cleanup; omit it and restart to disable its job.
Read the [branch cleanup safeguards](automation.md#-branch-cleanup) before enabling deletion.

```yaml
cleanup:
  inactive_after: 720h
  interval: 24h
  timeout: 5m
  exclude:
    - develop
```

`exclude` contains exact, case-sensitive branch names to preserve in addition to
protected, default, and open-PR branches. `inactive_after` is the latest-commit age
required for inactive cleanup.

## 📝 Summary settings

Add `ai` and `summarise` and configure [webhooks](#-comment-webhooks) for `/ab summarise`.
Set the key and model explicitly; read the [source-sharing notice](summaries.md#-ai-pull-request-summaries).

```yaml
ai:
  provider: openai
  openai:
    api_key: file:/etc/secrets/abgleich/openai-api-key
summarise:
  model: gpt-6.1-sol
  interval: 30s
  timeout: 2m
```

Omitting `summarise` disables summary work and its job even if `ai` remains. Only override `ai.openai.url` with a trusted endpoint:
it receives the key and source. Existing App permissions cover summaries.

## 🔬 Code-review settings

Review commands and shared source options live under one `review` mapping.
Merge the `code`, `sec`, and source settings below into that mapping as needed.
[`/ab review`](reviews.md#-request-all-configured-reviews) requests every enabled
review without an additional configuration block.

Configure `ai` using the [provider settings](#-summary-settings), configure
[webhooks](#-comment-webhooks), and add this independent block for `/ab codereview`:

```yaml
review:
  code:
    model: gpt-6-astra
    reasoning_effort: xhigh
    interval: 30s
    timeout: 10m
```

`model` is required and operator-selected. Optional `reasoning_effort` selects a
provider-supported effort; omission preserves the model default. Higher effort and a longer job timeout
can increase latency and token usage and must be evaluated with the selected model.
Omitting `review.code` disables
the command and its job; `ai` alone enables no AI feature.
Configured reviews require a usable AI provider or startup fails. Omit `ai`,
`summarise`, `review.code`, and `review.sec` to run without AI. Existing App permissions cover source
reads; no new credential is required. Read the [source-sharing notice](reviews.md#-ai-code-reviews)
and [evaluation requirements](development.md#-code-review-evaluation) before enabling.

## 🛡️ Security-review settings

Configure `ai` using the [provider settings](#-summary-settings), configure
[webhooks](#-comment-webhooks), and add this independent block for `/ab secreview`:

```yaml
review:
  sec:
    model: gpt-6-astra
    reasoning_effort: xhigh
    interval: 30s
    timeout: 10m
```

`model` is required and operator-selected. `reasoning_effort` follows the
[code-review setting](#-code-review-settings). Omit `review.sec` to disable its command
and job. Configured security reviews require a usable AI provider or startup fails;
no new provider or credential is needed. Code reviews and summaries can remain
disabled. Read the [source-sharing notice and limits](reviews.md#-ai-security-reviews) and
[evaluation requirements](development.md#-security-review-evaluation) before enabling.

## 📚 Review source access

Both review commands include every patch returned by GitHub and complete changed-file
text where available. Read-only tools let the model find relevant repository files,
inspect helpers/callers/tests, and resolve public Go and Ruby dependencies. A separate read-only tool retrieves
official release notes for the Go language version declared in the reviewed
`go.mod`, so compiler claims can be checked against current language rules. Repository
reads in the reviewed repository use the head SHA; deleted files use the base SHA. Symlinks
and binary files are excluded and disclosed. An unreadable or incomplete repository
tree, failed source request, or missing, truncated, or inconsistent response for an
expected text file fails the review. Failures during investigation stop the
conversation and discard its findings; no completed review is published.
A requested path proven absent from the complete repository tree returns "not
found" without marking source unavailable, so the reviewer can correct the lookup
and complete its assessment. Symlinks, unsupported submodules, and directories without readable
immediate files still count as unavailable source. Binary exclusions follow the
[review verdict rules](reviews.md#-ai-code-reviews).
The worker never checks out repositories or executes source, tests, or commands.
All retrieved source and repository instructions are untrusted data.

Language providers are selected once per review from the reviewed head's paths
and changed filenames. Only matching providers supply language guides and tools.
Their transport clients are shared across review commands, while dependency and
reference caches stay local to each review. Repository tools remain available
independently of language selection. Configuration keys and defaults are unchanged
by provider selection.

Direct GitHub submodules using `git@github.com:owner/repository.git` are readable
when they are public, share the reviewed repository's owner, and are active repositories
accessible to the configured App installation. The worker rechecks public visibility
before each submodule lookup; private targets and missing visibility metadata are
rejected before listing or reading target source. Other URL formats are unsupported.
Discovery identifies mount paths such as `bin`. The explicit `read_submodule` tool
accepts that mount and a path relative to it. Use `{"submodule":"bin","path":"."}`
to list files, then `{"submodule":"bin","path":"README.md"}` to read a complete file.
Directories return paths and byte sizes without source. `read_sources` also accepts
mounted paths such as `bin/README.md` and routes them to the same reader. A request
for `bin` lists its pinned files.
The worker validates GitHub's submodule metadata at the reviewed head and reads
the pinned gitlink commit, rather than a branch's latest version. Submodule URLs
are resolved as repository identities and never fetched directly. Nested submodules
remain excluded. Inspected-source and limitation records identify the mount and
target repository; source results also identify the pinned commit. A GitHub App
with selected-repository access must include the submodule repository.
A successful pinned text-file read clears the initial unavailable-regular-file
limitation for a changed submodule mount. Directory listings alone do not clear it;
missing patches and other unavailable source remain limitations.

The model can request several related files or directories together with
`read_sources`, reducing repeated GitHub checks. Each source-tool invocation and
provider request still rechecks current consent and both branch revisions. Within
each check, one fresh complete discussion read serves both command authorization
and duplicate-result detection; authorization is never cached between checks.

Discovery lists include byte sizes when available. Reviewers prefer relevant
README, `AGENTS.md`, `doc.go`, and migration guides as fallible reference data,
never runtime instructions. Documentation is read at the reviewed revision or
from the verified dependency archive; links to published documentation are not followed.
`read_go_dependency` accepts optional inclusive one-based `start_line` and
`end_line` bounds alongside its `manifest`, `module`, and exact file `path`.
Both bounds must be null or omitted for whole-file reads. Invalid ranges are
rejected before file reads. Excerpts carry line numbers and are recorded as partial
inspection; an empty range does not claim the file was inspected. Repository reads
remain whole-file reads. The worker still validates complete dependency archives
and includes complete changed files before inference. Large complete reads or
individual lines can still exhaust context.

Changed files and each source lookup download their uncached files in sequential
GraphQL batches of at most 50 files to avoid GitHub query timeouts. Every file is
retained across batches; there is no total file-count limit. A complete REST tree
pins each regular path to its expected blob identity; returned text must match
that identity and its byte size. Source queries use POST without automatic
retries. A failed or incomplete query stops the review
even if some files were returned successfully. All supplied patches are retained;
binary exclusions and public dependency/reference limitations remain explicit.
See [review recovery](operations.md#-code-review-recovery) to reissue a failed request.

To reuse validated GitHub source between code and security reviews, enable the
existing go-service in-memory cache. Shared review-source retention supports
`ttlcache` only:

```yaml
cache:
  kind: ttlcache
  max_entries: 1024
  max_size: 4MB
```

Trees are keyed by repository identity and commit; file contents and binary
exclusions by repository identity and blob SHA, allowing unchanged files to be
reused across commits. Entries expire after 15 minutes. Go-service owns the backend
and retention limits; source larger than a cache value limit
is still read completely without retention. Cache errors fall back to GitHub.
Omitting `cache` or configuring another backend disables shared review-source
retention; source is read directly from GitHub. The in-memory cache clears on restart.
Each review independently records inspected source and omissions. Authorization,
PR state, patches, and review history remain fresh reads, and simultaneous cache
misses can fetch the same source independently.

Source options alone enable neither review command. Dependency lookup follows
the modules, versions, and replacements in the reviewed `go.mod`; archives must
match its `go.sum`. Version-specific replacements take precedence over wildcard replacements.
This lookup inspects the declared dependency, without resolving a complete build graph.
Dependency directory lookups return sorted module-relative file paths and byte sizes without
source; `.` lists the module root. The model then requests exact file paths to
read their complete text, avoiding whole-package reads of generated SDKs.
A path confirmed absent from its verified module archive returns "not found" to
the reviewer without marking source unavailable. The reviewer can correct the
lookup and complete its assessment. Unreadable external source, unverified
archives, and unavailable language references are disclosed as external source
notes; unverified contents are never submitted to the model. These gaps alone
do not prevent approval when the reviewer has high confidence in the repository
changes. Material unresolved questions can still make the assessment incomplete.
Local replacements, workspaces, vendored source, and private dependencies
require repository source inspection. Go lookup
uses `https://proxy.golang.org` without GitHub or AI credentials. Override
`review.module_proxy` only with a trusted Go module proxy; it receives declared
module names. Go references use `https://go.dev/doc`; `review.go_docs` may select
a trusted documentation mirror. These requests also carry no credentials.
These Go tools accept neither arbitrary URLs nor model-selected versions.
Public dependency archives and language references remain cached per request.

`read_ruby_dependency` reads public gems from the exact version and platform in a
reviewed `Gemfile.lock`. It accepts `manifest`, `gem`, `platform`
(`ruby` for the generic artifact), and a file or directory `path`; `.` lists files.
Optional `start_line` and `end_line` work like Go dependency excerpts. For example:

```json
{"manifest":"test/Gemfile.lock","gem":"nonnative","platform":"ruby","path":"lib/nonnative.rb"}
```

Only `GEM` entries sourced from `https://rubygems.org/` are supported. Git/path
dependencies, private registries, ambiguous entries, and content-addressed gems
are disclosed as unavailable. Gem versions and allowed platforms come from the
lockfile, never a request to resolve current versions. The downloaded `.gem` must
match RubyGems' version/platform metadata and SHA-256; when the lockfile records
SHA-256 it must match that too. Replies distinguish registry verification from a
reviewed lockfile checksum. Archives are read in memory; Ruby code, Gemfiles,
gemspecs, and native extensions are never executed. Invalid paths and links are
rejected. Repeated source reads reuse the verified compressed archive within that
review; only the requested file or directory listing is retained from each scan.
Source is shared only after the complete archive passes validation.

Ruby source access shares one **32 MiB byte budget per review**, independently for
code and security reviews. It counts registry metadata and downloads, compressed
archive input, expanded data, and requested-file reads during every lookup,
including cached rescans and discarded trailing data. This bounds decompression
and retained source without separate file-count or excerpt limits. Known oversized downloads
are rejected from their HTTP headers before reading the body; unknown sizes are
checked while reading. A requested file whose declared size cannot fit the
remaining budget is rejected before buffering, including for an excerpt.
Cancellation is checked during network and archive reads. Exceeding the budget
is disclosed as unavailable external source, so a large or repeatedly inspected
gem may be omitted while the review continues under the normal verdict rules.

The default registry endpoint is `https://rubygems.org`; `review.gem_registry`
may select an operator-trusted mirror. Requests carry no GitHub or AI credentials.
Ruby source GETs use the shared HTTP client's safe-method retry policy, with at
most one retry and its default jittered backoff. When `Retry-After` seconds or an
HTTP date requires a longer wait than that backoff, the client returns the rate
limit without retrying. The review discloses unavailable source and continues
under the normal verdict rules. There is no separate Ruby pacing or cooldown
state, so one rate-limited lookup does not hold another review. Provider
inference requests still never retry.

GitHub source access uses the App's installation token and is limited to the reviewed
repository and the permitted submodules described above. Reviews do not browse arbitrary
repositories or websites. Implementations outside the supported source tools are outside the
review scope; their absence alone does not make a review incomplete.

The conversation runs within the existing inference/job deadlines, without local
provider-request or source-call limits. Source reads have no local byte, file-count,
or archive-size caps; `read_sources` directory reads include all immediate matching files.
Working source and request-local caches stay in memory, so large reviews can
increase memory usage and exceed provider context limits or worker deadlines.
Unavailable source is reported.

Comments list inspected source and failed reads independently of model prose.
Token details sum actual usage across all provider requests; missing usage on any
request makes the total unavailable. Source-enabled results use a new marker
version: an old patch-only result permits one new review of the same revisions.
Summaries remain focused on patches and do not use source tools.

## 🔌 HTTP transport

`github.client` and `ai.client` have separate connection pools, request timeouts,
TLS, and circuit-breaking settings. Circuit breaking and GitHub retries are
disabled unless configured. To enable safe-method retries, merge into `github`:

```yaml
github:
  client:
    retry:
      attempts: 2
      backoff: 100ms
```

Mutations are never retried by the transport; later runs read fresh state.
AI requests never retry, even with `ai.client.retry` configured. Their HTTP timeout
cannot extend the independent inference or job budget.

For deeper reviews, allow longer individual model requests alongside the feature
job timeout. The default one-minute HTTP timeout can interrupt high-effort reasoning:

```yaml
ai:
  client:
    timeout: 3m
```

This setting is shared by AI features; summaries retain their separate inference limit.

## 🗓️ Auto-merge schedule

`scheduler.interval` controls owner-wide sweeps. `scheduler.timeout` bounds a
complete sweep, including waiting for the automerge coordinator, and each event
check once it begins. Event checks start as the coordinator becomes available.
Cleanup, summaries, code reviews, and security reviews use their own blocks.

```yaml
scheduler:
  interval: 15m
  timeout: 5m
```

## ⏱️ Defaults and validation

| Setting | Default | Purpose |
| --- | --- | --- |
| `scheduler.interval` | `15m` | Time between owner-wide auto-merge sweeps. |
| `scheduler.timeout` | `5m` | Deadline for a complete auto-merge sweep or an executing event check. |
| `automerge.completion_check` | `""` | Final CI signal required for event-driven rebasing; an empty value disables event rebasing. |
| `automerge.dependabot.exclude_repositories` | `[]` | Full repository names requiring explicit Dependabot opt-in; omitted or empty settings preserve automatic enrollment everywhere. |
| `cleanup.inactive_after` | `720h` (30 days) | Latest-commit age required for inactive cleanup. |
| `cleanup.interval` | `24h` | Time between cleanup runs. |
| `cleanup.timeout` | `5m` | Deadline for a complete cleanup run. |
| `cleanup.exclude` | `[]` | Exact, case-sensitive branch names to preserve. |
| `summarise.model` | None | Required model when `summarise` is configured. |
| `summarise.interval` | `30s` | Time between summary queue processing runs. |
| `summarise.timeout` | `2m` | Deadline for a complete summary job run. |
| `review.code.model` | None | Required model when code reviews are configured. |
| `review.code.reasoning_effort` | Provider default | Optional model-supported reasoning effort. |
| `review.code.interval` | `30s` | Time between scheduled thread maintenance and recovery runs; manual requests wake the job immediately. |
| `review.code.timeout` | `2m` | Deadline for a complete code-review job run. |
| `review.sec.model` | None | Required model when security reviews are configured. |
| `review.sec.reasoning_effort` | Provider default | Optional model-supported reasoning effort. |
| `review.sec.interval` | `30s` | Time between scheduled thread maintenance and recovery runs; manual requests wake the job immediately. |
| `review.sec.timeout` | `2m` | Deadline for a complete security-review job run. |
| `review.module_proxy` | `https://proxy.golang.org` | Trusted credential-free Go module source endpoint. |
| `review.go_docs` | `https://go.dev/doc` | Trusted versioned Go release-note source or mirror. |
| `cache.max_entries` | `1024` | Maximum entries in the configured in-memory cache. |
| `cache.max_size` | `4MB` | Maximum encoded cache value size; larger source remains readable without caching. |
| `github.client.timeout` | `1m` | Deadline for an individual GitHub HTTP request. |
| `github.url` | `https://api.github.com/` | REST API base URL, shared with GraphQL. |
| `github.webhook.timeout` | `5s` | Complete synchronous delivery budget; must be positive and below `10s`. |
| `ai.client.timeout` | `1m` | Deadline for an AI HTTP request, bounded by the inference and job budgets. |
| `ai.openai.url` | `https://api.openai.com/v1/` | Trusted provider API base URL. |
| `transport.http.max_receive_size` | `4MB` | Incoming request body limit for all HTTP routes. |

Omitted or zero auto-merge, cleanup, summary, code-review, and security-review job durations, inactivity
thresholds, webhook budgets, and GitHub and AI client timeouts use the defaults
above. Negative schedule durations, inactivity thresholds, and HTTP client
timeouts are invalid. Review reasoning effort must be omitted or one of `none`,
`minimal`, `low`, `medium`, `high`, `xhigh`, or `max`; the chosen model must support it. Excluded branch and repository names must be nonempty.

## 🔐 Credentials

App private keys, webhook secrets, and AI keys use go-service sources such as
`file:/etc/secrets/abgleich/github-app-private-key.pem` or `env:OPENAI_API_KEY`.
Credentials and TLS sources, including trusted roots, are resolved once at startup.
Keep credential files in a `secrets/` directory.

> [!IMPORTANT]
> Missing or empty configured credential sources and invalid RSA PEM or TLS
> sources prevent startup. Restart the worker after replacing these sources.
> Installation access tokens renew automatically. Authentication failures during
> requests do not make worker health fail.
