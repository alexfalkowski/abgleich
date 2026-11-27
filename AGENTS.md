# AGENTS.md

## Shared guidance

Use `bin/AGENTS.md` for shared skills and cross-repository defaults.

In Go code, use `strings.Empty` from `github.com/alexfalkowski/go-service/v3/strings`
for explicit empty string values and `strings.IsEmpty` for emptiness checks.
Use named struct types in production Go code. Anonymous structs are allowed in
tests and GitHub GraphQL payloads.

## Accepted design

Read the [accepted-design index](docs/agents/accepted-design.md) and the relevant
area files before proposing a change or recording a finding or audit entry.
Apply each decision within its documented scope and record new owner-accepted
decisions in the relevant area file.

## Project purpose

Abgleich is a background worker that enables auto-merge or merges ready Dependabot
and labeled pull requests and keeps eligible auto-merge branches current.
Repositories opt in to that automation through their auto-merge setting.
Comment commands also provide automation status, optional AI PR summaries, and
AI code, security, and test reviews enabled through worker configuration. GitHub
performs all repository mutations; the worker does not clone repositories or execute Git. Each auto-merge
run reconciles fresh remote state; pending summary and review requests are held in memory.

## Package ownership

| Package | Owns |
| --- | --- |
| `internal/cmd` | CLI command registration and composition of package modules. |
| `internal/commands` | PR comment parsing, named handler dispatch, installation validation, shared human-comment authorization, overlap prevention, background queues, and reaction feedback. |
| `internal/github` | Service, API client, SDK aliases, owner-wide discovery, repository identity, remote access, and response validation. |
| `internal/automerge` | Repository opt-in, PR eligibility, merge and rebase decisions, automerge comment authorization and action, and registration of its own job and command. |
| `internal/cleanup` | Optional merged and inactive branch cleanup policy, pre-deletion rechecks, exclusions, and registration of its own job. |
| `internal/status` | Status comment authorization, snapshot rechecks, reply rendering and publication, and registration of its own command. |
| `internal/summarise` | Summary authorization, all changed-file patches returned by GitHub, queued processing, reply publication, and registration of its own job and command. |
| `internal/review` | Composition of review configuration, shared source access, feature modules, and aggregate review command registration. |
| `internal/review/code` | Code-review prompt, model configuration, and registration of its own job and command. |
| `internal/review/sec` | Security-review prompt, model configuration, and registration of its own job and command. |
| `internal/review/test` | Test-quality review prompt, model configuration, and registration of its own job and command. |
| `internal/review/internal/workflow` | Review admission, PR eligibility, consent and revision rechecks, per-feature queues, inference, feedback, and serialized publication. |
| `internal/review/internal/history` | Authenticated App review history, result markers, follow-up records, prior blocker evidence, and cross-feature approval checks. |
| `internal/review/internal/assessment` | Model result validation, finding reconciliation and placement, verdict selection, and safe native review rendering. |
| `internal/review/internal/threads` | Outdated App-owned thread discovery, fresh eligibility and ownership checks, and resolution by feature namespace. |
| `internal/review/internal/sources` | Pinned repository and submodule reads, per-review evidence and source tools, and provider composition and selection through a DI value group. |
| `internal/review/internal/sources/provider` | Shared repository-access and tool contracts, DI contributions, source endpoint configuration, and excerpt rendering. |
| `internal/review/internal/sources/golang` | Go guide, source tools, manifest/checksum resolution, module archives, language references, HTTP policy, and per-review caches. |
| `internal/review/internal/sources/ruby` | Ruby guide, source tools, lockfile resolution, gem verification, bounded archive reads, HTTP client policy, and per-review caches. |
| `internal/review/internal/sources/cucumber` | Selection of worker-owned Cucumber review guidance. |
| `internal/ai` | Optional provider configuration, credential resolution, text generation, and generation logging. |
| `internal/scheduler` | Reusable named jobs, timing, overlap prevention, logging, and lifecycle. |
| `internal/config` | Composition and injection of package-owned configuration. |
| `internal/health` | Health observers, independent of job outcomes. |
| `internal/transport/http` | Webhook HTTP registration, request deadlines, and responses. |

- Constructors belong in the packages owning their types; `cmd` composes their
  modules. Keep dependencies directed from `automerge` to `github`, `scheduler`,
  and `commands`. These dependencies do not import `automerge`.
  `status` depends on `automerge` for read-only policy assessment and on `github`
  and `commands` for remote access and command registration. `automerge` stays
  independent of `status`; strategy implementations remain internal to it.
  Strategies return decision data; `automerge` owns the assessment contract and
  explanations, while `status` owns GitHub Markdown rendering.
