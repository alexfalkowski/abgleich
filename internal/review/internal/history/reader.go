package history

import (
	"crypto/sha256"
	"fmt"
	"regexp"
	"strconv"

	"github.com/alexfalkowski/abgleich/internal/github"
	"github.com/alexfalkowski/go-service/v3/context"
	"github.com/alexfalkowski/go-service/v3/strings"
)

// Partial reviews retain feature identity for verdicts and thread maintenance.
// Their markers do not match successful results during deduplication.
var reviewThreadMarker = regexp.MustCompile(`(?m)^<!-- abgleich:([a-z]+):v2:(?:partial:)?[a-f0-9]{64} -->$`)

// Reader reads and authenticates persisted review results for a feature and PR.
type Reader struct {
	github     github.Service
	repository *github.Repository
	pull       *github.PullRequest
	feature    string
}

// NewReader binds a reader to the current PR snapshot. Each Read uses fresh history.
func NewReader(service github.Service, repository *github.Repository, pull *github.PullRequest, feature string) Reader {
	return Reader{github: service, repository: repository, pull: pull, feature: feature}
}

// Feature returns the namespace of a full or partial result marker.
// Callers must authenticate the enclosing review before trusting the namespace.
func Feature(body string) string {
	match := reviewThreadMarker.FindStringSubmatch(body)
	if len(match) == 2 {
		return match[1]
	}
	return strings.Empty
}

// View holds the native history and any assessments authenticated during one read.
type View struct {
	reviews       []*github.PullRequestReview
	authenticated []*github.PullRequestReview
	authChecked   bool
}

// Read looks for a reusable result using fresh native reviews and the caller's
// freshly authorized discussion history. The view is valid for this check only.
func (r Reader) Read(ctx context.Context, comments []*github.IssueComment) (bool, View, error) {
	marker := r.ResultMarker()
	legacy := r.latestMatchingComment(comments, marker)

	reviews, err := r.github.ListReviews(ctx, r.repository, r.pull)
	if err != nil {
		return false, View{}, err
	}
	history := View{reviews: reviews}

	if legacy == nil && !containsResult(reviews, marker) {
		return false, history, nil
	}

	authored, err := r.authenticatedAssessments(ctx, reviews)
	if err != nil {
		return false, View{}, err
	}

	history.authenticated, history.authChecked = authored, true

	latest := latestAssessment(authored)
	if legacy != nil && (latest == nil || latest.GetSubmittedAt().Before(legacy.GetCreatedAt().Time)) {
		return true, history, nil
	}
	return strings.Contains(latest.GetBody(), "\n"+marker), history, nil
}

func (r Reader) latestMatchingComment(comments []*github.IssueComment, marker string) *github.IssueComment {
	var latest *github.IssueComment
	for _, comment := range comments {
		if r.github.IsAppComment(comment) && strings.Contains(comment.GetBody(), "\n"+marker) &&
			(latest == nil || latest.GetCreatedAt().Before(comment.GetCreatedAt().Time)) {
			latest = comment
		}
	}
	return latest
}

func (r Reader) assessments(ctx context.Context, history View) ([]*github.PullRequestReview, error) {
	if history.authChecked {
		return history.authenticated, nil
	}
	return r.authenticatedAssessments(ctx, history.reviews)
}

func containsResult(reviews []*github.PullRequestReview, marker string) bool {
	for _, review := range reviews {
		if strings.Contains(review.GetBody(), "\n"+marker) {
			return true
		}
	}
	return false
}

// ResultMarker identifies a completed result for the reader's reviewed revisions.
func (r Reader) ResultMarker() string {
	return r.resultMarkerAt(r.pull.GetHead().GetSHA(), r.pull.GetBase().GetSHA(), false)
}

func (r Reader) resultMarkerAt(headSHA, baseSHA string, partial bool) string {
	identity := strings.Join("\n",
		r.feature,
		strings.ToLower(r.repository.GetFullName()),
		strconv.Itoa(r.pull.GetNumber()),
		r.pull.GetNodeID(),
		headSHA,
		baseSHA,
	)
	var kind string
	if partial {
		kind = "partial:"
	}
	return fmt.Sprintf("<!-- abgleich:%s:v2:%s%x -->", r.feature, kind, sha256.Sum256(strings.Bytes(identity)))
}

// PartialMarker records a result that still blocks changes and permits a retry.
func PartialMarker(marker string) string {
	return strings.ReplaceAll(marker, ":v2:", ":v2:partial:")
}
