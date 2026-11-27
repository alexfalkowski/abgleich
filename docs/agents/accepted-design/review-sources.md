# Accepted design for review sources

Part of the [accepted-design index](../accepted-design.md).

## Repository evidence

Include PR metadata, all available changed-file patches from complete GitHub
pagination, and complete changed-file text where available. Read at the
reviewed head SHA and use the reviewed base SHA for deleted files. Require a
complete tree, read regular files, and reject symlinks, binary data, and
inconsistent blob responses. Preserve patches when full source is unavailable.
Disclose confirmed binary exclusions separately, including absent patches.
Binary exclusions do not prevent approval.

Share read-only repository listing, file reads, and dependency tools across
review features. Follow-ups use `read_base_sources` for regular files at the
reviewed base, including when the base is unchanged. Every source tool call
uses the workflow's consent and revision guard.

Use worker-owned instructions and treat repository and external text as
untrusted data. Keep reviews static, with no code execution, merge-base
traversal, or citation verification. Report omissions and actual inspections
independently of model prose.

[Review context](../../../internal/review/internal/workflow/context.go) ·
[Changed-file reads](../../../internal/review/internal/sources/changes.go) ·
[Source inspection scenarios](../../../test/features/review/inspection.feature)

## Submodules

App-authenticated source reads are scoped to the reviewed repository and its
direct, public, same-owner GitHub submodules accessible to the installation.
Recheck public visibility before every submodule lookup. Reject private
targets and missing visibility metadata before reading source.

`read_submodule` resolves metadata at the reviewed head and reads the pinned
gitlink through GitHub. File lookups inside declared submodule paths use the
same operation. Nested submodules, direct fetches of submodule URLs, and
arbitrary GitHub repositories are outside the source boundary.

[Submodule reads](../../../internal/review/internal/sources/submodule.go) ·
[Submodule scenarios](../../../test/features/review/submodule.feature)

## Public dependencies and references

Use official Go release notes selected from the reviewed `go.mod` version
through a trusted, credential-free endpoint. Resolve only declared Go
dependencies and their replacements from the reviewed `go.mod` and `go.sum`.
Verify archives from the trusted public proxy without extracting or executing
them. Keep GitHub and provider credentials off the proxy. Workspaces and
vendoring require repository inspection. A path proven absent from a verified
archive returns `not found` without marking source unavailable.

External dependencies and references are best effort. Disclose unavailable
or unverified sources without sharing unverified contents. These gaps alone
leave approval possible when the reviewer has high confidence in the changed
repository code. Material unresolved questions can make an assessment
incomplete.

Provider requests, source calls, directory and package reads, and Go module
archives have no local count or size caps. Revisit limits when operational
evidence justifies them. Ruby gem lookups already use a shared per-review byte
budget for downloads and archive reads, including cached rescans, to bound
compressed archive resource use. Stream unrelated files, preserve cancellation
and verification, and disclose budget exhaustion as unavailable external source.

[Go source provider](../../../internal/review/internal/sources/golang/doc.go) ·
[Ruby source provider](../../../internal/review/internal/sources/ruby/doc.go) ·
[Manifest scenarios](../../../test/features/review/manifest.feature) ·
[Ruby source scenarios](../../../test/features/review/ruby.feature)

## Source retention

Keep working source, dependency and reference caches, and inspection records
in memory for each review, without checkout or archive extraction. The
`github.Service` caching decorator can additionally share validated trees by
repository identity and revision, and validated blobs by repository identity
and blob SHA. Use the configured go-service `ttlcache` backend directly.
Bypass shared source caching when omitted or configured with another backend.
Cache failures and value limits must not restrict source reads. Consent and
mutable PR state always require current reads.

[GitHub source cache](../../../internal/github/cache.go) ·
[Source cache scenarios](../../../test/features/review/cache.feature)