- `summarise` depends on `ai`, `github`, `commands`, and `scheduler`; these
  dependencies stay independent of `summarise`. Keep source selection and
  authorization in `summarise` and provider access and OpenAI SDK imports in `ai`.
  GitHub, AI, and review source packages build their outbound HTTP clients from
  package-owned settings using go-service and register pool cleanup with the lifecycle.
- `review.Module` composes `review/code`, `review/sec`, `review/test`, and shared source access.
  Each feature configures a separate `workflow.Command` instance and owns its
  prompt, model/schedule configuration, and registration. Workflow coordinates
  consent, revisions, inference, and publication using `sources`, `history`,
  `assessment`, and `threads`. These packages stay independent of workflow.
  History owns the persisted review format shared with assessment and the
  feature namespace checked by thread maintenance. Keep queues and jobs
  independent per command and share one final publication gate. Provider
  protocol belongs in `ai`. `sources/provider.Config` owns shared source settings
  and `review.Config` composes it with feature-owned settings. The feature and
  internal review packages stay independent of the parent `review` package.
  Never load repository or remote skills as runtime instructions.
- `sources.Module` composes provider modules, each contributing its strategy to
  the source DI value group. Keep language/framework guides,
  tools, manifest parsing, verification, HTTP policy, and request-local dependency
  caches in the owning `review/internal/sources` provider packages. Providers do
  not import workflow or each other. Sources supplies pinned repository reads
  and inspection records, and applies the workflow's consent and revision guard
  before every source tool call. Keep registry selection independent of provider
  implementations.
- Feature modules register named comment handlers before startup. Use
  `commands.Dispatcher.RegisterConcurrent` for review commands and
  `commands.Dispatcher.RegisterSerial` for other commands. Keep each handler in its
  owning feature package and keep `commands` independent of features.
  Use exact names with the `/ab` prefix.
  Use `commands.Authorizer` for fresh human comment identity, exact command,
  edit status, and write/admin permission checks. Keep PR eligibility and
  feature-specific consent history in the feature, and repeat authorization at
  its existing source-submission and publication/action boundaries.
- Keep owner-wide discovery and case-insensitive repository identity checks in
  `github.Service`. The reconciler receives the service and uses SDK repository
  snapshots, not GitHub configuration or a separately cached repository name.
- Keep accepted-action logging in the service's logging decorator, separate from
  remote access and reconciliation decisions.
- Keep the strategy interface, implementations, and shared PR eligibility helpers
  in `internal/automerge/internal/strategy`. `automerge.Module` constructs the
  strategies and provides their precedence order for injection into the reconciler.
- `cleanup` depends on `github` and `scheduler`.

## Configuration and documentation

- GitHub owner, credentials, and transport settings belong under `github`;
  provider settings belong under `ai`. HTTP client settings belong under
  `github.client`, `ai.providers.<name>.client`, `review.module_proxy`,
  `review.go_docs`, and `review.gem_registry`. Set `endpoint` in each block and keep
  the connection pools separate. AI requests never retry.
  Auto-merge job timing belongs under `scheduler`; the summary model and its job
  timing belong under `summarise`; the code-review model and its independent job
  timing belong under `review.code`; security-review settings belong under
  `review.sec`; test-review settings belong under `review.test`. Shared source
  settings stay alongside these sections under `review`; `review.Config` composes the feature-owned types and shared
  `sources/provider.Config`. Summaries remain patch-only. Keep HTTP request, inference,
  complete job, webhook, and health-check timeouts independent.
- Credentials are resolved once at startup through go-service sources. Missing
  or empty configured credentials prevent startup; replacement requires a
  process restart.
- Store credential files in a `secrets/` directory. The harness reuses committed
  test-only App keys and webhook secrets in `test/secrets/`, outside test reports.
  Keep these fixtures unchanged during scenarios; select invalid or missing
  fixture paths through test configuration for credential-failure coverage.
- Keep the overview, quickstart, and documentation links in `README.md`, with
  detailed guides in `docs/`. Update the owning guide when contracts change,
  retaining scannable emoji headings and concise examples. Keep this file focused
  on project decisions; shared language and workflow rules stay in `bin/AGENTS.md`
  and its skills.
- Read the complete README before and after editing it. `docs/configuration.md`
  owns worker settings, YAML examples, defaults, validation, and credential setup;
  feature guides own behaviour and safeguards, and `docs/operations.md` owns
  monitoring and recovery. Link to the owning guide instead of duplicating its
  guidance, and check moved links, conflicting examples, and repeated defaults.

## Cucumber tests

- Write features in natural domain language with one Given/When/Then
  progression, from the maintainer's or operator's perspective. Keep fixture
  modes, source markers, and payload formatting in step definitions. Retain
  protocol details when they express a public delivery or safety contract.
  Avoid interleaving setup, actions, and assertions as a procedural
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
