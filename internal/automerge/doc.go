// Package automerge enables auto-merge or merges ready Dependabot PRs and keeps
// eligible branches current in opted-in repositories. Reconciler owns policy;
// github owns remote state and mutations. Module registers Reconcile with the
// shared scheduler as the automerge job.
package automerge
