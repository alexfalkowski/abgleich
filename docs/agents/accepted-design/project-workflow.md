# Accepted design for the project workflow

Part of the [accepted-design index](../accepted-design.md).

## Worker behavior tests

Ruby Cucumber and Nonnative own worker behavior coverage. Keep `main_test.go`
as the standard `app.Run` launcher, without synthetic jobs or test-only startup
switches. Exercise the worker against independent fake GitHub and OpenAI APIs
on the shared local listener with disposable test credentials. The harness
never mutates real GitHub repositories or calls a live AI provider.

[Worker launcher](../../../main_test.go) ·
[Feature suite](../../../test/features) ·
[Development guide](../../development.md) ·
[CircleCI configuration](../../../.circleci/config.yml)

## Immutable scenario configuration

Every Cucumber and benchmark scenario selects a complete, checked-in YAML
configuration from `test/.config/` through `Abgleich.use_config`. Keep
`server.yaml` as the default. Select explicit profiles for other settings,
including changes across restarts. The Nonnative command Proc passes the
selected path directly to the worker. Test steps leave checked-in profiles
intact and avoid runtime YAML merging, temporary configs, and configuration
overrides through environment variables.

[Configuration selection](../../../test/lib/abgleich.rb) ·
[Nonnative command](../../../test/features/support/api.rb) ·
[Default profile](../../../test/.config/server.yaml)

## Model evaluation

Fake-provider scenarios verify worker integration. Model accuracy remains
unestablished. Evaluate approved data before broad use and keep live
evaluation outside automated scenarios.

[Evaluation guidance](../../development.md)
