package history

import (
	"crypto/sha256"
	"fmt"
	"strings"

	"github.com/alexfalkowski/abgleich/internal/github"
	"github.com/alexfalkowski/go-service/v3/context"
	"github.com/alexfalkowski/go-service/v3/slices"
)

// Reference identifies an earlier finding by the digest of its visible text.
type Reference struct {
	// ID is the stable finding identity used in follow-up reviews.
	ID string `json:"previous_id"`
	// Digest authenticates the exact visible finding text against the stored record.
	Digest string `json:"digest"`
	// ReviewID identifies the original review. Zero identifies the review storing this reference.
	ReviewID int64 `json:"review_id,omitempty"`
	// Inline selects review comments as the original location.
	Inline bool `json:"inline,omitempty"`
}

// Blocker carries an earlier finding into a follow-up.
type Blocker struct {
	// ID is the stable finding identity used in follow-up reviews.
	ID string `json:"previous_id"`
	// Evidence contains intact original finding text, when available.
	Evidence string `json:"evidence,omitempty"`
	// ReviewID identifies the original review. Zero indicates missing evidence.
	ReviewID int64 `json:"-"`
}

// HasBlockerEvidence reports whether a follow-up will send prior finding text.
func (f *Followup) HasBlockerEvidence() bool {
	return f != nil && slices.ContainsFunc(f.Blockers, func(blocker Blocker) bool {
		return blocker.Evidence != ""
	})
}

// FindingDigest fingerprints the displayed finding so edited evidence can be detected.
func FindingDigest(body string) string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(strings.TrimSpace(body))))
}

// Missing or edited finding text leaves the reference unresolved.
func (r Reader) blockers(ctx context.Context, refs []Reference, reviews []*github.PullRequestReview) []Blocker {
	origins := make(map[int64]*github.PullRequestReview, len(reviews))
	for _, review := range reviews {
		origins[review.GetID()] = review
	}

	comments := make(map[int64][]string)
	blockers := make([]Blocker, len(refs))
	for index, ref := range refs {
		blocker := &blockers[index]
		blocker.ID = ref.ID

		origin := origins[ref.ReviewID]
		if origin == nil || ref.Digest == "" {
			continue
		}

		if ref.Inline {
			if _, read := comments[ref.ReviewID]; !read {
				// An unavailable origin leaves its blockers unresolved.
				comments[ref.ReviewID], _ = r.github.ListReviewCommentBodies(ctx, r.repository, r.pull, origin)
			}
			blocker.Evidence = inlineEvidence(ref, comments[ref.ReviewID])
		} else {
			blocker.Evidence = overviewEvidence(ref, origin.GetBody())
		}

		if blocker.Evidence != "" {
			blocker.ReviewID = ref.ReviewID
		}
	}
	return blockers
}

func inlineEvidence(ref Reference, bodies []string) string {
	for _, body := range bodies {
		if FindingDigest(body) == ref.Digest {
			return strings.TrimSpace(body)
		}
	}
	return ""
}

func overviewEvidence(ref Reference, body string) string {
	_, tail, found := strings.Cut(body, "\n<!-- abgleich:finding:"+ref.ID+" -->\n")
	if !found {
		return ""
	}
	evidence, _, found := strings.Cut(tail, "\n<!-- /abgleich:finding:"+ref.ID+" -->")
	if !found || FindingDigest(evidence) != ref.Digest {
		return ""
	}
	return strings.TrimSpace(evidence)
}
