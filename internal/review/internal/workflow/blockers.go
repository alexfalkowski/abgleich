package workflow

import (
	"crypto/sha256"
	"fmt"
	"strings"

	"github.com/alexfalkowski/abgleich/internal/github"
	"github.com/alexfalkowski/go-service/v3/context"
	"github.com/alexfalkowski/go-service/v3/slices"
)

// A snapshot contains identities and a digest of the visible finding, which
// lets a follow-up recognize edits without persisting a second copy of the text.
type blockerReference struct {
	ID       string `json:"previous_id"`
	Digest   string `json:"digest"`
	ReviewID int64  `json:"review_id,omitempty"`
	Inline   bool   `json:"inline,omitempty"`
}

type reviewBlocker struct {
	ID       string `json:"previous_id"`
	Evidence string `json:"evidence,omitempty"`
	reviewID int64
}

func (r *reviewContext) hasBlockerEvidence() bool {
	return r != nil && r.followup != nil && slices.ContainsFunc(r.followup.Blockers, func(blocker reviewBlocker) bool {
		return blocker.Evidence != ""
	})
}

func findingDigest(body string) string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(strings.TrimSpace(body))))
}

func (c *Command) readBlockers(ctx context.Context, task *reviewTask, refs []blockerReference, reviews []*github.PullRequestReview) []reviewBlocker {
	origins := make(map[int64]*github.PullRequestReview, len(reviews))
	for _, review := range reviews {
		origins[review.GetID()] = review
	}

	comments := make(map[int64][]string)
	blockers := make([]reviewBlocker, len(refs))
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
				comments[ref.ReviewID], _ = c.github.ListReviewCommentBodies(ctx, task.repository, task.request, origin)
			}
			blocker.Evidence = inlineEvidence(ref, comments[ref.ReviewID])
		} else {
			blocker.Evidence = overviewEvidence(ref, origin.GetBody())
		}
		if blocker.Evidence != "" {
			blocker.reviewID = ref.ReviewID
		}
	}
	return blockers
}

func inlineEvidence(ref blockerReference, bodies []string) string {
	for _, body := range bodies {
		if findingDigest(body) == ref.Digest {
			return strings.TrimSpace(body)
		}
	}
	return ""
}

func overviewEvidence(ref blockerReference, body string) string {
	_, tail, found := strings.Cut(body, "\n<!-- abgleich:finding:"+ref.ID+" -->\n")
	if !found {
		return ""
	}
	evidence, _, found := strings.Cut(tail, "\n<!-- /abgleich:finding:"+ref.ID+" -->")
	if !found || findingDigest(evidence) != ref.Digest {
		return ""
	}
	return strings.TrimSpace(evidence)
}
