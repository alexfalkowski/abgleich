package github

import (
	"slices"
	"strconv"

	"github.com/alexfalkowski/go-service/v2/context"
	"github.com/alexfalkowski/go-service/v2/errors"
	"github.com/alexfalkowski/go-service/v2/net/http"
	"github.com/alexfalkowski/go-service/v2/net/http/media"
	"github.com/alexfalkowski/go-service/v2/strings"
	"github.com/google/go-github/v93/github"
)

// Native REST reviews omit performed_via_github_app. Authenticated GraphQL
// authorship identifies this installation without trusting a login or body marker.
const reviewAuthorsQuery = `query PullRequestReviewAuthors($ids: [ID!]!) {
  nodes(ids: $ids) {
    ... on PullRequestReview { id viewer_did_author: viewerDidAuthor }
  }
}`

var reviewStates = map[string]string{
	"COMMENT":         "COMMENTED",
	"APPROVE":         "APPROVED",
	"REQUEST_CHANGES": "CHANGES_REQUESTED",
}

// AddReview submits one review pinned to pull's reviewed head. A
// concurrent push may make it old; commit_id is not a current-head guard.
func (s *service) AddReview(ctx context.Context, repo *Repository, pull *PullRequest, review *PullRequestReviewRequest) error {
	publication := *review
	expectedState, ok := reviewStates[publication.GetEvent()]
	if !ok {
		return errors.New("github: unsupported review event")
	}
	publication.CommitID = new(pull.GetHead().GetSHA())

	created, _, err := s.client.installation.PullRequests.CreateReview(ctx, repo.GetOwner().GetLogin(), repo.GetName(), pull.GetNumber(), &publication)
	if err != nil {
		return err
	}

	if err := s.validateReview(repo, pull, created); err != nil {
		return err
	}

	bodyMatches := created.GetBody() == publication.GetBody()
	commitMatches := created.GetCommitID() == publication.GetCommitID()
	stateMatches := created.GetState() == expectedState
	isSubmitted := !created.GetSubmittedAt().IsZero()
	if !bodyMatches || !commitMatches || !stateMatches || !isSubmitted {
		return errors.New("github: review publication returned mismatched content or submission state")
	}

	authored, err := s.HasAppReview(ctx, []*PullRequestReview{created})
	if err != nil {
		return err
	}
	if !authored {
		return errors.New("github: review publication returned no authenticated App author")
	}

	return nil
}

// ListReviews collects every native review page before returning history.
func (s *service) ListReviews(ctx context.Context, repo *Repository, pull *PullRequest) ([]*PullRequestReview, error) {
	var reviews []*PullRequestReview
	opts := &github.ListOptions{PerPage: pageSize}

	for {
		page, response, err := s.client.installation.PullRequests.ListReviews(ctx, repo.GetOwner().GetLogin(), repo.GetName(), pull.GetNumber(), opts)
		if err != nil {
			return nil, err
		}
		if page == nil {
			return nil, errors.New("github: review history returned a missing page")
		}

		for _, review := range page {
			if err := s.validateReview(repo, pull, review); err != nil {
				return nil, err
			}
		}

		reviews = append(reviews, page...)
		if response.NextPage == 0 {
			return reviews, nil
		}
		if response.NextPage <= opts.Page {
			return nil, errors.New("github: review history has invalid pagination")
		}

		opts.Page = response.NextPage
	}
}

// HasAppReview authenticates bot candidates in batches so forged markers cannot
// cause one API request per review. Missing or mismatched data fails closed.
func (s *service) HasAppReview(ctx context.Context, reviews []*PullRequestReview) (bool, error) {
	authored, err := s.AppReviews(ctx, reviews)
	return len(authored) > 0, err
}

