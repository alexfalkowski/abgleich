# Contributing to Abgleich

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
  retaining scannable emoji headings and concise examples. Record agreed project
  decisions in the area files linked from the [accepted-design index](accepted-design.md).
  Shared language and workflow rules live in [bin/AGENTS.md](../../bin/AGENTS.md)
  and its skills.
- Read the complete README before and after editing it. `docs/configuration.md`
  owns worker settings, YAML examples, defaults, validation, and credential setup;
  feature guides own behaviour and safeguards, and `docs/operations.md` owns
  monitoring and recovery. Link to the owning guide instead of duplicating its
  guidance, and check moved links, conflicting examples, and repeated defaults.

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
