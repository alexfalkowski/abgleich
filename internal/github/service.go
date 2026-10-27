package github

import (
	"github.com/alexfalkowski/go-service/v2/context"
	"github.com/alexfalkowski/go-service/v2/errors"
	"github.com/alexfalkowski/go-service/v2/net/http"
	"github.com/alexfalkowski/go-service/v2/net/http/media"
	"github.com/alexfalkowski/go-service/v2/strings"
	"github.com/alexfalkowski/go-service/v2/telemetry/logger"
	"github.com/google/go-github/v93/github"
)

const pageSize = 100

// GitHub exposes server-side rebases through GraphQL, while the REST branch
// update only merges. Keep using go-github's authenticated request transport.
// https://docs.github.com/en/graphql/reference/pulls#updatepullrequestbranch
const rebaseMutation = `mutation RebasePullRequest($input: UpdatePullRequestBranchInput!) {
  updatePullRequestBranch(input: $input) {
    pullRequest { id }
  }
}`

// GitHub enables auto-merge through GraphQL and merges once repository
// requirements pass. Keep using go-github's authenticated request transport.
// https://docs.github.com/en/graphql/reference/pulls#enablepullrequestautomerge
const autoMergeMutation = `mutation EnableAutoMerge($input: EnablePullRequestAutoMergeInput!) {
  enablePullRequestAutoMerge(input: $input) {
    pullRequest { id }
  }
}`

// NewService constructs shared worker and webhook access through the App client.
// Construction does not contact GitHub; each sweep discovers fresh remote state.
func NewService(cfg *Config, client *Client, log *logger.Logger) Service {
	return &loggingService{Service: &service{config: cfg, client: client}, logger: log}
}

