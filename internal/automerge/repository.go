package automerge

import "github.com/alexfalkowski/go-service/v2/context"

// GitRepository provides pull request snapshots and branch operations for one
// repository. Reads must return current remote state. Rebase requests a remote
// update guarded by the supplied head SHA; success means the request was
// accepted, and a later read determines whether the branch is current.
type GitRepository interface {
	// ListOpenPullRequests returns all pages of open pull requests before the
	// caller starts mutating branches. Eligibility is decided by the caller.
	// A failed page returns an error without partial results.
	ListOpenPullRequests(ctx context.Context) ([]PullRequest, error)

	// GetPullRequest reads a fresh snapshot by its repository-scoped number,
	// including auto-merge state and head and base branch identities. The caller
	// uses this to recheck eligibility and detect changes before a rebase.
	GetPullRequest(ctx context.Context, number int) (PullRequest, error)

	// Compare reports how many commits from the named base branch are missing
	// from the observed head commit. An unavailable comparison is an error;
	// zero missing commits means the head is current with that base.
	Compare(ctx context.Context, base, head string) (Comparison, error)

	// Rebase requests a rebase of pull.ID, guarded by pull.Head.SHA. A changed
	// head, conflict, or rejected update must return an error. Success confirms
	// acceptance; the caller discovers completion through a later read.
	Rebase(ctx context.Context, pull PullRequest) error
}
