package github

import (
	"github.com/alexfalkowski/go-service/v2/context"
	"github.com/alexfalkowski/go-service/v2/errors"
	"github.com/alexfalkowski/go-service/v2/strings"
)

// ReviewThread contains the state and authenticated ownership needed to maintain
// an existing conversation. IsAppAuthor requires both its root and parent review.
type ReviewThread struct {
	// ID is the GraphQL node identity used for final rechecks and resolution.
	ID string

	// ReviewBody contains the parent review's result marker for feature ownership checks.
	ReviewBody string

	// IsOutdated reports stale placement from GitHub, without establishing that a finding was fixed.
	IsOutdated bool

	// IsResolved reports whether the conversation is already closed.
	IsResolved bool

	// ViewerCanResolve reports whether the authenticated installation may resolve the thread.
	ViewerCanResolve bool

	// IsAppAuthor requires API-authenticated Bot authorship of both the root comment and parent review.
	IsAppAuthor bool
}

// reviewThreadFields keeps discovery and final rechecks on the same evidence.
// Only the root comment is needed: viewerDidAuthor on it and its parent review
// authenticates App ownership, while the review body identifies the feature's
// result namespace. Replies do not affect maintenance eligibility.
const reviewThreadFields = `
 id isOutdated isResolved viewerCanResolve
 pullRequest { id number state repository { nameWithOwner } }
 comments(first: 1) {
   nodes {
     id viewerDidAuthor author { __typename }
     pullRequestReview {
       id body viewerDidAuthor author { __typename }
       pullRequest { id number state repository { nameWithOwner } }
     }
   }
 }`

// reviewThreadsQuery pages through a PR's inline conversations by PR node ID.
// Parent identities prevent accepting threads from another PR or repository;
// pageInfo lets the caller finish discovery before resolving any thread.
const reviewThreadsQuery = `query PullRequestReviewThreads($id: ID!, $after: String) {
 node(id: $id) {
   ... on PullRequest {
     id number state repository { nameWithOwner }
     reviewThreads(first: 100, after: $after) {
       nodes { ` + reviewThreadFields + ` }
       pageInfo { endCursor hasNextPage }
     }
   }
 }
}`

// reviewThreadQuery refreshes one candidate by thread node ID before resolution.
// The shared fields recheck ownership, outdated/resolved state, permission,
// and whether the parent PR is still open.
const reviewThreadQuery = `query PullRequestReviewThread($id: ID!) {
 node(id: $id) { ... on PullRequestReviewThread { ` + reviewThreadFields + ` } }
}`

// resolveReviewThreadMutation returns the thread ID and resolved state so the
// service can confirm acceptance. GitHub has no expected-state guard; callers
// refresh eligibility beforehand, but a change after that read can still race.
// https://docs.github.com/en/graphql/reference/pulls#resolvereviewthread
const resolveReviewThreadMutation = `mutation ResolveReviewThread($input: ResolveReviewThreadInput!) {
 resolveReviewThread(input: $input) { thread { id isResolved } }
}`

// ListReviewThreads reads complete pagination and only the root of each thread.
func (s *service) ListReviewThreads(ctx context.Context, repo *Repository, pull *PullRequest) ([]*ReviewThread, error) {
	body := ReviewThreadsRequest{Query: reviewThreadsQuery}
	body.Variables.ID = pull.GetNodeID()

	var threads []*ReviewThread
	for {
		var response ReviewThreadsResponse
		if err := s.graphql(ctx, body, &response); err != nil {
			return nil, err
		}
		if err := response.validate(repo, pull); err != nil {
			return nil, err
		}

		page := response.Data.Node.ReviewThreads
		for _, wire := range page.Nodes {
			thread, err := wire.validate(repo, pull)
			if err != nil {
				return nil, err
			}
			threads = append(threads, thread)
		}
		if !*page.PageInfo.HasNextPage {
			return threads, nil
		}

		cursor := page.PageInfo.EndCursor

		if body.Variables.After != nil && *cursor == *body.Variables.After {
			return nil, errors.New("github: review threads returned no advancing cursor")
		}
		body.Variables.After = cursor
	}
}

// GetReviewThread refreshes state and authenticated ownership for one thread.
// A closed parent PR returns nil so callers leave its conversations unchanged.
func (s *service) GetReviewThread(ctx context.Context, repo *Repository, pull *PullRequest, id string) (*ReviewThread, error) {
	body := ReviewThreadRequest{Query: reviewThreadQuery}
	body.Variables.ID = id

	var response ReviewThreadResponse
	if err := s.graphql(ctx, body, &response); err != nil {
		return nil, err
	}
	if err := response.Errors.Validate(); err != nil {
		return nil, err
	}

	wire := response.Data.Node
	thread, err := wire.validate(repo, pull)
	if err != nil {
		return nil, err
	}
	if thread.ID != id {
		return nil, errors.New("github: review thread returned a different identity")
	}

	if wire.PullRequest.State != "OPEN" {
		return nil, nil
	}

	return thread, nil
}

