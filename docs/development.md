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
`make features-codereview` and `make features-secreview` select separate scenarios
and example groups, including their cases in the shared `features/review` files.
Cases exercising both commands belong to one group so they run once across CI.
Run groups one at a time: the harness shares ports. Untagged features belong to the
worker group; group definitions live in [Makefile](../Makefile).

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

## 🛡️ Security-review evaluation

Fake-provider scenarios verify integration only. Before broad use, evaluate the
unchanged security prompt on approved real vulnerable changes and safe controls.
Record expected outcomes before calls, keep grading notes out of model input, and
retain raw answers, confirmed findings, misses, false positives, latency, and
actual token usage. Use fresh cases beyond prompt/source development examples,
including vulnerabilities requiring helpers or dependencies. A small evaluation
cannot establish comprehensive security accuracy.
Live calls need separate authorization and never run in CI.

## 📝 Changes

Release history is available in [GitHub releases](https://github.com/alexfalkowski/abgleich/releases).
