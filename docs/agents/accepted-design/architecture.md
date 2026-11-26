# Package boundaries

Part of the [accepted-design index](../accepted-design.md).

## Package ownership

| Package | Owns |
| --- | --- |
| `internal/cmd` | CLI command registration and composition of package modules. |
| `internal/commands` | PR comment parsing, named handler dispatch, installation validation, shared human-comment authorization, deadline-aware command serialization, background queues, and reaction feedback. |
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
