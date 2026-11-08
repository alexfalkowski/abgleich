package github

import (
	"github.com/alexfalkowski/go-service/v2/cache/config"
	"github.com/alexfalkowski/go-service/v2/context"
	"github.com/alexfalkowski/go-service/v2/telemetry/logger"
)

const pageSize = 100

// Service owns repository discovery, identity checks, REST reads, and guarded
// mutations. Callers own pull request eligibility, rebase, and merge policy.
type Service interface {
	// ReadSourceFiles batches uncached regular entries from ListSourceFiles at a fixed
	// revision. Only complete UTF-8 blobs matching the repository, tree SHA and
	// byte size are returned, keyed by path. Binary entries are omitted; failed,
	// missing, or inconsistent responses reject the group. Validated blobs are
	// cached by repository identity and blob SHA when ttlcache is configured.
	// Reads do not retry.
	ReadSourceFiles(ctx context.Context, repo *Repository, revision string, files []*TreeEntry) (map[string]string, error)

	// ListSourceFiles returns all entries from a complete recursive tree at an
	// immutable commit SHA, retaining directories, symlinks, and submodules as
	// metadata without following them. ReadSourceFiles accepts only entries passing
	// IsRegularSourceFile. Truncated trees, invalid or duplicate paths, and missing
	// regular-file blob identities are errors. Validated trees are cached by
	// repository identity and revision when ttlcache is configured.
	ListSourceFiles(ctx context.Context, repo *Repository, revision string) ([]*TreeEntry, error)

	// ListBranches collects complete, validated branch pagination before cleanup.
	ListBranches(ctx context.Context, repo *Repository) ([]*Branch, error)

	// GetBranch rechecks branch identity, protection, and head without redirects.
	GetBranch(ctx context.Context, repo *Repository, name string) (*Branch, error)

	// GetCommit reads an immutable commit and requires a matching SHA and date.
	GetCommit(ctx context.Context, repo *Repository, sha string) (*RepositoryCommit, error)

	// ListBranchPullRequests collects complete head-branch history, including open PRs.
	ListBranchPullRequests(ctx context.Context, repo *Repository, branch string) ([]*PullRequest, error)

	// DeleteBranch deletes a branch only while its expected SHA still matches.
	// GraphQL failures and unconfirmed responses fail without mutation retries.
	DeleteBranch(ctx context.Context, repo *Repository, branch *Branch) error

	// ValidateInstallation checks the configured App, owner, and suspension state
	// without caching. Comment commands must call it before accessing repositories.
	ValidateInstallation(ctx context.Context) error

	// GetComment reads the current comment and verifies its PR and author identity.
	GetComment(ctx context.Context, repo *Repository, number int, expected *IssueComment) (*IssueComment, error)

	// GetCommentWithHistory reads the complete discussion and validates the
	// expected command's PR, author, and creation identity. A missing command
	// returns nil with the current history; no snapshot is cached.
	GetCommentWithHistory(ctx context.Context, repo *Repository, number int, expected *IssueComment) (*IssueComment, []*IssueComment, error)

	// HasWritePermission checks the author's current repository permissions.
	HasWritePermission(ctx context.Context, repo *Repository, author *User) (bool, error)

	// AddLabel adds one label without replacing others or retrying mutations.
	AddLabel(ctx context.Context, repo *Repository, number int, label string) error

	// AddCommentReaction adds a reaction as the App. Repeated reactions are
	// idempotent in GitHub; mutation requests are not retried.
	AddCommentReaction(ctx context.Context, repo *Repository, comment *IssueComment, content Reaction) error

	// AddComment publishes feedback once and validates the response. An ambiguous
	// failure may have published the comment; callers must not retry automatically.
	AddComment(ctx context.Context, repo *Repository, number int, body string) error

	// ListComments collects complete comment history and validates each comment's PR identity.
	ListComments(ctx context.Context, repo *Repository, number int) ([]*IssueComment, error)

	// IsAppComment requires a bot author and the configured GitHub App identity.
	IsAppComment(comment *IssueComment) bool

	// AddReview submits a COMMENT, APPROVE, or REQUEST_CHANGES review at the reviewed head and validates
	// publication and authenticated authorship. Ambiguous failures are not retried.
	AddReview(ctx context.Context, repo *Repository, pull *PullRequest, review *PullRequestReviewRequest) error

	// ListReviews returns complete native review history with validated PR identities.
	ListReviews(ctx context.Context, repo *Repository, pull *PullRequest) ([]*PullRequestReview, error)

	// HasAppReview checks whether any candidate belongs to the authenticated
	// installation viewer. Human authors are skipped; bot checks are batched.
	HasAppReview(ctx context.Context, reviews []*PullRequestReview) (bool, error)

	// AppReviews returns authenticated installation-viewer reviews in input order.
	// Human authors are skipped; bot checks are batched and fail on incomplete results.
	AppReviews(ctx context.Context, reviews []*PullRequestReview) ([]*PullRequestReview, error)

	// ListReviewThreads completes thread pagination with authenticated root and review ownership.
	ListReviewThreads(ctx context.Context, repo *Repository, pull *PullRequest) ([]*ReviewThread, error)

	// GetReviewThread refreshes one thread and returns nil for a closed parent PR.
	GetReviewThread(ctx context.Context, repo *Repository, pull *PullRequest, id string) (*ReviewThread, error)

	// ResolveReviewThread resolves once; callers refresh eligibility immediately before it.
	ResolveReviewThread(ctx context.Context, repo *Repository, pull *PullRequest, thread *ReviewThread) error

	// ListPullRequestFiles rejects incomplete pagination and mismatched file counts.
	ListPullRequestFiles(ctx context.Context, repo *Repository, pull *PullRequest) ([]*CommitFile, error)

	// ListRepositories validates the authenticated owner and collects all pages
	// before returning active owned repositories accessible to the installation.
	ListRepositories(ctx context.Context) ([]*Repository, error)

	// FindRepository completes installation discovery and returns the active
	// owned repository with this full name, ignoring case, or nil when absent.
	FindRepository(ctx context.Context, name string) (*Repository, error)

	// GetRepository reads full settings omitted by discovery and validates that
	// the response identifies the discovered repository.
	GetRepository(ctx context.Context, repo *Repository) (*Repository, error)

	// IsSameRepository checks that both PR branches belong to the discovered
	// repository, comparing names case-insensitively.
	IsSameRepository(repo *Repository, pull *PullRequest) bool

	// IsRepository compares a repository's full owner/name with name, ignoring
	// case. It performs no remote access or partial-name matching.
	IsRepository(repo *Repository, name string) bool

	// IsBaseRepository checks that the PR targets the discovered repository,
	// comparing names case-insensitively. Its head may belong to a fork.
	IsBaseRepository(repo *Repository, pull *PullRequest) bool

	// IsSamePullRequestIdentity matches PR number and node ID across snapshots
	// and requires current to target repo, comparing repository names ignoring
	// case. Branch refs and commit SHAs may differ.
	IsSamePullRequestIdentity(repo *Repository, current, expected *PullRequest) bool

	// IsSamePullRequest compares identity, branch repositories, refs, and head
	// SHA across snapshots. Repository names are compared case-insensitively.
	IsSamePullRequest(current, expected *PullRequest) bool

	// ListOpenPullRequests collects all pages in ascending creation order before
	// returning so mutations cannot change ordering partway through pagination.
	ListOpenPullRequests(ctx context.Context, repo *Repository) ([]*PullRequest, error)

	// GetPullRequest reads fresh auto-merge, branch, and identity state.
	GetPullRequest(ctx context.Context, repo *Repository, number int) (*PullRequest, error)

	// ListPullRequestAutomationEvents collects all label and auto-merge-disable
	// events through GraphQL. Event names are GraphQL type names. Errors,
	// incomplete pages, and mismatched PR identities reject the whole history.
	ListPullRequestAutomationEvents(ctx context.Context, pull *PullRequest) ([]*IssueEvent, error)

	// Compare reports commits missing from head relative to base. A missing
	// behind_by is an error rather than evidence that the branch is current.
	Compare(ctx context.Context, repo *Repository, base, head string) (*CommitsComparison, error)

	// ListCheckRuns collects all latest check runs and validates their head SHA.
	ListCheckRuns(ctx context.Context, repo *Repository, sha string) ([]*CheckRun, error)

	// ListCommitStatuses collects the latest status for each context, validating
	// the commit SHA and complete pagination of the combined-status response.
	ListCommitStatuses(ctx context.Context, repo *Repository, sha string) ([]*CommitStatus, error)

	// Rebase requests a server-side rebase guarded by the expected head SHA.
	// HTTP failures, GraphQL errors, and mismatched responses fail without
	// retrying the mutation. Success confirms acceptance, not completion.
	Rebase(ctx context.Context, pull *PullRequest) error

	// EnableAutoMerge requests squash merging once GitHub's requirements pass,
	// guarded by the expected head SHA. Failures are not retried.
	EnableAutoMerge(ctx context.Context, pull *PullRequest) error

	// Merge squash-merges a ready pull request, guarded by its head SHA.
	// A rejected or unconfirmed merge fails without retrying.
	Merge(ctx context.Context, repo *Repository, pull *PullRequest) error
}

// NewService constructs shared worker and webhook access through the App client.
// Construction does not contact GitHub; each sweep discovers fresh remote state.
// Shared source retention uses ttlcache only; other backends are bypassed.
func NewService(cfg *Config, cacheCfg *config.Config, client *Client, log *logger.Logger) Service {
	var remote Service = &service{config: cfg, client: client}
	if cacheCfg.IsEnabled() && cacheCfg.Kind == cacheKind {
		remote = &cachingService{Service: remote}
	}
	return &loggingService{Service: remote, logger: log}
}

type service struct {
	config *Config
	client *Client
}
