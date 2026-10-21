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

// NewService constructs owner-wide GitHub access using one shared client.
// Construction does not contact GitHub; each sweep discovers fresh remote state.
func NewService(cfg *Config, client *Client, log *logger.Logger) Service {
	return &loggingService{Service: &service{config: cfg, client: client}, logger: log}
}

// Service owns repository discovery, identity checks, REST reads, and guarded
// mutations. Callers own pull request eligibility, rebase, and merge policy.
type Service interface {
	// ListRepositories validates the authenticated owner and collects all pages
	// before returning active owned repositories visible to the token.
	ListRepositories(ctx context.Context) ([]*Repository, error)

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

	// ListOpenPullRequests collects all pages before returning so mutations
	// cannot change ordering partway through pagination.
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

// ListRepositories validates the authenticated personal account and discovers
// all owned repositories visible to its token, including private repositories.
// Archived, disabled, and foreign repositories are excluded. Every page must
// succeed before any repositories are returned.
func (s *service) ListRepositories(ctx context.Context) ([]*Repository, error) {
	user, _, err := s.client.Users.Get(ctx, strings.Empty)
	if err != nil {
		return nil, errors.Prefix("github authenticated owner", err)
	}

	if strings.ToLower(user.GetLogin()) != strings.ToLower(s.config.Owner) {
		return nil, errors.New("github authenticated account does not match configured owner")
	}

	opts := &github.RepositoryListByAuthenticatedUserOptions{
		Affiliation: "owner", Visibility: "all", Sort: "full_name", Direction: "asc",
		PerPage: pageSize,
	}

	var repositories []*Repository
	for {
		page, response, err := s.client.Repositories.ListByAuthenticatedUser(ctx, opts)
		if err != nil {
			return nil, errors.Prefix("github list repositories", err)
		}

		for _, repo := range page {
			if !s.isActiveOwnedRepository(repo) {
				continue
			}

			name := repo.GetOwner().GetLogin() + "/" + repo.GetName()
			if strings.IsEmpty(repo.GetName()) || strings.ToLower(repo.GetFullName()) != strings.ToLower(name) {
				return nil, errors.New("github repository has incomplete or inconsistent identity")
			}

			repositories = append(repositories, repo)
		}

		if response.NextPage == 0 {
			return repositories, nil
		}

		opts.Page = response.NextPage
	}
}

// GetRepository reads full repository settings, including AllowAutoMerge and
// AllowSquashMerge, which GitHub omits from repository lists. A mismatched
// repository identity is an error rather than permission to process its PRs.
func (s *service) GetRepository(ctx context.Context, repo *Repository) (*Repository, error) {
	current, _, err := s.client.Repositories.Get(ctx, repo.GetOwner().GetLogin(), repo.GetName())
	if err != nil {
		return nil, errors.Prefix("github get repository", err)
	}

	sameName := strings.ToLower(current.GetName()) == strings.ToLower(repo.GetName())
	sameOwner := strings.ToLower(current.GetOwner().GetLogin()) == strings.ToLower(repo.GetOwner().GetLogin())
	sameFullName := strings.ToLower(current.GetFullName()) == strings.ToLower(repo.GetFullName())
	if !sameName || !sameOwner || !sameFullName {
		return nil, errors.New("github repository settings returned no matching repository")
	}

	return current, nil
}

// ListOpenPullRequests returns all open pull requests for repo. Every page must
// succeed before results are returned, so callers can finish pagination before
// mutating branches.
func (s *service) ListOpenPullRequests(ctx context.Context, repo *Repository) ([]*PullRequest, error) {
	var pulls []*PullRequest
	opts := &github.PullRequestListOptions{State: "open", PerPage: pageSize}

	for {
		page, response, err := s.client.PullRequests.List(ctx, repo.GetOwner().GetLogin(), repo.GetName(), opts)
		if err != nil {
			return nil, errors.Prefix("github list pull requests", err)
		}

		pulls = append(pulls, page...)

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
	pull, _, err := s.client.PullRequests.Get(ctx, repo.GetOwner().GetLogin(), repo.GetName(), number)
	if err != nil {
		return nil, errors.Prefix("github get pull request", err)
	}

	return pull, nil
}

// Compare compares head with base in repo and reports missing base commits
// through BehindBy. An absent behind_by field is an error.
func (s *service) Compare(ctx context.Context, repo *Repository, base, head string) (*CommitsComparison, error) {
	comparison, _, err := s.client.Repositories.CompareCommits(ctx, repo.GetOwner().GetLogin(), repo.GetName(),
		base, head, &github.ListOptions{PerPage: 1})
	if err != nil {
		return nil, errors.Prefix("github compare commits", err)
	}

	if comparison == nil || comparison.BehindBy == nil {
		return nil, errors.New("github comparison is missing behind_by")
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

	request, err := s.client.NewRequest(ctx, http.MethodPost, "/graphql", body)
	if err != nil {
		return err
	}

	request.Header.Set(http.AcceptKey, media.JSON)

	var response UpdatePullRequestBranchResponse
	if _, err := s.client.Do(request, &response); err != nil {
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

	request, err := s.client.NewRequest(ctx, http.MethodPost, "/graphql", body)
	if err != nil {
		return err
	}

	request.Header.Set(http.AcceptKey, media.JSON)

	var response EnablePullRequestAutoMergeResponse
	if _, err := s.client.Do(request, &response); err != nil {
		return err
	}

	return response.Validate(pull.GetNodeID())
}

// Merge requests a squash merge with the expected head SHA. GitHub enforces
// repository rules. HTTP success without a confirmed merge is an error.
func (s *service) Merge(ctx context.Context, repo *Repository, pull *PullRequest) error {
	result, _, err := s.client.PullRequests.Merge(ctx, repo.GetOwner().GetLogin(), repo.GetName(), pull.GetNumber(),
		strings.Empty, &github.PullRequestOptions{SHA: pull.GetHead().GetSHA(), MergeMethod: "squash"})
	if err != nil {
		return errors.Prefix("github merge", err)
	}

	if result == nil || !result.GetMerged() {
		return errors.New("github merge returned no confirmed merge")
	}

	return nil
}

func (s *service) isActiveOwnedRepository(repo *Repository) bool {
	return !repo.GetArchived() && !repo.GetDisabled() &&
		strings.ToLower(repo.GetOwner().GetLogin()) == strings.ToLower(s.config.Owner)
}
