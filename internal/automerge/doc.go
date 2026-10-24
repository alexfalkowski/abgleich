// Package automerge applies ordered Dependabot, label, and rebase strategies in
// opted-in repositories. Labeled PRs may come from forks; rebasing requires both
// branches in the discovered repository. The internal strategy package owns
// policy and fresh pre-mutation checks; the reconciler owns strategy order and
// repository opt-in. GitHub access and mutations belong to github.
// Module registers Reconcile with the shared scheduler as the automerge job.
// Command handles automerge PR comments by adding the existing opt-in label.
// Reconciler also exposes read-only policy assessment for status reporting.
// Shared dispatch belongs to commands and inbound HTTP handling to transport/http.
package automerge
