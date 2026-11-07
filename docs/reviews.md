# 🔍 AI reviews

[← Back to README](../README.md)

## 🔍 Request all configured reviews

Post `/ab review` to request every enabled review, currently code and security
reviews. It uses the existing [review configuration](configuration.md#-code-review-settings)
and needs no additional setting or job. Omitted review features are skipped;
with no reviews configured, the command receives rejection feedback.

The original unedited human comment authorizes source sharing for every enabled
review. Each workflow independently rechecks current write/admin permission,
PR eligibility, consent, and head/base revisions before source access and publication,
using the safeguards below. Review requests remain available with `hold` and
without auto-merge opt-in; forks are rejected.

Each feature retains its own model, queue, schedule, verdict, and native review.
The command attempts every configured review even if another declines or fails.
Acceptance feedback requires every review to be queued or already complete.
Partial acceptance retains queued work; a full queue returns HTTP 503. Reissue
`/ab review` after recovery to request the remaining reviews. Pending and
successful results are reused independently, including results requested through
`/ab codereview` or `/ab secreview`. Individual commands also reuse results
from `/ab review`. [Restart and failure recovery](operations.md#-code-review-recovery)
still applies per feature. Summaries are requested separately with `/ab summarise`.

## 🔬 AI code reviews

With [code reviews configured](configuration.md#-code-review-settings), post `/ab codereview` to
request a review of an open, non-draft PR with both branches in the target
repository. Fork requests are rejected without AI inference.
A new, unedited human comment from a current writer or administrator authorizes
sharing PR metadata, patches, whole changed-file text, and relevant repository
source with OpenAI, including private source. No topic, auto-merge setting, label, or removal of `hold` is needed.
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
Both reviews can also inspect public upstream files in other GitHub repositories,
including chart templates and documentation, without additional configuration.
Those reads use no credentials and disclose the resolved commit and inspected
paths; see [review source access](configuration.md#-review-source-access).
These are static source reviews: running tests, rendering Helm manifests,
deploying, and inspecting live services are outside their scope. Lack of execution
alone does not make a review unfinished; material unresolved source gaps still do.

The App submits one native review with an overview and a separate inline
conversation for each finding that can be placed on the supplied diff. Findings
without a reliable diff location remain in the overview with their claimed
location, when supplied. Each finding has a short 🐛 title with explicit severity
(`P0` critical, `P1` high, `P2` medium, or `P3` low), followed by separate
🔎 **Trigger**, 💥 **Impact**, and 🛠️ **Suggested fix** paragraphs. The worker
renders these labels and escapes each model-provided field. Security-review
triggers include attacker-controlled input and prerequisites. A review with no
supported findings still publishes its overview. The worker selects the verdict
from the structured result:

| Assessment | GitHub verdict |
| --- | --- |
| One or more supported findings, including overview-only findings | `REQUEST_CHANGES` |
| No findings, explicit model completion, and no worker-recorded missing text patches or unavailable source | `APPROVE` |
| No findings with incomplete investigation or missing context | `COMMENT`; approval withheld |

The overview shows the worker's assessment and decision separately from model
prose, with ✅ for approval, ❗ for requested changes, and ⚠️ for comments.
With no findings, **Review unfinished** means the model explicitly reports that
it did not finish. **Review completed with limitations** means the model reports
completion but missing patches or unavailable source limit the assessment. A
**Limitations** list identifies each missing patch and unavailable source. Both
outcomes withhold approval, carry no successful-result marker, and allow a new
manual request on the same revisions. Findings still request changes when other
source is unavailable. Model completion is a self-report, not proof of coverage.
Confirmed binary files are outside the text review scope. **Excluded binary files**
lists them separately; their omitted contents and patches do not prevent approval.
Every body contains escaped model text, with links and mentions inactive.
Review text is published without credential redaction.
Details identify the request, head/base SHAs, model, and actual token
usage, or unavailable usage; no dollar costs are estimated. Missing patches are
disclosed, along with the source actually inspected and any unavailable reads.
Citations are not verified, and code/tests are not executed. Findings may be incorrect
or incomplete; an approval records no supported findings within the inspected scope.
Reviews never label, merge, or rebase. Open inline
conversations can block merging when the repository requires conversation
resolution. A review targets the analyzed head; a push after the final check can
still make it outdated. Evaluate approved data before
[broad use](development.md#-code-review-evaluation).

Code and security reviews share one GitHub reviewer identity. Final verdict checks
and publication are serialized within the worker process. A clean review comments
instead of approving while the other feature's latest decisive App assessment
still requests changes, including across new revisions and restarts. A later
complete clean assessment or explicit human dismissal clears that feature's
findings; incomplete comments do not. Both commands keep independent inference
and queues. This does not require both reviews to run before the first approval.

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
A matching successful App result prevents another model call across restarts.
New head or base revisions permit a fresh review; summaries and security reviews
remain independent.
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
supported by the diff and inspected source. It excludes speculation and unrelated
existing issues. Both review commands use the [shared source lookup](configuration.md#-review-source-access).
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
from summaries and code reviews for each PR head/base pair; all three can run for
the same revision. See [security-review recovery](operations.md#-security-review-recovery).
