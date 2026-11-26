# AI summaries

Part of the [accepted-design index](../accepted-design.md).

- Keep summaries optional and advisory. Require configured AI summaries, an open
  non-draft PR with both branches in the repository, an unedited human `/ab summarise` comment,
  and current write or admin permission. The authorized comment opts in to AI
  source sharing, independently of auto-merge settings, labels, and repository
  topics. Recheck authorization and both branch revisions before source submission
  and reply publication; discard stale explanations.
- Include every changed file and patch returned by GitHub in summary context;
  do not impose local file-count or patch-size limits. Treat source text as
  untrusted data; generation receives no tools or GitHub credentials and never
  executes code or requests merge automation. Use explicit provider credentials
  and disable response storage and automatic inference retries. Log generation
  outcomes through a decorator without source, response text, or credentials.
- Use the shared in-memory command queue and an independent summary job so
  inference does not hold the webhook command slot. Recheck existing App summaries
  before generation and publication; a published summary prevents another model
  call even after restart or new commits. Pending requests do not survive restart.
