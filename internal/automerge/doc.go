// Package automerge applies ordered Dependabot, label, and rebase strategies in
// opted-in repositories. Labeled PRs may come from forks; rebasing requires both
// branches in the discovered repository. The internal strategy package owns
// policy and fresh pre-mutation checks; the reconciler owns strategy order and
// repository opt-in. GitHub access and mutations belong to github.
// Module registers Reconcile with the shared scheduler as the automerge job.
// Commands handles verified PR comments by adding the existing opt-in label;
// inbound HTTP handling belongs to transport/http.
package automerge
