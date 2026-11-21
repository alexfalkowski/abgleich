# 🔍 AI reviews

[← Back to README](../README.md)

## 🔍 Request all configured reviews

Post `/ab review` to request every enabled review, currently code, security, and test
reviews. It uses the existing [review configuration](configuration.md#-code-review-settings)
and needs no additional setting or job. Omitted review features are skipped;
with no reviews configured, the command receives rejection feedback.

The original unedited human comment authorizes source sharing for every enabled
review. The command checks current consent and PR eligibility once for all features
within the webhook deadline. Each workflow independently rechecks current
write/admin permission, PR eligibility, consent, and head/base revisions before
source access and publication, using the safeguards below. Review requests remain available with `hold` and
without auto-merge opt-in; forks are rejected.

Each feature retains its own model, queue, schedule, verdict, and native review.
The command attempts every configured review even if another declines or fails.
Acceptance feedback requires every review to be queued or already complete.
Partial acceptance retains queued work; a full queue returns HTTP 503. Reissue
`/ab review` after recovery to request the remaining reviews. Pending and
successful results are reused independently, including results requested through
`/ab codereview`, `/ab secreview`, or `/ab testreview`. Individual commands also reuse results
from `/ab review`. [Restart and failure recovery](operations.md#-code-review-recovery)
still applies per feature. Summaries are requested separately with `/ab summarise`.

## 🔬 AI code reviews

With [code reviews configured](configuration.md#-code-review-settings), post `/ab codereview` to
request a review of an open, non-draft PR with both branches in the target
repository. Fork requests are rejected without AI inference.
A new, unedited human comment from a current writer or administrator authorizes
sharing PR metadata, patches, whole changed-file text, and relevant repository
source with OpenAI, including private source in the reviewed repository. Submodule
source access is limited to public targets described under
[source access](configuration.md#-review-source-access).
No topic, auto-merge setting, label, or removal of `hold` is needed.
Approve provider data handling and billing before enabling reviews.

The independent `codereview` job processes at most four queued requests serially;
a full queue returns HTTP 503. Accepted requests wake the job immediately; requests
received during active work wait for it to finish. These queue runs process only
manual requests; repository-wide thread maintenance retains its configured interval.
The complete source-lookup conversation uses
half the remaining job budget, reserving time for final checks and publication.
With the default two-minute job, that is at most one minute. Increase the feature
`timeout` for deeper reviews; individual HTTP timeouts remain independent. Source is untrusted data; the model receives read-only
source tools without GitHub credentials. Each review uses one conversation bounded by its deadline, with no
response storage or automatic retries, and no review-specific output-token cap.
Large changes can exceed provider context limits or worker deadlines.
Reviews focus on the PR's changes and related source in the reviewed repository,
with verified Go modules, Ruby gems, and Go language references available through the
[source tools](configuration.md#-review-source-access). Those tools also inspect
permitted direct submodules at their pinned commits. Reviews do not browse arbitrary
GitHub repositories; missing implementations outside those tools do not prevent completion.
The code-review prompt requires input and state handling findings to trace a
supported triggering path through producers and callers, accounting for upstream
generation and validation. Hypothetical inputs outside those preconditions alone
do not justify findings unless an explicit contract or trust boundary requires
handling them. The reviewer must challenge candidates against inspected evidence
that they are not bugs and describe the surviving trigger and prerequisites.

Each review combines shared evidence/result rules with its feature-specific
focus. The worker selects separate Go, Ruby, and Cucumber providers from the reviewed
repository's file tree and changed paths, including deletions. Mixed repositories
receive every applicable guide and source tool; unrelated language support is
omitted. These guides ship with the worker;
repository skills are never loaded as instructions. Ruby guidance follows
manifests and supported versions; Cucumber guidance traces scenarios through
steps, support hooks, configuration, and assertions.
These are static source reviews: running tests, rendering Helm manifests,
deploying, and inspecting live services are outside their scope. Lack of execution
alone does not make a review unfinished; material unresolved source gaps still do.

The App submits one native review with an overview and a separate inline
conversation for each new P0, P1, or P2 finding that can be placed on the supplied
diff. The overview lists P3 findings, findings without reliable diff placement,
and still-open findings from earlier reviews. It shows their reported locations
when supplied and links still-open findings to the original review. Each finding has a short
🐛 title with explicit severity (`P0` critical, `P1` high, `P2` medium, or `P3` low),
followed by separate
🔎 **Trigger**, 💥 **Impact**, and 🛠️ **Suggested fix** paragraphs. The worker
renders these labels and escapes each model-provided field. Security-review
triggers include attacker-controlled input and prerequisites. A review with no
supported findings still publishes its overview. Review prompts prioritize
concrete issues that matter to the current PR. The overview can briefly mention
a concrete risk that belongs to separate work when it helps the maintainer
make a decision.
The worker selects the verdict from the structured result:

| Assessment | GitHub verdict |
| --- | --- |
| Any P0, P1, or P2 finding, including overview-only findings | `REQUEST_CHANGES` |
| Only P3 findings or no findings, with model-reported completion, earlier blockers accounted for, and no missing text patches or unavailable repository source | `APPROVE` |
| Only P3 findings or no findings, with incomplete investigation, unverified earlier blockers, or missing required context | `COMMENT` (approval withheld) |

The model assigns severity from the consequence of a supported trigger. P3
covers localized defects worth correcting in the current PR. P2 covers material
failures of supported behavior and consequential gaps in protection for a
changed material behavior or safeguard. If missing evidence could move a
finding across this boundary, the prompt tells the model to report the review
as incomplete.

The overview shows the worker's assessment and decision separately from model
prose, with ✅ for approval, ❗ for requested changes, and ⚠️ for comments.
**Review unfinished** means the model reports an incomplete investigation or
earlier blockers remain unverified. **Review completed with limitations** means the model reports
completion but missing patches or unavailable repository source limit the assessment. A
**Limitations** list identifies each missing patch and unavailable repository source. Both
outcomes withhold approval, carry no successful-result marker, and allow a new
manual request on the same revisions. P0 through P2 findings still request
changes after an incomplete investigation, and a new request can retry that
review on the same revisions. Model completion is a self-report.
**External source notes** disclose unavailable dependencies and language references
separately. These gaps alone do not prevent approval when the reviewer has high
confidence in the changed repository code. Material unresolved questions still
require an incomplete assessment.
Confirmed binary files are outside the text review scope. **Excluded binary files**
lists them separately; their omitted contents and patches do not prevent approval.
Every body contains escaped model text, with links and mentions inactive.
Dollar signs remain literal text rather than starting mathematical expressions.
Indented model examples remain plain text instead of forming Markdown code blocks.
Review text is published without credential redaction.
Details identify the request, head/base SHAs, model, and actual token
usage, or unavailable usage; no dollar costs are estimated. Missing patches are
disclosed, along with the source actually inspected and any unavailable reads.
Citations are not verified, and code/tests are not executed. Findings may be incorrect
or incomplete. An approval indicates the review reported no P0, P1, or P2
findings in the inspected scope.
Reviews never label, merge, or rebase. Open inline
conversations can block merging when the repository requires conversation
resolution. A review targets the analyzed head; a push after the final check can
still make it outdated. Evaluate approved data before
[broad use](development.md#-code-review-evaluation).

All review features share one GitHub reviewer identity. Final verdict checks
and publication are serialized within the worker process. An approval-eligible
review publishes a comment while any other feature's latest decisive App assessment
still requests changes, including across new revisions and restarts or after that
feature is disabled. Each feature's history is evaluated independently. A
complete assessment that meets the severity and context requirements for
approval or an explicit human dismissal clears that feature's request for
changes. Feature identities come from authenticated App result
markers, so adding a review requires no verdict allowlist. Commands keep independent
inference and queues. This does not require every review to run before the first approval.

GitHub's [review and branch rules](https://docs.github.com/en/pull-requests/how-tos/review-pull-requests/reviewing-proposed-changes-in-a-pull-request)
determine whether a request for changes blocks merging and whether an App approval
satisfies review requirements. Require CI and human/code-owner reviews separately
where needed, and enable stale-approval dismissal for changed commits. There is no
atomic review-state guard: another instance or review after the final read can race
publication. These verdicts apply to new requests after upgrading; existing
authenticated results are reused without retroactively changing their verdicts.

Authorization, comment identity, PR eligibility, and both branch identities and
revisions are rechecked before submission and publication. Changes discard stale
results and require a new request; already transmitted source cannot be recalled.
Before publication, the worker also checks that the latest authenticated
assessment for the same feature still matches the history used for inference.
A matching successful App result prevents another model call across restarts
when it is still the latest assessment for that feature. Restoring an older
revision permits a new review when a newer assessment superseded its result.
New head or base revisions permit another review. If the latest authenticated
App assessment for the same feature has a usable follow-up record, Abgleich
uses its saved blocker references and fingerprints
to show what changed since that review, including after a force push or rebase.
Consent for each review command also covers sending these saved findings and
changed-path comparisons to the provider. The reviewer receives the full current
PR context and prioritizes verifying earlier blockers and checking the effects of
the new changes. A newly discovered issue in earlier work warrants a finding
when the current diff contains a material defect supported by inspected evidence.
An unverified earlier blocker withholds approval. When a fix removes the
blocking consequence, the reviewer can clear the blocker and report a remaining
P3 issue separately. The review discloses unavailable fingerprint comparisons
and still asks the reviewer to verify the saved findings against current source.

Reviews can carry a compact follow-up record in their raw Markdown, so it
survives worker restarts. The record saves blocker identities and digests.
The worker reads intact finding text from its original authenticated review.
Edited, removed, or unavailable finding text leaves its blocker unverified.
A follow-up that includes earlier finding text checks it again before each
provider request and stops if the text changed or became unavailable.
Incomplete requests for changes carry both new and unverified earlier blockers.
When the record is too large, the worker first drops fingerprints. If the
blockers still cannot fit, it publishes the request for changes as retryable and
stops follow-ups while that record is unavailable. A maintainer can inspect the
review and dismiss it if appropriate before retrying. Legacy reviews without
a follow-up record can receive a full assessment. A dismissal of the latest
assessment ends carry-forward for that feature. If the worker cannot verify
App authorship, it stops the review. Each review feature and the summary have
independent histories.
See [review recovery](operations.md#-code-review-recovery).

If an accepted review's PR merges, closes, or becomes a draft, the next processing
recheck attempts a cancellation comment linking to the original request, provided
the comment remains authorized and the job has time left. This applies while
queued and after generation starts; a request canceled before generation sends
no source to the provider.
An in-flight provider request may finish before cancellation is observed, but its
findings are discarded. Cancellation comments carry no successful-result marker.

After draining its queue, each scheduled review run uses its remaining deadline to
resolve its own App-authored inline threads that GitHub marks outdated on open
PRs with both branches in the repository, including drafts. Fork PRs receive no
thread maintenance. Current threads stay open; replies do not prevent resolution
of an outdated App thread. This needs no new command or inference and
does not depend on topics, `hold`, auto-merge settings, or the original requester's
continued permission. Outdated means the comment's location became stale; it does
not establish that the issue was fixed. A fresh AI review still requires the
corresponding `/ab` command. GitHub provides no expected-state guard for thread
resolution, so changes after the final state and ownership check can race it.

## 🛡️ AI security reviews

With [security reviews configured](configuration.md#-security-review-settings), post `/ab secreview`
to request a focused security review, including in private repositories. Both
branches must belong to the target repository; fork requests are rejected without
AI inference. A new, unedited human comment from a current writer or administrator
consents to sharing PR metadata, patches, whole changed files, and relevant
repository source with OpenAI.
Approve provider data handling and billing before enabling it. The PR must be open
and non-draft; topics, labels, `hold`, and auto-merge settings do not gate review.

The security prompt asks for bugs introduced or materially worsened by the change,
with attacker-controlled input, prerequisites, impact, and practical mitigation
supported by the diff and inspected source. It maps the attacker, execution
identity and authority, protected resource, and unauthorized result for changed
trust boundaries. Relevant checks follow callers, removed guards, failure paths,
queued work, caches, and authorization changes. CI and release changes receive
checks for privileged execution and artifact handoffs; AI integrations receive
checks for tool authority, context isolation, and output handling; availability
checks require an input-to-cost path and shared impact after accounting for
existing bounds. These checks are selected from the changed behavior and its callers.

For a known risk, the reviewer checks what changed in reachability and impact,
which boundary owns mitigation, and whether a documented deferral covers the new
path. Risks within that scope can appear as advisory follow-ups. A new security
consequence, exposure beyond that scope, or a security guarantee broken by the PR
can still warrant a finding.

The reviewer challenges both suspected findings and assumed protections. Missing
tests, a risky-looking API, malicious prompt text, or an absent secondary defense
alone does not establish a vulnerability or increase severity. The overview
describes the inspected security scope and identifies exact unresolved facts and
the evidence needed to resolve them. Material unresolved questions make the
assessment incomplete. Supported P0 through P2 findings still request changes.
The security review excludes speculation and unrelated existing issues.
All review commands use the [shared source lookup](configuration.md#-review-source-access).
Missing patches and unavailable source are disclosed; citations are not verified
and code/tests are not executed. A reply
with no supported findings does not certify security or constitute a complete audit.
Evaluate real examples before [broad use](development.md#-security-review-evaluation).

The independent `secreview` job uses its own bounded queue and releases the webhook
slot before inference. It publishes the same native overview and inline
conversations described for [code reviews](#-ai-code-reviews), and follows their
safeguards for source handling, inference budgets, escaped replies, token usage,
and consent/branch rechecks. It uses the same source-lookup conversation bounded by its deadline,
without response storage, inference retries, fallback, or a review-specific output-token cap. It uses the same
[verdict rules](#-ai-code-reviews) and never labels,
merges, rebases, or executes code. Successful results are identified separately
from summaries and other reviews for each PR head/base pair; these tasks can run
for the same revision. See [security-review recovery](operations.md#-security-review-recovery).

## 🧪 AI test reviews

With [test reviews configured](configuration.md#-test-review-settings), post
`/ab testreview` to assess whether a PR's tests protect changed behavior.
`/ab review` includes it when enabled. A new, unedited human comment from a current
writer or administrator consents to sharing PR metadata, patches, complete changed
files, and relevant repository source, including private source, with the selected
AI provider. The PR must be open and non-draft with both branches in the target
repository; forks are rejected. Topics, labels, `hold`, and auto-merge settings do
not gate review.

The reviewer inspects changed tests and concrete coverage gaps for changed
production behavior. It traces the real caller-facing behavior through the
repository's existing harness, fixtures, helpers, hooks, and assertions. It checks
assertions, fixtures, determinism, and isolation where they affect the
reliability of checks for changed behavior. It searches nearby coverage before
claiming a gap. A blocking test finding needs a supported failure path and a
material consequence that the current tests could miss. A changed assertion that
allows an unauthorized action to pass can qualify. Routine variants of already
covered behavior are left out.

Missing-test findings identify an established test layer that can exercise the
regression or cite a PR requirement for automated coverage at a new layer.
Gaps that require new test infrastructure, such as a browser runner for an
HTTP-only suite, can appear in the overview when they help assess a concrete
remaining risk. Reported manual checks can provide validation evidence.

Correct library registration and direct configuration forwarding do not require
dedicated tests merely because removing that wiring would leave existing tests
green. Integration findings must identify meaningful repository-owned decisions,
transformations, validation, error handling, or compatibility behavior beyond
correct library use. The reviewer must establish a supported failure path before
requesting fault injection and must not demand new production seams solely to
force dependency failures in otherwise correct wiring.

The prompt asks the reviewer to inspect the repository root `.gocov` at the
reviewed head and treat matching code as deliberately excluded from required test
coverage, including exclusions added by the PR. It must not request additional
tests for that code or suggest removing or narrowing exclusions to require tests.
Patterns follow the repository's coverage tooling semantics. Excluded source stays
available as context, and weaknesses in changed existing tests remain in scope.
The overview describes applicable exclusions; they do not by themselves make the
assessment incomplete. This policy applies to test reviews; code and security
reviews retain their scope.

These are static source assessments. Tests and mutation tests are not run,
coverage is not measured, and suspected flakiness is not reproduced. Findings
explain the behavior at risk, how existing tests could miss it, and the smallest
useful correction through the established test boundary. Pure style preferences
and unrelated production bugs or build/CI workflow improvements are excluded.

The independent `testreview` job uses the same [shared review safeguards and verdict
rules](#-ai-code-reviews), including source tools, consent/revision rechecks,
inference budgets, safe native reviews, actual token usage, and outdated-thread
maintenance. Its queue and revision-specific `testreview` result markers remain
independent of other reviews. See [recovery](operations.md#-test-review-recovery)
and [evaluation guidance](development.md#-test-review-evaluation).
