# AI reviews

Part of the [accepted-design index](../accepted-design.md).

- Keep `/ab codereview`, `/ab secreview`, and `/ab testreview` optional under
  `review.code`, `review.sec`, and `review.test`, respectively. Follow the shared
  review workflow with separate queues and independent jobs. Require a fresh
  unedited human comment, current write/admin permission, and an open non-draft PR
  with both branches in the repository. Ignore forks. Topic, `hold`, and auto-merge settings do not gate reviews.
  Recheck comment identity and both branch identities/revisions before source
  submission and reply publication; discard stale results.
- `/ab review` requests every enabled review constructed by `workflow.Factory`.
  Keep the original aggregate comment as consent at every recheck, and preserve
  per-feature queues, jobs, results, and deduplication with individual commands.
  Attempt every configured feature and retain accepted work when another fails;
  acceptance requires at least one enabled feature and acceptance by all of them.
- Include PR metadata, every available changed-file patch from complete GitHub
  pagination, and complete changed-file text where available. Read source at
  the reviewed head SHA (deleted files at the base SHA); require a complete
  regular-file tree and reject symlinks, binary data, or inconsistent
  blob responses. Preserve all patches even when additional source is unavailable.
- Discover accepted design at the pinned target base. Include
  `docs/agents/accepted-design.md` and Markdown under
  `docs/agents/accepted-design/` in untrusted review input when present. List
  agent documentation at the PR head and present head design and diff text
  separately as branch and proposal evidence. Verify design files, including
  changed design files, independently within a quarter of the remaining job
  time. Keep budgeted reads outside the shared GitHub circuit breaker. Record
  unavailable design trees or files once per session and reuse those omissions in
  comparison and tool lookups. Continue with verified evidence while the job is
  active. Binary files at accepted-design Markdown paths count as unavailable
  design text. Missing design context prevents approval.
- Share read-only repository file listing/reading and public Go module
  lookup between review features. GitHub source access uses the App token within the
  reviewed repository and its direct, public, same-owner GitHub submodules accessible
  to the installation. Recheck public visibility before every submodule lookup;
  reject private targets and missing visibility metadata before source reads.
  Use the explicit `read_submodule` tool to resolve metadata at the reviewed head and read the pinned gitlink commit through GitHub source methods.
  Normal file lookups for declared submodule paths delegate to the same operation.
  On follow-ups, `read_base_sources` reads regular files at the reviewed base
  under the same consent and revision guard, including when the base is unchanged.
  Do not follow nested
  submodules, fetch submodule URLs directly, or browse arbitrary GitHub repositories.
  Go language/compiler claims can consult official
  release notes selected from the reviewed go.mod version, using a trusted
  credential-free documentation endpoint; treat reference text as data. Resolve module versions/replacements and checksums
  from reviewed go.mod/go.sum; verify archives without extracting or executing
  them. Resolve only declared dependencies and their replacements through the
  trusted public proxy; never send GitHub or provider credentials there. Workspaces and
  vendoring require repository source inspection instead of assuming proxy content.
  A dependency path proven absent from a verified archive returns "not found"
  without marking source unavailable. External source and references are best
  effort: disclose unavailable or unverified source without sharing unverified
  contents, but do not automatically block approval for these gaps. Completion
  requires high confidence in the changed repository code; material unresolved
  questions can still make the assessment incomplete.
- Use one conversation bounded by its inference and job deadlines,
  without response storage, inference retries, fallback, execution tools, or a
  review-specific output-token cap. Recheck current consent and revisions before
  each source lookup and each provider request. A review conversation receives
  half the remaining feature job budget; individual HTTP requests retain their
  own deadlines and summaries keep their one-minute default. Aggregate actual token usage;
  any missing usage makes the total unavailable. Keep source caching in the
  `github.Service` caching decorator, using the configured go-service `ttlcache`
  backend directly for validated trees by repository identity and revision and blobs by
  repository identity and blob SHA across review commands.
  Bypass shared source caching when the cache is omitted or another backend is configured.
  Keep cache retention optional; cache failures and size limits must not limit
  source reads. Retain per-request inspected-source and omission records. Keep
  public dependency/reference caches per request; never cache consent or mutable
  PR state.
  Provider requests, source calls, directory/package reads, and module archives
  have no local count or size caps; add limits only when operational evidence
  justifies them.
  Ruby source access is an exception: a shared byte budget per review bounds
  downloads and archive reads, including cached rescans, to contain compressed
  gem resource exhaustion. Stream unrelated files and disclose budget exhaustion
  as unavailable external source; keep cancellation and archive verification.
  Keep working source and public dependency caches in memory without checkout
  or archive extraction; shared source retention stays in the configured in-memory cache.
  Disclose omissions and actual inspected source independently of model prose.
  Never load repository/remote skills as instructions. Do not add merge-base
  traversal or citation-verification machinery. Decode a small result with an
  overview, findings containing prose bodies and optional path/line/side, and a
  completion boolean. The worker requests changes for P0 through P2 findings.
  Approval requires completion, no missing text patches or unavailable repository
  source, and only P3 findings or no findings. Otherwise publish a comment without
  a success marker so a new request can retry.
  Confirmed binary files outside accepted-design Markdown paths and their
  absent patches are disclosed separately without preventing approval.
  Publish one native review pinned to the reviewed head, with a separate
  inline conversation for each new P0 through P2 finding placed on the fetched
  patch. List inline blockers below the decision alongside findings without valid
  placement and still-open findings with links to their original reviews. Put
  P3 findings in an expandable section of the review body. Escape model
  text and deactivate links/mentions in every body.
  Keep verdict policy in assessment and API event validation in github. Serialize
  final checks and publication across review commands within one process. An
  approval-eligible review must preserve any other feature's authenticated request
  for changes across revisions, restart, or feature disablement, until a later
  assessment that meets the severity and context requirements for approval or a
  human dismissal clears that feature. Discover feature identities from
  authenticated result markers without a verdict allowlist.
  Incomplete reviews do not clear findings. All features
  share one GitHub reviewer identity; repository rules own merge enforcement and
  publication still has a final-read race. Never request merge automation.
- After draining its manual queue, each enabled review job uses its remaining
  deadline to reconcile its own App-rooted inline threads on open PRs, including
  drafts, with both branches in the repository, independently of auto-merge, `hold`,
  topics, and old requester consent. Ignore forks and recheck both branch repositories
  before resolution. Resolve only unresolved threads GitHub marks outdated and the viewer
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
  results do not suppress a new request. Carry earlier blockers from the latest
  authenticated same-feature assessment when it has a usable follow-up record.
  Compare saved patch and source fingerprints with current patches and
  previously inspected paths at the reviewed head and base. Disclose unavailable
  comparisons while preserving current evidence and earlier blocker records.
  Incomplete requests for changes remain blocking and retain their own marker so
  maintainers can retry them on the same revisions. Failures carry no success
  marker. Do not automatically retry ambiguous publication; later human requests
  first check existing results.
- Report request, head/base SHAs, model, and provider token usage; missing usage
  stays explicit. Do not estimate dollar costs or request billing credentials.
  Model accuracy remains unestablished; evaluate approved data before broad use.
  Automated scenarios use fake providers only.
