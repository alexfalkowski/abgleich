package github

import (
	_ "embed"
	"slices"

	"github.com/alexfalkowski/go-service/v2/context"
	"github.com/alexfalkowski/go-service/v2/errors"
)

// Native REST reviews omit performed_via_github_app. Authenticated GraphQL
// authorship identifies this installation without trusting a login or body marker.
//
//go:embed graphql/review_authors.gql
var reviewAuthorsQuery string

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
		if review.GetUser().GetType() == UserBot {
			ids = append(ids, review.GetNodeID())
		}
	}

	authored := make(map[string]bool, len(ids))
	for batch := range slices.Chunk(ids, pageSize) {
		body := PullRequestReviewAuthorsRequest{Query: reviewAuthorsQuery}
		body.Variables.IDs = batch

		var response PullRequestReviewAuthorsResponse
		if err := s.graphql(ctx, body, &response); err != nil {
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
