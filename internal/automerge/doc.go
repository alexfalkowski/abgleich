// Package automerge applies ordered Dependabot, label, and rebase strategies in
// opted-in repositories. Labeled PRs may come from forks; rebasing requires both
// branches in the discovered repository. The internal strategy package owns
// policy and fresh pre-mutation checks; the reconciler owns strategy order and
// repository opt-in. GitHub access and mutations belong to github.
// Coordinator serializes scheduled sweeps and queued repository notifications,
// preserving another pass for events received during a check. Module connects
// its lifecycle and registers its scheduled sweep as the automerge job.
// Config supplies the final CI signal for event-driven rebasing. Scheduled recovery
// preserves its existing rebase policy when that signal never arrives.
// Command handles automerge PR comments by adding the existing opt-in label.
// Reconciler also exposes read-only policy assessment for status reporting.
// Shared dispatch belongs to commands and inbound HTTP handling to transport/http.
package automerge
