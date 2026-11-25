package history

import (
	"github.com/alexfalkowski/abgleich/internal/github"
	"github.com/alexfalkowski/go-service/v3/context"
	"github.com/alexfalkowski/go-service/v3/slices"
	"github.com/alexfalkowski/go-service/v3/strings"
)

// Previous holds the authenticated baseline for one attempt.
type Previous struct {
	latest      *github.PullRequestReview
	snapshot    *Snapshot
	assessments []*github.PullRequestReview
}

// ReadPrevious authenticates the feature's history and checks the latest
// follow-up record before the workflow collects source.
func (r Reader) ReadPrevious(ctx context.Context, view View) (Previous, error) {
	assessments, err := r.assessments(ctx, view)
	if err != nil {
		return Previous{}, err
	}

	latest := latestAssessment(assessments)

	snapshot, err := r.snapshot(latest)
	if err != nil {
		return Previous{}, err
	}
	return Previous{assessments: assessments, latest: latest, snapshot: snapshot}, nil
}

// ReadFollowup reads intact original finding text for a usable previous record.
// The snapshot supplies the old fingerprints for source comparison.
func (r Reader) ReadFollowup(ctx context.Context, previous Previous) (*Followup, *Snapshot) {
	snapshot := previous.snapshot
	if snapshot == nil {
		return nil, nil
	}
	return &Followup{
		HeadSHA: snapshot.HeadSHA, BaseSHA: snapshot.BaseSHA, References: snapshot.Blockers,
		Blockers: r.blockers(ctx, snapshot.Blockers, previous.assessments),
	}, snapshot
}

func (r Reader) authenticatedAssessments(ctx context.Context, reviews []*github.PullRequestReview) ([]*github.PullRequestReview, error) {
	var candidates []*github.PullRequestReview
	for _, review := range reviews {
		if r.assessmentCandidate(review) {
			candidates = append(candidates, review)
		}
	}

	return r.github.AppReviews(ctx, candidates)
}

func latestAssessment(reviews []*github.PullRequestReview) *github.PullRequestReview {
	if len(reviews) == 0 {
		return nil
	}
	return reviews[len(reviews)-1]
}

func (r Reader) snapshot(review *github.PullRequestReview) (*Snapshot, error) {
	state := github.ReviewState(review.GetState())
	if review == nil || state == github.ReviewDismissed {
		return nil, nil
	}

	snapshot := decodeSnapshot(review.GetBody())
	marker := reviewThreadMarker.FindString(review.GetBody())
	partial := strings.Contains(marker, ":v2:partial:")
	if !r.matchesSnapshot(review, snapshot, marker, partial) {
		if partial && state == github.ReviewChangesRequested {
			return nil, ErrPartialBlockersUnavailable
		}
		return nil, nil
	}

	for index := range snapshot.Blockers {
		if snapshot.Blockers[index].ReviewID == 0 {
			snapshot.Blockers[index].ReviewID = review.GetID()
		}
	}
	return snapshot, nil
}

func (r Reader) matchesSnapshot(review *github.PullRequestReview, snapshot *Snapshot, marker string, partial bool) bool {
	return snapshot != nil && snapshot.HeadSHA == review.GetCommitID() &&
		(!partial || len(snapshot.Blockers) > 0) && marker == r.resultMarkerAt(snapshot.HeadSHA, snapshot.BaseSHA, partial)
}

func (r Reader) assessmentCandidate(review *github.PullRequestReview) bool {
	marker := reviewThreadMarker.FindStringSubmatch(review.GetBody())
	if len(marker) != 2 || marker[1] != r.feature {
		return false
	}

	switch github.ReviewState(review.GetState()) {
	case github.ReviewDismissed, github.ReviewApproved, github.ReviewChangesRequested:
		return true
	case github.ReviewCommented:
		return !strings.Contains(marker[0], ":v2:partial:")
	default:
		return false
	}
}

// CheckPrevious compares the current authenticated history and finding text
// with the evidence already collected for this attempt. A reason ends the attempt.
func (r Reader) CheckPrevious(ctx context.Context, view View, previous Previous, followup *Followup) (string, error) {
	authored, err := r.assessments(ctx, view)
	if err != nil {
		return strings.Empty, err
	}
	if !sameAssessment(previous.latest, latestAssessment(authored)) {
		return "The App review history changed. Findings from this attempt were discarded.", nil
	}
	if followup == nil || len(followup.References) == 0 {
		return strings.Empty, nil
	}

	current := r.blockers(ctx, followup.References, authored)
	if !slices.Equal(current, followup.Blockers) {
		return "Earlier finding text changed or became unavailable. Findings from this attempt were discarded.", nil
	}
	return strings.Empty, nil
}

func sameAssessment(previous, current *github.PullRequestReview) bool {
	return previous.GetID() == current.GetID() && previous.GetState() == current.GetState() &&
		previous.GetCommitID() == current.GetCommitID() && previous.GetBody() == current.GetBody()
}
