# Accepted design for AI summaries

Part of the [accepted-design index](../accepted-design.md).

## Consent and advisory scope

Summaries are optional and advisory. Require an open, non-draft PR with both
branches in the repository and a fresh, unedited human `/ab summarise`
comment from someone with current write or admin permission. That comment
authorizes source sharing independently of auto-merge settings, labels, and
repository topics. Recheck authorization and both branch revisions before
provider submission and publication. Discard stale explanations.

Include every changed file and patch GitHub returns, without local file-count
or patch-size limits. Summary evidence consists of PR metadata and patches.
Treat source as untrusted data. Generation receives no tools or GitHub
credentials and cannot execute code or request merge automation. Use explicit
provider credentials with response storage and automatic inference retries
disabled. Log generation outcomes through the AI decorator without source,
response text, or credentials.

[Summary command](../../../internal/summarise/command.go) ·
[Summary context](../../../internal/summarise/context.go) ·
[Generation logging](../../../internal/ai/logging.go) ·
[Summary guide](../../summaries.md)

## Queue and deduplication

Use the shared in-memory command queue and an independent summary job to keep
inference outside the webhook command slot. The inference timeout is at most
one minute and half the remaining job budget. A restart loses pending requests.

Check for an existing App summary before generation and publication. One
published App summary prevents another model call even after restart or new
commits. Reviews have their own revision-based history.

[Summary processing](../../../internal/summarise/process.go) ·
[Summary scenarios](../../../test/features/summarise/commands.feature)
