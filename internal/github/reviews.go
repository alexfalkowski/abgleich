package github

import (
	"strconv"

	"github.com/alexfalkowski/go-service/v3/context"
	"github.com/alexfalkowski/go-service/v3/errors"
	"github.com/alexfalkowski/go-service/v3/strings"
	"github.com/google/go-github/v93/github"
)

// ReviewEvent selects the verdict requested when publishing a native review.
type ReviewEvent string

// String returns the event value sent when publishing a review.
func (e ReviewEvent) String() string {
	return string(e)
}

// ReviewState identifies the recorded outcome of a submitted or dismissed review.
type ReviewState string

// String returns the state value recorded by GitHub.
func (s ReviewState) String() string {
	return string(s)
}

// DiffSide selects the base or head side of a pull request patch.
type DiffSide string

// String returns the side value used for inline review placement.
func (s DiffSide) String() string {
	return string(s)
}

// IsDefined reports whether a diff side is present, without validating its value.
func (s DiffSide) IsDefined() bool {
	return !strings.IsEmpty(s.String())
}

// ReviewComment submits feedback without approving or requesting changes.
const ReviewComment ReviewEvent = "COMMENT"

// ReviewApprove submits an approving review.
const ReviewApprove ReviewEvent = "APPROVE"

// ReviewRequestChanges submits findings that request changes.
const ReviewRequestChanges ReviewEvent = "REQUEST_CHANGES"

// ReviewCommented identifies submitted feedback without a verdict.
const ReviewCommented ReviewState = "COMMENTED"

// ReviewApproved identifies a submitted approval.
const ReviewApproved ReviewState = "APPROVED"

// ReviewChangesRequested identifies a submitted request for changes.
const ReviewChangesRequested ReviewState = "CHANGES_REQUESTED"

// ReviewDismissed identifies a review dismissed after submission.
const ReviewDismissed ReviewState = "DISMISSED"

// DiffLeft places a finding on the base side of a pull request patch.
const DiffLeft DiffSide = "LEFT"

// DiffRight places a finding on the head side of a pull request patch.
const DiffRight DiffSide = "RIGHT"

var reviewStates = map[ReviewEvent]ReviewState{
	ReviewComment:        ReviewCommented,
	ReviewApprove:        ReviewApproved,
	ReviewRequestChanges: ReviewChangesRequested,
}

// AddReview submits one review pinned to pull's reviewed head. A
// concurrent push may make it old; commit_id is not a current-head guard.
func (s *service) AddReview(ctx context.Context, repo *Repository, pull *PullRequest, review *PullRequestReviewRequest) error {
	publication := *review
	expectedState, ok := reviewStates[ReviewEvent(publication.GetEvent())]
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
	stateMatches := created.GetState() == expectedState.String()
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
