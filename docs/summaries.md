# 📝 AI pull request summaries

[← Back to README](../README.md)

With [summaries configured](configuration.md#-summary-settings), `/ab summarise` requests an
advisory summary of an open, non-draft PR with both branches in a repository
accessible to the App. Fork requests are rejected without AI inference.
No repository topic, auto-merge setting, or label is required.

The independent job processes at most four queued requests serially. Inference
uses at most one minute and half the remaining job budget, leaving other commands
available. A full queue returns HTTP 503. Acceptance means queued or already
summarized; the later reply identifies the analyzed head SHA and request.
**An existing App summary prevents another model call or summary comment, even
after restart or new commits.** See [summary recovery](operations.md#-summary-recovery).

> [!IMPORTANT]
> An authorized `/ab summarise` comment authorizes sending PR metadata and
> changed-file patches returned by GitHub to OpenAI, including private
> repositories. Approve the provider's data handling and
> billing before enabling summaries.
> Summaries are advisory and never label, approve, merge, rebase, or execute code.

Context includes untruncated PR titles, descriptions, filenames, and every patch
returned by GitHub, without local file-count or size limits. Files missing patch
text are disclosed to the model; full files and repository trees are not fetched.
Large PRs can increase cost or exceed the context window or job deadline. Source
is untrusted data; the model receives no tools or GitHub credentials.

The prompt requests fewer than 120 words: an outcome, up to three **What changes**
bullets, and up to three **Where to look** bullets explaining what to inspect and
why. File links target the analyzed revision, excluding deleted files and files
without patches. **Missing context** identifies omissions; a collapsed
**Summary details** footer records commit, request, source scope, model, and token
usage, including cached input and cache writes when reported. Missing or
inconsistent usage is shown as unavailable without preventing a summary. These
are provider-reported token counts; the worker does not estimate dollar costs
or request billing access. Output has a 2,048-token budget; length and quality
depend on the model. Malformed sections
remain escaped plain text; per-line citations are neither generated nor validated.
Inspect the diff and evaluate the model on approved PRs before widening use;
summaries cannot establish whole-PR accuracy.

Authorization, PR eligibility, and both branch revisions are rechecked before
submission and publication. Revocation blocks the next observed action; revision
changes discard the explanation and require a new request. Transmitted source
cannot be recalled, and changes after the final read can race publication.
Refusals, failures, incomplete or empty responses, and timeouts produce failure
feedback when still authorized.

The adapter uses the official [OpenAI Go SDK](https://developers.openai.com/api/docs/libraries),
explicit credentials, `store: false`, and no automatic inference retries.
