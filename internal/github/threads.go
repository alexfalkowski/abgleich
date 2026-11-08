package github

import (
	_ "embed"

	"github.com/alexfalkowski/go-service/v2/context"
	"github.com/alexfalkowski/go-service/v2/errors"
	"github.com/alexfalkowski/go-service/v2/strings"
)

// reviewThreadFields keeps discovery and final rechecks on the same evidence.
// Only the root comment is needed: viewerDidAuthor on it and its parent review
// authenticates App ownership, while the review body identifies the feature's
// result namespace. Replies do not affect maintenance eligibility.
//
//go:embed graphql/review_thread_fields.gql
var reviewThreadFields string

// reviewThreadsQuery pages through a PR's inline conversations by PR node ID.
// Parent identities prevent accepting threads from another PR or repository;
// pageInfo lets the caller finish discovery before resolving any thread.
//
//go:embed graphql/review_threads.gql
var reviewThreadsQuery string

// reviewThreadQuery refreshes one candidate by thread node ID before resolution.
// The shared fields recheck ownership, outdated/resolved state, permission,
// and whether the parent PR is still open.
//
//go:embed graphql/review_thread.gql
var reviewThreadQuery string

// ListReviewThreads reads complete pagination and only the root of each thread.
func (s *service) ListReviewThreads(ctx context.Context, repo *Repository, pull *PullRequest) ([]*ReviewThread, error) {
	body := ReviewThreadsRequest{Query: reviewThreadsQuery + reviewThreadFields}
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
	body := ReviewThreadRequest{Query: reviewThreadQuery + reviewThreadFields}
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

	if wire.PullRequest.State != graphQLOpen {
		return nil, nil
	}

	return thread, nil
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
