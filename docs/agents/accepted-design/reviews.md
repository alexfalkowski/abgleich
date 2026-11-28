# Accepted design for AI reviews

Part of the [accepted-design index](../accepted-design.md).

## Per-feature requests and consent

Keep `/ab codereview`, `/ab secreview`, and `/ab testreview` optional under
`review.code`, `review.sec`, and `review.test`. Each uses the shared workflow
with its own queue and job. Require a fresh, unedited human comment, current
write or admin permission, and an open, non-draft PR with both branches in
the repository. Eligibility is independent of topics, `hold`, and auto-merge
settings.

Authorize review deliveries independently. Share two webhook delivery slots
across all review commands and return HTTP 503 before calling GitHub when both
are occupied. Keep the serial commands' delivery slots independent. Bound
accepted work in each feature's queue and keep inference independent of the
webhook.

`/ab review` requests every enabled review constructed by `workflow.Factory`.
Keep the original aggregate comment as consent at every recheck. Preserve
per-feature queues, jobs, results, and deduplication across aggregate and
individual requests. Attempt every enabled feature and retain accepted work
when another fails. Report overall acceptance only when at least one feature
is enabled and all enabled features accept.

[Review factory](../../../internal/review/internal/workflow/factory.go) ·
[Authorization scenarios](../../../test/features/review/authorization.feature) ·
[Aggregate scenarios](../../../test/features/review/factory.feature)

## Inference boundaries

Recheck current authorization, PR eligibility, and both branch identities and
revisions before source submission, each source tool lookup, each provider
request, and publication. Discard stale results.

Use one conversation bounded by its inference and job deadlines, without
response storage, inference retries, fallback, execution tools, or a
review-specific output-token cap. The conversation receives half the remaining
feature job budget. Individual HTTP requests retain their own deadlines.

Report the request, head and base SHAs, model, and actual provider token usage.
Any missing usage makes the total unavailable. Keep billing credentials and
dollar estimates outside the workflow.

[Inference](../../../internal/review/internal/workflow/inference.go) ·
[Inference scenarios](../../../test/features/review/inference.feature) ·
[Review source boundaries](review-sources.md) ·
[Review guide](../../reviews.md)

## Verdicts and publication

Treat PR descriptions as context for checking accepted trade-offs and
verification limits against inspected behavior and repository requirements.
Attribute reported validation and explain relevant residual risks in the
overview when they fall below the feature's blocking criteria. Require
supporting repository evidence for claimed deferrals. Keep independently
supported defects and unmet requirements in findings.

Decode a structured result with an overview, a completion boolean, and findings
with prose bodies and optional path, line, and diff side. P0 through P2
findings request changes. With only P3 findings or none, approval requires a
complete assessment, earlier blockers accounted for, and all required text
patches and repository source available. Otherwise publish a comment without
a success marker so a new request can retry.

Test review owns standalone regression-coverage gaps. It requests changes for a
missed known material regression, weakened protection for a material contract,
a material failure of the test run, or an unverified critical safeguard with a
concrete high-impact failure path. Use the overview for other concrete risks
and omit routine coverage suggestions.
Reassess earlier blockers under the same criteria and require evidence that a
material failure remains unprotected before blocking on another test gap.

Publish one native review pinned to the reviewed head. Open a separate inline
conversation for each new P0 through P2 finding that can be placed on the
fetched patch. List inline blockers below the decision alongside findings
without valid placement and still-open findings linked to their original
reviews. Put P3 findings in an expandable section. Escape model text and
deactivate links and mentions in every body.

Assessment owns verdict policy and `github` validates API events. All review
features share one GitHub reviewer identity, so serialize their final checks
and publication within the worker process. Withhold approval while another
feature's authenticated request for changes stands, even after a new revision,
restart, or feature disablement. A later assessment that meets the severity
and context requirements for approval or a human dismissal clears that
feature's request for changes. Incomplete assessments leave blockers in place.
Discover feature identities from authenticated result markers without a verdict
allowlist.

GitHub repository rules enforce merge requirements. Reviews never request
merge automation. GitHub has no atomic review-state guard, so another
instance or review can race publication after the final read.

[Assessment](../../../internal/review/internal/assessment/doc.go) ·
[Final publication](../../../internal/review/internal/workflow/publication.go) ·
[Verdict scenarios](../../../test/features/review/verdict.feature) ·
[Publication scenarios](../../../test/features/review/publication.feature)

## Result history and follow-ups

Deduplicate successful App results using worker-generated markers keyed by
feature, case-insensitive target repository, PR identity, head, and base.
Read complete discussion and native review history and require
API-authenticated App attribution. Source-enabled reviews use v2 markers so
patch-only results do not suppress them. New revisions allow new requests.
Re-review also remains possible when a newer assessment supersedes the result
for a restored revision. Summaries keep independent history.

Carry earlier blockers from the latest authenticated same-feature assessment
when it has a usable follow-up record. Compare saved patch and source
fingerprints with current patches and previously inspected paths at the reviewed
head and base. Disclose unavailable comparisons and preserve current evidence
and earlier blocker records. Incomplete requests for changes remain blocking
and use a separate marker that permits a retry on the same revisions.
Failures carry no success marker. After ambiguous publication, a later
human request checks existing results before generating another review.

[Authenticated assessment history](../../../internal/review/internal/history/assessment.go) ·
[Follow-up scenarios](../../../test/features/review/followup.feature)

## Outdated threads

After draining its manual queue, each scheduled review run spends its
remaining deadline on its own App-rooted inline threads. Queue-triggered runs
process manual requests only. Maintenance covers open PRs, including drafts,
with both branches in the repository. It is independent of auto-merge,
`hold`, topics, and original requester consent.

Complete repository discovery and each PR's thread pagination before attempting
resolutions.
Refresh both branch repositories, PR and thread identity, ownership, and state
before each resolution. Resolve only unresolved threads GitHub marks outdated
and the viewer can resolve. Authenticate the root and parent review and match
the feature's result namespace. Preserve queue and maintenance failures and
continue other candidates after an individual failure. After a failed or
ambiguous mutation, later scheduled runs rediscover current state.

Outdated placement alone gives no evidence that the finding was fixed. Keep
current threads open and inference manual. Maintenance uses the existing
feature job and configuration, with scheduled discovery and no persistent
worker state. GitHub has no expected-state guard for resolution, so changes
after the final read can race the mutation.

[Thread reconciler](../../../internal/review/internal/threads/reconciler.go) ·
[Thread scenarios](../../../test/features/review/threads.feature)
