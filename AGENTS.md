# AGENTS.md

## Shared guidance

Use [bin/AGENTS.md](bin/AGENTS.md) for shared skills and cross-repository defaults.

In Go code, use `strings.Empty` from `github.com/alexfalkowski/go-service/v3/strings`
for explicit empty string values and `strings.IsEmpty` for emptiness checks.
Use named struct types in production Go code. Anonymous structs are allowed in
tests and GitHub GraphQL payloads.

## Project purpose

Abgleich is a background worker that enables auto-merge or merges ready Dependabot
and labeled pull requests and keeps eligible auto-merge branches current.
Repositories opt in to that automation through their auto-merge setting.
Comment commands also provide automation status, optional AI PR summaries, and
AI code, security, and test reviews enabled through worker configuration. GitHub
performs all repository mutations; the worker does not clone repositories or execute Git. Each auto-merge
run reconciles fresh remote state; pending summary and review requests are held in memory.

## Accepted design

Read the [accepted-design index](docs/agents/accepted-design.md) and the relevant
[area files](docs/agents/accepted-design/) before proposing a change or recording
a finding. Apply each decision to its documented scope and record owner-agreed
choices in the owning area file.

## Contributing

Follow the [contributor guidance](docs/agents/contributing.md) for configuration,
documentation, Cucumber fixtures and validation. Start with `make help` at the
repository root to discover supported checks.