// ResolveReviewThread confirms matching resolved state without retrying failures.
// Callers own eligibility and must refresh the PR and thread before mutation.
func (s *service) ResolveReviewThread(ctx context.Context, _ *Repository, _ *PullRequest, thread *ReviewThread) error {
	body := ResolveReviewThreadRequest{Query: resolveReviewThreadMutation}
	body.Variables.Input.ThreadID = thread.ID

	var response ResolveReviewThreadResponse
	if err := s.graphql(ctx, body, &response); err != nil {
		return err
	}
	if err := response.Errors.Validate(); err != nil {
		return err
	}

	resolved := response.Data.Resolve.Thread
	sameThread := resolved != nil && resolved.ID == thread.ID
	isResolved := resolved != nil && resolved.IsResolved != nil && *resolved.IsResolved

	if !sameThread || !isResolved {
		return errors.New("github: resolve review thread returned no matching resolved thread")
	}

	return nil
}

// ReviewThreadsRequest queries one page of threads by pull request node ID.
// A nil After cursor starts pagination.
type ReviewThreadsRequest struct {
	// Variables binds the operation's GraphQL arguments.
	Variables struct {
		// ID identifies the PR node whose inline conversations are requested.
		ID string `json:"id"`

		// After selects the next page; nil starts pagination.
		After *string `json:"after"`
	} `json:"variables"`

	// Query contains the GraphQL operation submitted with these variables.
	Query string `json:"query"`
}

// ReviewThreadsResponse carries thread ownership, state, and pagination.
// HTTP success still requires GraphQL, identity, and completeness validation.
type ReviewThreadsResponse struct {
	// Data contains returned operation data, which still requires error and identity validation.
	Data struct {
		// Node contains the requested PR and thread page; nil cannot establish identity.
		Node *struct {
			threadPullRequest

			// ReviewThreads contains one page of thread state and ownership evidence.
			ReviewThreads struct {
				// Nodes contains thread candidates; nil indicates an incomplete discovery page.
				Nodes []*reviewThreadNode `json:"nodes"`

				// PageInfo reports whether discovery must continue before the results are complete.
				PageInfo struct {
					// EndCursor selects the following page when HasNextPage is true.
					EndCursor *string `json:"endCursor"`

					// HasNextPage reports whether more results remain; nil indicates an incomplete response.
					HasNextPage *bool `json:"hasNextPage"`
				} `json:"pageInfo"`
			} `json:"reviewThreads"`
		} `json:"node"`
	} `json:"data"`

	// Errors retains GraphQL failures even when the HTTP response succeeds.
	Errors Errors `json:"errors"`
}

// ReviewThreadRequest queries one review thread by its node ID.
type ReviewThreadRequest struct {
	// Query contains the GraphQL operation submitted with these variables.
	Query string `json:"query"`

	// Variables binds the operation's GraphQL arguments.
	Variables struct {
		// ID identifies the thread node to refresh immediately before maintenance.
		ID string `json:"id"`
	} `json:"variables"`
}

// ReviewThreadResponse carries one thread's state and authenticated ownership.
// The service validates its identity and parent PR before returning a snapshot.
type ReviewThreadResponse struct {
	// Data contains returned operation data, which still requires error and identity validation.
	Data struct {
		// Node contains the thread's current state and ownership evidence; nil fails validation.
		Node *reviewThreadNode `json:"node"`
	} `json:"data"`

	// Errors retains GraphQL failures even when the HTTP response succeeds.
	Errors Errors `json:"errors"`
}

// ResolveReviewThreadRequest resolves a thread by node ID without an
// expected-state guard; callers must refresh eligibility before submission.
type ResolveReviewThreadRequest struct {
	// Query contains the GraphQL operation submitted with these variables.
	Query string `json:"query"`

	// Variables binds the operation's GraphQL arguments.
	Variables struct {
		// Input supplies the mutation arguments, including the target identity.
		Input struct {
			// ThreadID identifies the thread to resolve after final eligibility rechecks.
			ThreadID string `json:"threadId"`
		} `json:"input"`
	} `json:"variables"`
}

