# Accepted design

This index records Abgleich's accepted design choices, tradeoffs, and support
boundaries. The area files carry the same weight as [AGENTS.md](../../AGENTS.md).

Read the relevant area files before proposing a change or recording a finding
or audit entry. Revisit a decision when the owner asks or new evidence changes
its documented contract or tradeoff.

- [GitHub automation](accepted-design/automation.md) covers reconciliation,
  repository eligibility, label consent, and guarded mutations.
- [Runtime](accepted-design/runtime.md) covers independent jobs, shutdown,
  and health reporting.
- [AI summaries](accepted-design/summaries.md) covers consent, patch context,
  and one published summary per PR.
- [AI reviews](accepted-design/reviews.md) covers per-feature queues,
  inference, verdicts, review history, and thread maintenance.
- [Review sources](accepted-design/review-sources.md) covers revision-pinned
  source, external dependencies, inspection limits, and caching.
- [Project workflow](accepted-design/project-workflow.md) covers the
  Cucumber harness, checked-in configuration, and model evaluation.
