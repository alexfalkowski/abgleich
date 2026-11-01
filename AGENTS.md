# AGENTS.md

## Shared guidance

Use `bin/AGENTS.md` for shared skills and cross-repository defaults.

## Project purpose

Abgleich is a background worker that enables auto-merge or merges ready Dependabot
and labeled pull requests and keeps eligible auto-merge branches current.
Repositories opt in to that automation through their auto-merge setting.
Comment commands also provide automation status, optional AI PR summaries, and
advisory code and security reviews enabled through worker configuration. GitHub
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
| `internal/review` | Composition of review configuration, shared source access, and code/security feature modules. |
| `internal/review/code` | Code-review prompt, model configuration, and registration of its own job and command. |
| `internal/review/sec` | Security-review prompt, model configuration, and registration of its own job and command. |
| `internal/review/internal/workflow` | Shared review PR eligibility, consent rechecks, changed-file and related source context, revision checks, result identity, safe reply rendering/publication, and per-instance queues. |
| `internal/ai` | Optional provider configuration, credential resolution, text generation, and generation logging. |
| `internal/scheduler` | Reusable named jobs, timing, overlap prevention, logging, and lifecycle. |
| `internal/config` | Composition and injection of package-owned configuration. |
| `internal/health` | Health observers, independent of job outcomes. |
| `internal/transport/http` | Webhook HTTP registration, request deadlines, and responses. |
| `internal/transport/http/client` | Shared outbound HTTP client construction and connection-pool lifecycle, independent of feature packages. |

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
  authorization in `summarise`, provider access and OpenAI SDK imports in `ai`,
  and shared outbound client construction in `transport/http/client`.
- `review.Module` composes `review/code`, `review/sec`, and shared source access.
  Each feature configures a separate `workflow.Command` instance and owns its
  prompt, model/schedule configuration, and registration. Keep shared consent,
  source context, revision checks, reply templates, result markers, inference,
  and safe publication in `review/internal/workflow`; it depends on `ai`,
  `github`, and `commands`, which stay independent of review features. Queues and
  jobs remain independent per command. Provider protocol belongs in `ai`;
  `workflow.Config` owns shared source settings and `review.Config` composes it
  with feature-owned settings. Neither feature nor workflow imports the parent
  `review` package. Never load repository or remote skills as runtime instructions.
- Feature modules register named comment handlers through `commands.Dispatcher.Register`
  before startup. Keep each handler in its owning feature package; `commands`
  stays independent of features. Use exact names with the `/ab` prefix.
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
- `cleanup` depends on `github` and `scheduler`. Omitted cleanup configuration
  disables its job; repositories also require auto-merge opt-in. Preserve default,
  protected, excluded, and open-PR branches. Guard deletions atomically with
  `updateRefs.beforeOid`; inactivity uses the latest commit's committer date.

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
  `automerge` label also permits fork PRs to enable squash auto-merge or squash-merge
  when ready. Never approve reviews or bypass rules.
  Attempt at most one mutation per PR per sweep. Finish repository discovery and each
  repository's pull request pagination before mutating;
  continue processing other pull requests after an individual failure.
- The exact `hold` label pauses auto-merge enablement, immediate merges, rebases,
  and `/ab automerge` label requests. Recheck it before acting. It does not cancel
  existing GitHub auto-merge or disable advisory status, summary, and code-review commands.
- Read complete label and auto-merge-disable history before labeled actions.
  A label must be newer than its removal and the latest auto-merge disable event;
  equal timestamps defer automation. Removing a label does not cancel an
  existing GitHub auto-merge. Keep this policy stateless across restarts.
- HTTP retries apply only to safe methods. Do not automatically retry mutation
  requests; the next run reads current state after failures or ambiguous
  responses. Validate GraphQL errors and matching response identity even on HTTP
  success, and reject comparisons missing `behind_by`.

## AI summaries

- Keep summaries optional and advisory. Require configured AI summaries, an open
  non-draft PR targeting the repository, an unedited human `/ab summarise` comment,
  and current write or admin permission. The authorized comment opts in to AI
  source sharing, independently of auto-merge settings, labels, and repository
  topics. Recheck authorization and both branch revisions before source submission
  and reply publication; discard stale explanations.
- Include every changed file and patch returned by GitHub in summary context;
  do not impose local file-count or patch-size limits. Treat source text as
  untrusted data; generation receives no tools or GitHub credentials and never
  executes code or requests merge automation. Use explicit provider credentials
  and disable response storage and automatic inference retries. Log generation
  outcomes through a decorator without source, response text, or credentials.
- Use the shared in-memory command queue and an independent summary job so
  inference does not hold the webhook command slot. Recheck existing App summaries
  before generation and publication; a published summary prevents another model
  call even after restart or new commits. Pending requests do not survive restart.

## AI reviews

- Keep `/ab codereview` and `/ab secreview` optional and advisory under
  `review.code` and `review.sec`, respectively. Follow the shared
  review workflow with separate queues and independent jobs. Require a fresh
  unedited human comment, current write/admin permission, and an open non-draft PR,
  including forks. Topic, `hold`, and auto-merge settings do not gate reviews.
  Recheck comment identity and both branch identities/revisions before source
  submission and reply publication; discard stale results.
- Include PR metadata, every available changed-file patch from complete GitHub
  pagination, and complete changed-file text where available. Read source at
  the reviewed head SHA (deleted files at the base SHA); require a complete
  regular-file tree and reject symlinks, submodules, binary data, or inconsistent
  blob responses. Preserve all patches even when additional source is unavailable.