// Service owns repository discovery, identity checks, REST reads, and guarded
// mutations. Callers own pull request eligibility, rebase, and merge policy.
type Service interface {
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

	// HasWritePermission checks the author's current repository permissions.
	HasWritePermission(ctx context.Context, repo *Repository, author *User) (bool, error)

	// AddLabel adds one label without replacing others or retrying mutations.
	AddLabel(ctx context.Context, repo *Repository, number int, label string) error

	// AddCommentReaction adds a reaction as the App. Repeated reactions are
	// idempotent in GitHub; mutation requests are not retried.
	AddCommentReaction(ctx context.Context, repo *Repository, comment *IssueComment, content string) error

	// AddComment publishes feedback once and validates the response. An ambiguous
	// failure may have published the comment; callers must not retry automatically.
	AddComment(ctx context.Context, repo *Repository, number int, body string) error

	// ListComments collects complete comment history and validates each comment's PR identity.
	ListComments(ctx context.Context, repo *Repository, number int) ([]*IssueComment, error)

	// IsAppComment requires a bot author and the configured GitHub App identity.
	IsAppComment(comment *IssueComment) bool

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

	// IsBaseRepository checks that the PR targets the discovered repository,
	// comparing names case-insensitively. Its head may belong to a fork.
	IsBaseRepository(repo *Repository, pull *PullRequest) bool

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

type service struct {
	config *Config
	client *Client
}

// ListRepositories validates the installation and discovers owned repositories
// it can access, including private repositories.
// Archived, disabled, and foreign repositories are excluded. Every page must
// succeed before any repositories are returned.
func (s *service) ListRepositories(ctx context.Context) ([]*Repository, error) {
	if err := s.ValidateInstallation(ctx); err != nil {
		return nil, err
	}

	opts := &github.ListOptions{PerPage: pageSize}

	var repositories []*Repository
	for {
		page, response, err := s.client.installation.Apps.ListRepos(ctx, opts)
		if err != nil {
			return nil, errors.Prefix("github: list repositories", err)
		}

		if page.GetRepositories() == nil {
			return nil, errors.New("github: installation repositories returned an incomplete page")
		}

		for _, repo := range page.GetRepositories() {
			if !s.isActiveOwnedRepository(repo) {
				continue
			}

			name := repo.GetOwner().GetLogin() + "/" + repo.GetName()
			if strings.IsEmpty(repo.GetName()) || strings.ToLower(repo.GetFullName()) != strings.ToLower(name) {
				return nil, errors.New("github: repository has incomplete or inconsistent identity")
			}

			repositories = append(repositories, repo)
		}

		if response.NextPage == 0 {
			return repositories, nil
		}

		opts.Page = response.NextPage
	}
}

// FindRepository resolves a webhook's repository name through fresh, complete
// installation discovery. Absent or ineligible repositories return nil; failed
// discovery returns an error without a repository.
func (s *service) FindRepository(ctx context.Context, name string) (*Repository, error) {
	repositories, err := s.ListRepositories(ctx)
	if err != nil {
		return nil, err
	}

	for _, repository := range repositories {
		if strings.ToLower(repository.GetFullName()) == strings.ToLower(name) {
			return repository, nil
		}
	}

	return nil, nil
}

// GetRepository reads full repository settings, including AllowAutoMerge and
// AllowSquashMerge, which GitHub omits from repository lists. A mismatched
// repository identity is an error rather than permission to process its PRs.
func (s *service) GetRepository(ctx context.Context, repo *Repository) (*Repository, error) {
	current, _, err := s.client.installation.Repositories.Get(ctx, repo.GetOwner().GetLogin(), repo.GetName())
	if err != nil {
		return nil, errors.Prefix("github: get repository", err)
	}

	sameName := strings.ToLower(current.GetName()) == strings.ToLower(repo.GetName())
	sameOwner := strings.ToLower(current.GetOwner().GetLogin()) == strings.ToLower(repo.GetOwner().GetLogin())
	sameFullName := strings.ToLower(current.GetFullName()) == strings.ToLower(repo.GetFullName())
	if !sameName || !sameOwner || !sameFullName {
		return nil, errors.New("github: repository settings returned no matching repository")
	}

	return current, nil
}

// ListOpenPullRequests returns all open pull requests for repo, oldest first by
// creation time. Every page must succeed before results are returned, so callers
// can finish pagination before mutating branches.
func (s *service) ListOpenPullRequests(ctx context.Context, repo *Repository) ([]*PullRequest, error) {
	var pulls []*PullRequest
	seen := make(map[int]bool)
	opts := &github.PullRequestListOptions{State: "open", Sort: "created", Direction: "asc", PerPage: pageSize}

	for {
		page, response, err := s.client.installation.PullRequests.List(ctx, repo.GetOwner().GetLogin(), repo.GetName(), opts)
		if err != nil {
			return nil, errors.Prefix("github: list pull requests", err)
		}
		if page == nil {
			return nil, errors.New("github: pull requests returned an incomplete page")
		}

		// New pull requests can shift page boundaries while discovery is running.
		for _, pull := range page {
			if seen[pull.GetNumber()] {
				continue
			}

			seen[pull.GetNumber()] = true
			pulls = append(pulls, pull)
		}

		if response.NextPage == 0 {
			break
		}
		opts.Page = response.NextPage
	}

	return pulls, nil
}

// GetPullRequest reads a fresh pull request snapshot by number from repo,
// including its current auto-merge and branch state.
func (s *service) GetPullRequest(ctx context.Context, repo *Repository, number int) (*PullRequest, error) {
	pull, _, err := s.client.installation.PullRequests.Get(ctx, repo.GetOwner().GetLogin(), repo.GetName(), number)
	if err != nil {
		return nil, errors.Prefix("github: get pull request", err)
	}

	return pull, nil
}

// Compare compares head with base in repo and reports missing base commits
// through BehindBy. An absent behind_by field is an error.
func (s *service) Compare(ctx context.Context, repo *Repository, base, head string) (*CommitsComparison, error) {
	comparison, _, err := s.client.installation.Repositories.CompareCommits(ctx, repo.GetOwner().GetLogin(), repo.GetName(),
		base, head, &github.ListOptions{PerPage: 1})
	if err != nil {
		return nil, errors.Prefix("github: compare commits", err)
	}

	if comparison == nil || comparison.BehindBy == nil {
		return nil, errors.New("github: comparison is missing behind_by")
	}

	return comparison, nil
}

// Rebase requests a server-side rebase using pull's node ID and expected head
// SHA. HTTP failures, GraphQL errors, and mismatched response IDs are errors.
// Success confirms acceptance, not completion; the mutation is not retried.
func (s *service) Rebase(ctx context.Context, pull *PullRequest) error {
	nodeID := pull.GetNodeID()
	body := UpdatePullRequestBranchRequest{Query: rebaseMutation}
	body.Variables.Input = UpdatePullRequestBranchInput{
		PullRequestID: nodeID, ExpectedHeadOID: pull.GetHead().GetSHA(), UpdateMethod: "REBASE",
	}

	request, err := s.client.installation.NewRequest(ctx, http.MethodPost, "graphql", body)
	if err != nil {
		return err
	}

	request.Header.Set(http.AcceptKey, media.JSON)

	var response UpdatePullRequestBranchResponse
	if _, err := s.client.installation.Do(request, &response); err != nil {
		return err
	}

	return response.Validate(nodeID)
}

// EnableAutoMerge enables squash auto-merge using the PR's node ID and expected
// head SHA. GitHub enforces repository rules. Failures are not retried.
func (s *service) EnableAutoMerge(ctx context.Context, pull *PullRequest) error {
	body := EnablePullRequestAutoMergeRequest{Query: autoMergeMutation}
	body.Variables.Input = EnablePullRequestAutoMergeInput{
		PullRequestID: pull.GetNodeID(), ExpectedHeadOID: pull.GetHead().GetSHA(), MergeMethod: "SQUASH",
	}

	request, err := s.client.installation.NewRequest(ctx, http.MethodPost, "graphql", body)
	if err != nil {
		return err
	}

	request.Header.Set(http.AcceptKey, media.JSON)

	var response EnablePullRequestAutoMergeResponse
	if _, err := s.client.installation.Do(request, &response); err != nil {
		return err
	}

	return response.Validate(pull.GetNodeID())
}

// Merge requests a squash merge with the expected head SHA. GitHub enforces
// repository rules. HTTP success without a confirmed merge is an error.
func (s *service) Merge(ctx context.Context, repo *Repository, pull *PullRequest) error {
	result, _, err := s.client.installation.PullRequests.Merge(ctx, repo.GetOwner().GetLogin(), repo.GetName(), pull.GetNumber(),
		strings.Empty, &github.PullRequestOptions{SHA: pull.GetHead().GetSHA(), MergeMethod: "squash"})
	if err != nil {
		return errors.Prefix("github: merge", err)
	}

	if result == nil || !result.GetMerged() {
		return errors.New("github: merge returned no confirmed merge")
	}

	return nil
}

func (s *service) isActiveOwnedRepository(repo *Repository) bool {
	return !repo.GetArchived() && !repo.GetDisabled() &&
		strings.ToLower(repo.GetOwner().GetLogin()) == strings.ToLower(s.config.Owner)
}
