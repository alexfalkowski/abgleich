package github

import (
	"github.com/alexfalkowski/go-service/v2/context"
	"github.com/alexfalkowski/go-service/v2/errors"
	"github.com/alexfalkowski/go-service/v2/net/http"
	"github.com/alexfalkowski/go-service/v2/net/http/media"
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
		request, err := s.client.installation.NewRequest(ctx, http.MethodPost, "graphql", body)
		if err != nil {
			return nil, err
		}

		request.Header.Set(http.AcceptKey, media.JSON)

		var response ReviewThreadsResponse
		if _, err := s.client.installation.Do(request, &response); err != nil {
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

	request, err := s.client.installation.NewRequest(ctx, http.MethodPost, "graphql", body)
	if err != nil {
		return nil, err
	}

	request.Header.Set(http.AcceptKey, media.JSON)

	var response ReviewThreadResponse
	if _, err := s.client.installation.Do(request, &response); err != nil {
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

	request, err := s.client.installation.NewRequest(ctx, http.MethodPost, "graphql", body)
	if err != nil {
		return err
	}

	request.Header.Set(http.AcceptKey, media.JSON)

	var response ResolveReviewThreadResponse
	if _, err := s.client.installation.Do(request, &response); err != nil {
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