// ResolveReviewThreadResponse confirms a matching thread's resolved state.
// GraphQL errors or missing confirmation must not be treated as acceptance.
type ResolveReviewThreadResponse struct {
	// Data contains returned operation data, which still requires error and identity validation.
	Data struct {
		// Resolve contains the resolveReviewThread acceptance payload.
		Resolve struct {
			// Thread confirms the target thread and its resolved state; nil cannot confirm acceptance.
			Thread *struct {
				// ID must match the requested thread node ID to confirm acceptance.
				ID string `json:"id"`

				// IsResolved must be present and true to confirm that the mutation resolved the thread.
				IsResolved *bool `json:"isResolved"`
			} `json:"thread"`
		} `json:"resolveReviewThread"`
	} `json:"data"`

	// Errors retains GraphQL failures even when the HTTP response succeeds.
	Errors Errors `json:"errors"`
}

func (r *ReviewThreadsResponse) validate(repo *Repository, pull *PullRequest) error {
	if err := r.Errors.Validate(); err != nil {
		return err
	}

	node := r.Data.Node
	if node == nil || !node.matches(repo, pull) {
		return errors.New("github: review threads returned no matching pull request")
	}

	page := node.ReviewThreads
	if page.Nodes == nil || page.PageInfo.HasNextPage == nil {
		return errors.New("github: review threads returned an incomplete page")
	}
	if *page.PageInfo.HasNextPage {
		cursor := page.PageInfo.EndCursor
		if cursor == nil || strings.IsEmpty(*cursor) {
			return errors.New("github: review threads returned no next cursor")
		}
	}

	return nil
}

type threadPullRequest struct {
	ID         string `json:"id"`
	State      string `json:"state"`
	Repository struct {
		NameWithOwner string `json:"nameWithOwner"`
	} `json:"repository"`
	Number int `json:"number"`
}

func (p *threadPullRequest) matches(repo *Repository, pull *PullRequest) bool {
	sameNodeID := p.ID == pull.GetNodeID()
	sameNumber := p.Number == pull.GetNumber()
	sameRepository := strings.ToLower(p.Repository.NameWithOwner) == strings.ToLower(repo.GetFullName())
	validState := p.State == "OPEN" || p.State == "CLOSED" || p.State == "MERGED"

	return sameNodeID && sameNumber && sameRepository && validState
}

type reviewThreadNode struct {
	ID               string            `json:"id"`
	IsOutdated       *bool             `json:"isOutdated"`
	IsResolved       *bool             `json:"isResolved"`
	ViewerCanResolve *bool             `json:"viewerCanResolve"`
	PullRequest      threadPullRequest `json:"pullRequest"`
	Comments         struct {
		Nodes []*struct {
			ID              string `json:"id"`
			ViewerDidAuthor *bool  `json:"viewerDidAuthor"`
			Author          struct {
				Type string `json:"__typename"`
			} `json:"author"`
			Review *struct {
				ID              string  `json:"id"`
				Body            *string `json:"body"`
				ViewerDidAuthor *bool   `json:"viewerDidAuthor"`
				Author          struct {
					Type string `json:"__typename"`
				} `json:"author"`
				PullRequest threadPullRequest `json:"pullRequest"`
			} `json:"pullRequestReview"`
		} `json:"nodes"`
	} `json:"comments"`
}

func (n *reviewThreadNode) validate(repo *Repository, pull *PullRequest) (*ReviewThread, error) {
	if n == nil || strings.IsEmpty(n.ID) || n.IsOutdated == nil || n.IsResolved == nil || n.ViewerCanResolve == nil || !n.PullRequest.matches(repo, pull) {
		return nil, errors.New("github: review thread returned incomplete or mismatched state")
	}
	if err := n.validateOwnership(repo, pull); err != nil {
		return nil, err
	}
	return n.snapshot(), nil
}

func (n *reviewThreadNode) snapshot() *ReviewThread {
	root := n.Comments.Nodes[0]
	review := root.Review
	return &ReviewThread{
		ID: n.ID, ReviewBody: *review.Body, IsOutdated: *n.IsOutdated, IsResolved: *n.IsResolved, ViewerCanResolve: *n.ViewerCanResolve,
		IsAppAuthor: *root.ViewerDidAuthor && *review.ViewerDidAuthor && root.Author.Type == "Bot" && review.Author.Type == "Bot",
	}
}

func (n *reviewThreadNode) validateOwnership(repo *Repository, pull *PullRequest) error {
	if len(n.Comments.Nodes) != 1 || n.Comments.Nodes[0] == nil {
		return errors.New("github: review thread returned no root comment")
	}

	root := n.Comments.Nodes[0]
	review := root.Review
	if strings.IsEmpty(root.ID) || root.ViewerDidAuthor == nil || review == nil {
		return errors.New("github: review thread returned incomplete ownership")
	}
	if strings.IsEmpty(review.ID) || review.Body == nil || review.ViewerDidAuthor == nil || !review.PullRequest.matches(repo, pull) {
		return errors.New("github: review thread returned incomplete or mismatched review ownership")
	}

	return nil
}
