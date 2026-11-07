[![CircleCI](https://circleci.com/gh/alexfalkowski/abgleich.svg?style=svg)](https://circleci.com/gh/alexfalkowski/abgleich)
[![codecov](https://codecov.io/gh/alexfalkowski/abgleich/graph/badge.svg?token=S9SPVVYQAY)](https://codecov.io/gh/alexfalkowski/abgleich)
[![Go Reference](https://pkg.go.dev/badge/github.com/alexfalkowski/abgleich.svg)](https://pkg.go.dev/github.com/alexfalkowski/abgleich)
[![Stability: Active](https://masterminds.github.io/stability/active.svg)](https://masterminds.github.io/stability/active.html)

# 🔄 Abgleich

Abgleich enables auto-merge or merges ready Dependabot and labeled pull requests,
and rebases stale auto-merge branches across a GitHub account's opted-in repositories.
Optional features provide branch cleanup, automation status, AI PR summaries,
and AI code and security reviews with GitHub verdicts.
GitHub performs all repository mutations; the worker never clones repositories or
executes Git.

## 🚀 Quickstart

Run these steps from the repository root.

### ✅ Prerequisites

- Go matching [go.mod](go.mod).
- Git with GitHub SSH access for the `bin/` submodule.
- Ruby with Bundler for dependency setup and the Cucumber harness.
- Make, plus `curl` for the readiness check below.

### 🏗️ Bootstrap and build

```sh
git submodule sync
git submodule update --init
make dep
make build
```

### ▶️ Configure and start

Save the [base configuration](docs/configuration.md#-base-configuration) as
`config.yaml`, fill in your owner, App IDs, and private-key source, and complete
[App setup](docs/configuration.md#-github-app-setup). [Configuration](docs/configuration.md)
also covers optional features. Start the worker:

```sh
./abgleich server -config file:config.yaml
```

Checks begin immediately. In another terminal, verify readiness:

```sh
curl --fail --silent --show-error http://localhost:11000/abgleich/readyz
```

HTTP 200 with body `SERVING` confirms process readiness. Use structured logs for
job outcomes; failed branch checks do not affect readiness.

## 🎯 Automation

Repositories opt in through **Allow auto-merge**. Abgleich enables or completes
squash merging for eligible Dependabot and `automerge`-labeled PRs and rebases
stale auto-merge branches. Both branches must belong to the repository; fork PRs
are ignored. The exact `hold` label pauses merge automation.

See [merge automation](docs/automation.md) for eligibility, label consent,
event-triggered checks, branch cleanup, and API usage.

## 💬 Comment commands

With [webhooks configured](docs/configuration.md#-comment-webhooks), post a new
unedited PR comment as a maintainer with current write or admin access:

| Command | Action |
| --- | --- |
| `/ab automerge` | Request auto-merge through the `automerge` label. |
| `/ab status` | Explain the PR's current automation policy. |
| `/ab summarise` | Queue an optional AI summary. |
| `/ab review` | Queue every configured AI review. |
| `/ab codereview` | Queue only the configured code review. |
| `/ab secreview` | Queue only the configured security review. |

See [comment delivery and reactions](docs/commands.md),
[summary behavior](docs/summaries.md), and [review behavior](docs/reviews.md).
An authorized AI command consents to sharing PR source with the configured
provider, including private source. Read the feature safeguards before enabling it.

## ⚙️ Configuration

See [configuration](docs/configuration.md) for GitHub App setup, credentials,
webhooks, optional features, source access, timeouts, and defaults.

## 🛠️ Operations

See [operations](docs/operations.md) for health endpoints, monitoring, deployment,
and recovery after failures or restarts.

## 🧩 Design

See [design](docs/design.md) for package boundaries, reconciliation, command
processing, and job registration.

## 🛠️ Development

See [development](docs/development.md) for the Cucumber harness, local validation,
and AI review evaluation. Release history is available in
[GitHub releases](https://github.com/alexfalkowski/abgleich/releases).
