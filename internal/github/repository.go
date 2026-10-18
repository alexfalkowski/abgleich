package github

import (
	"github.com/alexfalkowski/go-service/v2/context"
	"github.com/alexfalkowski/go-service/v2/errors"
	"github.com/alexfalkowski/go-service/v2/net/http"
	"github.com/alexfalkowski/go-service/v2/net/http/media"
	"github.com/alexfalkowski/go-service/v2/strings"
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

// NewRepository creates a repository backed by the injected API client.
// Construction does not read credentials or contact GitHub.
func NewRepository(cfg *Config, client *Client) *Repository {
	return &Repository{config: cfg, client: client}
}

// Repository reads pull request state and requests branch updates through GitHub's
// REST and GraphQL APIs. It owns remote access and response validation; callers own
// rebase policy.
type Repository struct {
	config *Config
	client *Client
}

// IsSameRepository reports whether both pull request branches belong to the
// configured repository. Repository names are compared case-insensitively.
func (r *Repository) IsSameRepository(pull *PullRequest) bool {
	name := strings.ToLower(r.config.Owner + "/" + r.config.Repository)
	headRepository := strings.ToLower(pull.GetHead().GetRepo().GetFullName())
	baseRepository := strings.ToLower(pull.GetBase().GetRepo().GetFullName())

	return headRepository == name && baseRepository == name
}

// ListOpenPullRequests collects all pages before returning. This keeps a caller's
// mutations from changing list ordering while pagination is still in progress.
func (r *Repository) ListOpenPullRequests(ctx context.Context) ([]*PullRequest, error) {
	var pulls []*PullRequest
	opts := &github.PullRequestListOptions{State: "open", PerPage: pageSize}

	for {
		page, response, err := r.client.PullRequests.List(ctx, r.config.Owner, r.config.Repository, opts)
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

// GetPullRequest returns a fresh snapshot, including current auto-merge state
// and branch identities. API errors are returned without retrying.
func (r *Repository) GetPullRequest(ctx context.Context, number int) (*PullRequest, error) {
	pull, _, err := r.client.PullRequests.Get(ctx, r.config.Owner, r.config.Repository, number)
	if err != nil {
		return nil, errors.Prefix("github get pull request", err)
	}

	return pull, nil
}

// Compare reports commits missing from head relative to base. An absent
// comparison count is an error rather than evidence that the branch is current.
func (r *Repository) Compare(ctx context.Context, base, head string) (*CommitsComparison, error) {
	comparison, _, err := r.client.Repositories.CompareCommits(ctx, r.config.Owner, r.config.Repository,
		base, head, &github.ListOptions{PerPage: 1})
	if err != nil {
		return nil, errors.Prefix("github compare commits", err)
	}

	if comparison == nil || comparison.BehindBy == nil {
		return nil, errors.New("github comparison is missing behind_by")
	}

	return comparison, nil
}

// Rebase asks GitHub to rebase a pull request using its expected head SHA.
// Both HTTP failures and GraphQL errors are returned. A successful response
// confirms acceptance; GitHub may finish updating the branch asynchronously.
func (r *Repository) Rebase(ctx context.Context, pull *PullRequest) error {
	nodeID := pull.GetNodeID()
	body := UpdatePullRequestBranchRequest{Query: rebaseMutation}
	body.Variables.Input = UpdatePullRequestBranchInput{
		PullRequestID: nodeID, ExpectedHeadOID: pull.GetHead().GetSHA(), UpdateMethod: "REBASE",
	}

	request, err := r.client.NewRequest(ctx, http.MethodPost, "/graphql", body)
	if err != nil {
		return err
	}

	request.Header.Set(http.AcceptKey, media.JSON)

	var response UpdatePullRequestBranchResponse
	if _, err := r.client.Do(request, &response); err != nil {
		return err
	}

	return response.Validate(nodeID)
}
