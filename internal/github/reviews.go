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

// AddReview submits one advisory review pinned to pull's reviewed head. A
// concurrent push may make it old; commit_id is not a current-head guard.
func (s *service) AddReview(ctx context.Context, repo *Repository, pull *PullRequest, review *PullRequestReviewRequest) error {
	publication := *review
	publication.Event = new("COMMENT")
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
	isCommented := created.GetState() == "COMMENTED"
	isSubmitted := !created.GetSubmittedAt().IsZero()
	if !bodyMatches || !commitMatches || !isCommented || !isSubmitted {
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
	var ids []string
	for _, review := range reviews {
		if review.GetUser().GetType() == "Bot" {
			ids = append(ids, review.GetNodeID())
		}
	}

	for batch := range slices.Chunk(ids, pageSize) {
		body := PullRequestReviewAuthorsRequest{Query: reviewAuthorsQuery}
		body.Variables.IDs = batch

		request, err := s.client.installation.NewRequest(ctx, http.MethodPost, "graphql", body)
		if err != nil {
			return false, err
		}
		request.Header.Set(http.AcceptKey, media.JSON)

		var response PullRequestReviewAuthorsResponse
		if _, err := s.client.installation.Do(request, &response); err != nil {
			return false, err
		}
		if err := response.Validate(batch); err != nil {
			return false, err
		}

		for _, node := range response.Data.Nodes {
			if *node.ViewerDidAuthor {
				return true, nil
			}
		}
	}

	return false, nil
}

func (s *service) validateReview(repo *Repository, pull *PullRequest, review *PullRequestReview) error {
	pullPath := "repos/" + repo.GetFullName() + "/pulls/" + strconv.Itoa(pull.GetNumber())
	expectedURL := strings.ToLower(s.client.installation.BaseURL() + pullPath)
	reviewURL := strings.ToLower(review.GetPullRequestURL())
	reviewID := review.GetID()
	nodeID := review.GetNodeID()
	if reviewID <= 0 || strings.IsEmpty(nodeID) || reviewURL != expectedURL {
		return errors.New("github: review returned an incomplete or mismatched identity")
	}

	return nil
}