// AppReviews returns authenticated App reviews in their input order. Authorship
// is checked in batches; incomplete or mismatched responses fail closed.
func (s *service) AppReviews(ctx context.Context, reviews []*PullRequestReview) ([]*PullRequestReview, error) {
	var ids []string
	for _, review := range reviews {
		if review.GetUser().GetType() == "Bot" {
			ids = append(ids, review.GetNodeID())
		}
	}

	authored := make(map[string]bool, len(ids))
	for batch := range slices.Chunk(ids, pageSize) {
		body := PullRequestReviewAuthorsRequest{Query: reviewAuthorsQuery}
		body.Variables.IDs = batch

		request, err := s.client.installation.NewRequest(ctx, http.MethodPost, "graphql", body)
		if err != nil {
			return nil, err
		}
		request.Header.Set(http.AcceptKey, media.JSON)

		var response PullRequestReviewAuthorsResponse
		if _, err := s.client.installation.Do(request, &response); err != nil {
			return nil, err
		}
		if err := response.Validate(batch); err != nil {
			return nil, err
		}

		for _, node := range response.Data.Nodes {
			authored[node.ID] = *node.ViewerDidAuthor
		}
	}

	var matches []*PullRequestReview
	for _, review := range reviews {
		if authored[review.GetNodeID()] {
			matches = append(matches, review)
		}
	}
	return matches, nil
}

func (s *service) validateReview(repo *Repository, pull *PullRequest, review *PullRequestReview) error {
	pullPath := strings.Concat("repos/", repo.GetFullName(), "/pulls/", strconv.Itoa(pull.GetNumber()))
	expectedURL := strings.ToLower(s.client.installation.BaseURL() + pullPath)
	reviewURL := strings.ToLower(review.GetPullRequestURL())
	reviewID := review.GetID()
	nodeID := review.GetNodeID()
	if reviewID <= 0 || strings.IsEmpty(nodeID) || reviewURL != expectedURL {
		return errors.New("github: review returned an incomplete or mismatched identity")
	}

	return nil
}

// PullRequestReviewAuthorsRequest queries authenticated authorship for a batch
// of review node IDs.
type PullRequestReviewAuthorsRequest struct {
	// Query contains the GraphQL operation submitted with these variables.
	Query string `json:"query"`
	// Variables binds the operation's GraphQL arguments.
	Variables struct {
		// IDs lists review node IDs whose authenticated authorship must be checked.
		IDs []string `json:"ids"`
	} `json:"variables"`
}

// PullRequestReviewAuthorsResponse requires each requested review identity and
// explicit authorship. Call Validate even when the HTTP request succeeds.
type PullRequestReviewAuthorsResponse struct {
	// Data contains returned operation data, which still requires error and identity validation.
	Data struct {
		// Nodes returns one authorship result for each requested review identity.
		Nodes []*struct {
			// ViewerDidAuthor reports authenticated viewer authorship; nil cannot establish ownership.
			ViewerDidAuthor *bool `json:"viewer_did_author"`
			// ID identifies the review whose authorship was returned.
			ID string `json:"id"`
		} `json:"nodes"`
	} `json:"data"`
	// Errors retains GraphQL failures even when the HTTP response succeeds.
	Errors Errors `json:"errors"`
}

// Validate requires exactly one explicit authorship result for every requested
// ID, regardless of response order. False is a valid authorship result.
func (r *PullRequestReviewAuthorsResponse) Validate(expectedIDs []string) error {
	if err := r.Errors.Validate(); err != nil {
		return errors.Prefix("github: review authorship", err)
	}
	if len(r.Data.Nodes) != len(expectedIDs) {
		return errors.New("github: review authorship returned an incomplete batch")
	}

	expected := make(map[string]bool, len(expectedIDs))
	for _, id := range expectedIDs {
		expected[id] = true
	}
	for _, node := range r.Data.Nodes {
		if node == nil || !expected[node.ID] || node.ViewerDidAuthor == nil {
			return errors.New("github: review authorship returned no matching review")
		}
		delete(expected, node.ID)
	}

	return nil
}
