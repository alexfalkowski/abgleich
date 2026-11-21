package history

import (
	"github.com/alexfalkowski/abgleich/internal/github"
	"github.com/alexfalkowski/go-service/v3/context"
	"github.com/alexfalkowski/go-service/v3/slices"
	"github.com/alexfalkowski/go-service/v3/strings"
)

// NoFindings denotes an approval-eligible review without findings.
const NoFindings = "No findings."

// LowSeverityFindings denotes an approval-eligible review with only P3 findings.
const LowSeverityFindings = "Low-severity findings."

// AssessmentLine renders the persisted assessment line used when interpreting
// authenticated reviews whose approval was withheld by another feature.
func AssessmentLine(assessment string) string {
	return "**Assessment:** " + assessment
}

// OtherFeatureBlocks reports whether another feature has an authenticated,
// unresolved request for changes, including features disabled since publication.
func (r Reader) OtherFeatureBlocks(ctx context.Context, view View) (bool, error) {
	var candidates []*github.PullRequestReview
	for _, review := range view.reviews {
		if name := Feature(review.GetBody()); name != "" && name != r.feature {
			candidates = append(candidates, review)
		}
	}

	reviews, err := r.github.AppReviews(ctx, candidates)
	if err != nil {
		return false, err
	}

	// Resolve each feature independently, including features no longer enabled.
	// Incomplete comments do not clear prior findings, including across revisions.
	// A complete assessment with at most P3 findings clears its feature even if another feature blocks approval.
	assessed := make(map[string]bool)
	for _, review := range slices.Backward(reviews) {
		name := Feature(review.GetBody())
		if assessed[name] {
			continue
		}

		switch github.ReviewState(review.GetState()) {
		case github.ReviewChangesRequested:
			return true, nil
		case github.ReviewApproved, github.ReviewDismissed:
			assessed[name] = true
		case github.ReviewCommented:
			assessed[name] = isApprovalAssessment(review.GetBody())
		}
	}
	return false, nil
}

func isApprovalAssessment(body string) bool {
	return strings.Contains(body, "\n"+AssessmentLine(NoFindings)+"\n") ||
		strings.Contains(body, "\n"+AssessmentLine(LowSeverityFindings)+"\n")
}