- Share read-only repository file listing/reading and public Go module
  lookup between both reviews. Go language/compiler claims can consult official
  release notes selected from the reviewed go.mod version, using a trusted
  credential-free documentation endpoint; treat reference text as data. Resolve module versions/replacements and checksums
  from reviewed go.mod/go.sum; verify archives without extracting or executing
  them. Resolve only declared dependencies and their replacements through the
  trusted public proxy; never send GitHub or provider credentials there. Workspaces and
  vendoring require repository source inspection instead of assuming proxy content.
- Use one conversation bounded by its inference and job deadlines,
  without response storage, inference retries, fallback, execution tools, or a
  review-specific output-token cap. Recheck current consent and revisions before
  each source lookup and each provider request. A review conversation receives
  half the remaining feature job budget; individual HTTP requests retain their
  own deadlines and summaries keep their one-minute default. Aggregate actual token usage;
  any missing usage makes the total unavailable. Keep source caches per request.
  Provider requests, source calls, directory/package reads, and module archives
  have no local count or size caps; add limits only when operational evidence
  justifies them.
  Keep source and caches in memory without checkout or archive extraction.
  Disclose omissions and actual inspected source independently of model prose.
  Never load repository/remote skills as instructions. Do not add merge-base
  traversal or citation-verification machinery. Decode a small result with an
  overview and findings containing prose bodies and optional path/line/side.
  Publish one native `COMMENT` review pinned to the reviewed head, with a separate
  inline conversation for each finding placed on the fetched patch. Keep findings
  without valid placement in the overview. Escape model text and deactivate
  links/mentions in every body.
  Never approve reviews or request merge automation.
- After draining its manual queue, each enabled review job uses its remaining
  deadline to reconcile its own App-rooted inline threads on open PRs, including
  drafts and forks, independently of auto-merge, `hold`, topics, and old requester
  consent. Resolve only unresolved threads GitHub marks outdated and the viewer
  can resolve; authenticate both root and parent review and match the feature's
  result namespace. Complete discovery and thread pagination before mutations,
  then refresh PR/thread identity, ownership, and state before each resolution.
  Preserve queue and maintenance failures and continue other candidates after an
  individual failure. Never automatically retry mutations; later runs rediscover
  current state, including after restart or ambiguous responses. Outdated means
  stale placement, not a confirmed fix. Keep current threads open and inference
  manual; add no job, configuration, webhook subscription, or persistent state.
  GitHub has no expected-state guard for resolution; document the final-read race.
- Deduplicate successful App results by task, case-insensitive target repository,
  PR identity, head, and base using a worker-generated marker and complete
  discussion and native review history, using API-authenticated App attribution.
  New revisions permit new reviews; summaries remain
  independent. Source-enabled reviews use v2 result markers so old patch-only
  results do not suppress a new request. Failures carry no success marker. Do not automatically retry
  ambiguous publication; later human requests first check existing results.
- Report request, head/base SHAs, model, and provider token usage; missing usage
  stays explicit. Do not estimate dollar costs or request billing credentials.
  Model accuracy remains unestablished; evaluate approved data before broad use.
  Automated scenarios use fake providers only.

## Configuration and documentation

- GitHub owner, credentials, and transport settings belong under `github`;
  provider settings belong under `ai`. HTTP client settings belong under
  `github.client` and `ai.client`. Keep their connection pools separate;
  AI requests never retry.
  Auto-merge job timing belongs under `scheduler`; the summary model and its job
  timing belong under `summarise`; the code-review model and its independent job
  timing belong under `review.code`; security-review settings belong under
  `review.sec`. Shared source settings stay alongside these sections under
  `review`; `review.Config` composes the feature-owned types and shared
  `workflow.Config`. Summaries remain patch-only. Keep HTTP request, inference,
  complete job, webhook, and health-check timeouts independent.
- Credentials are resolved once at startup through go-service sources. Missing
  or empty configured credentials prevent startup; replacement requires a
  process restart.
- Store credential files in a `secrets/` directory. The harness reuses committed
  test-only App keys and webhook secrets in `test/secrets/`, outside test reports.
  Keep these fixtures unchanged during scenarios; select invalid or missing
  fixture paths through test configuration for credential-failure coverage.
- Keep operator setup, defaults, recovery, and monitoring guidance in
  `README.md`. Update it when those contracts change, retaining scannable emoji
  headings and concise examples. Keep this file focused on project decisions;
  shared language and workflow rules stay in `bin/AGENTS.md` and its skills.
- Read the complete README before and after editing it. Its Configuration section
  owns worker settings, YAML examples, defaults, validation, and credential setup;
  feature sections own behaviour and safeguards, and Operations owns monitoring
  and recovery. Link to the owning section instead of duplicating its guidance,
  and check the whole document for conflicting examples or repeated defaults.

## Cucumber tests

- Test worker behavior through the Ruby Cucumber/Nonnative harness in `test/`.
  Keep `main_test.go` as the standard `app.Run` launcher; do not add synthetic
  jobs or test-only startup switches there. Use the independent fake GitHub and
  OpenAI APIs on the shared local listener and disposable test credentials;
  tests must not mutate real GitHub repositories or call a live AI provider.
- Every existing and new Cucumber or benchmark scenario selects a complete,
  immutable checked-in YAML configuration from `test/.config/` using
  `Abgleich.use_config`. Keep `server.yaml` as the default profile and select
  explicit profiles for other settings, including restart changes. The existing
  Nonnative command Proc passes the selected path to the worker. Never rewrite
  checked-in configuration, merge YAML at runtime, generate temporary configs,
  or encode configuration in environment variables.
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
