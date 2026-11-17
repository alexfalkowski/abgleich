# 🛠️ Development

[← Back to README](../README.md)

## 🗂️ Structure

The project follows the structure in [golang-standards/project-layout](https://github.com/golang-standards/project-layout).

## ✍️ Style

This project favours the [Uber Go Style Guide](https://github.com/uber-go/guide/blob/master/style.md)

## 🏁 Setup

After completing the [quickstart's bootstrap steps](../README.md#-bootstrap-and-build), run the auto-merge features:

```sh
make features feature=features/automerge
```

Use `make help` for commands and CI feature groups, such as
`make features-dependabot` and `make features-summaries`.
`make features-codereview`, `make features-secreview`, and
`make features-testreview` select separate scenarios
and example groups, including their cases in the shared `features/review` files.
Cases exercising multiple review commands belong to one group so they run once across CI.
CI splits code review between `make features-codereview-results` for source access,
finding placement, and verdicts, and `make features-codereview-workflow` for the
remaining command and lifecycle cases. The `@review-results` tag selects the first
group; `make features-codereview` still runs both groups locally.
Shared review scenarios are grouped by domain in
[`test/features/review`](../test/features/review). Names such as
`authorization.feature`, `inspection.feature`, `manifest.feature`, and
`verdict.feature` match the corresponding files in
[`workflow`](../internal/review/internal/workflow). Run one domain from the
repository root:

```sh
make features feature=features/review/authorization.feature
```

Run groups one at a time: the harness shares ports. Untagged features belong to the
worker group; group definitions live in [Makefile](../Makefile).
Keep `@manual` and `@startup` for Nonnative lifecycle hooks: `@manual` stops workers
started by scenario steps, while `@startup` starts and stops the worker automatically.

The Ruby Cucumber/Nonnative harness runs the worker against independent fake
GitHub REST/GraphQL and OpenAI APIs on one local listener, with accelerated
schedules and disposable credentials. It never mutates real repositories or calls
a live model. Keep committed fixtures in `test/secrets/` unchanged; failure
scenarios select separate invalid or missing paths. Every existing and new
Cucumber or benchmark scenario selects a complete, immutable checked-in YAML
configuration from `test/.config/` through `Abgleich.use_config`. Steps never
rewrite configuration files or generate runtime configurations; worker startup
and restarts read the selected file directly. Logs and results go to
`test/reports/`. Browse [features](../test/features) for behavior coverage and
[CI](../.circleci/config.yml) for build and release workflows.

## 🔬 Code-review evaluation

Fake-provider scenarios verify integration, not model accuracy. Before broad use,
evaluate operator-approved PRs with known bugs and clean changes, recording useful
findings, misses, false positives, latency, and token usage. Validate source
selection as well as final answers: useful context must be found automatically. Live calls require separate authorization
and never run in CI. Retain the complete inputs, lookup transcripts, raw answers,
and grading notes so the results can be inspected and reproduced.
Before relying on approvals, measure false approvals on known faulty changes,
missed high-severity bugs, and false requests for changes on clean controls.
Include incomplete-context cases and caller/dependency bugs requiring automatic
source discovery. Re-evaluate after model, prompt, or source-selection changes;
fake-provider scenarios establish verdict integration only.
Include Ruby library fixes and mixed Go/Ruby Cucumber changes, with clean controls,
and verify that the reviewer finds relevant steps, hooks, manifests, and locked gem source.
Shared prompt rules live in `internal/review/internal/workflow/prompts`; each
feature retains its review focus. Language/framework providers in
`internal/review/internal/sources` own their matching rules, guides, and tools.
Each provider's module contributes a `provider.Result` to the `review_sources` DI
value group. Add that module to `sources.Module`; registry selection and workflow
need no changes for another language.

## 🛡️ Security-review evaluation

The [security prompt](../internal/review/sec/prompt.txt) adapts review methods from
[Cloudflare's attack classes](https://github.com/cloudflare/security-audit-skill/blob/c1c8a8c1471069fb0e188eeaff69b8e8db6564a8/skills/security-audit/ATTACK-CLASSES.md),
[UnitOneAI's threat modeling](https://github.com/UnitOneAI/SecuritySkills/blob/70bc259bb01abb3015ad2ad859ad5253cbf0bcab/skills/appsec/threat-modeling/SKILL.md),
and [Red Hat's finding verification](https://github.com/RedHatProductSecurity/prodsec-skills/blob/b55c6b9c3392645f62da59a447c29473c413e371/module/skills/fp-check/SKILL.md)
to the worker's static source review. These are pinned maintenance references;
runtime review instructions ship with the worker.

Fake-provider scenarios verify integration only. Before broad use, evaluate the
unchanged security prompt on approved real vulnerable changes and safe controls.
Record expected outcomes before calls, keep grading notes out of model input, and
retain raw answers, confirmed findings, misses, false positives, latency, and
actual token usage. Use fresh cases beyond prompt/source development examples,
including vulnerabilities requiring helpers or dependencies. A small evaluation
cannot establish comprehensive security accuracy.
Live calls need separate authorization and never run in CI.

For prompt changes, freeze the current and proposed prompts and expected outcomes
before making calls. Compare them with the same model, reasoning effort, source
tools, and conversation deadline. Include vulnerable changes, fixed or otherwise
safe controls, and material unresolved questions; keep advisory text, expected
findings, and grading notes out of model input. Record false approvals, missed
vulnerabilities, false findings, incomplete assessments, source lookups, latency,
and actual token usage separately for each prompt. Inspect every reported finding
against the source rather than counting any nonempty answer as a detection.
Retain the complete inputs, prompt and source revisions, lookup transcripts,
raw responses, and grading notes outside version control. Public historical cases
may be familiar to the model; authored examples and a small comparison establish
limited regression evidence, not general security accuracy. Add source tools only
when inspected misses demonstrate a retrieval gap.

## 📝 Changes

Release history is available in [GitHub releases](https://github.com/alexfalkowski/abgleich/releases).

## 🧪 Test-review evaluation

The [test-review prompt](../internal/review/test/prompt.txt) adapts the shared
[testing principles](../bin/skills/testing-standards/references/principles.md),
[test-gap criteria](../bin/skills/test-gaps-find/references/find-rules.md), and
[go-linux test guidance](https://github.com/alexfalkowski/go-linux/blob/master/AGENTS.md)
into worker-owned static review instructions. Repository-specific assertion
libraries and Linux fixture conventions are not universal requirements.
Runtime reviews never load those skills as instructions.

Fake-provider scenarios establish integration, not detection quality. Before
broad use, evaluate approved PRs with known ineffective assertions, missing
behavior coverage, isolation/lifecycle failures, and clean controls. Include
Go and Ruby/Cucumber harnesses, production-only changes already covered by
existing scenarios, and behavior-preserving changes that need no additional tests.
Include `.gocov` exclusions already present or added by the PR, regex patterns,
and changes mixing excluded code with genuine test gaps in other paths. Expect no
missing-test demands for excluded code, while retaining findings for ineffective
changed tests and gaps outside the exclusions.
Record expected outcomes before live calls and inspect every finding against the
source. Follow the [code-review evaluation process](#-code-review-evaluation) for
source discovery, misses, false positives, incomplete assessments, latency, and
token usage. Live evaluation requires separate authorization and never runs in CI.

For prompt comparisons, freeze the original PR head/base and both prompts, then
repeat reviews with the same model, effort, source tools, and conversation
deadline. Keep expected findings and grading notes out of model inputs. Measure
how many confirmed gaps are found in the first review, misses, false positives,
and variation between runs, alongside latency and token usage. Distinguish gaps
already present in the original revision from issues introduced by later fixes.
Include fresh PRs and clean controls beyond the examples used to develop the
prompt; fewer findings or earlier approval alone does not demonstrate improvement.
