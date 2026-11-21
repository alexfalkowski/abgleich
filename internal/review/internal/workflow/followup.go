package workflow

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"maps"
	"strings"

	"github.com/alexfalkowski/abgleich/internal/github"
	"github.com/alexfalkowski/go-service/v3/context"
	"github.com/alexfalkowski/go-service/v3/encoding/json"
	"github.com/alexfalkowski/go-service/v3/errors"
	"github.com/alexfalkowski/go-service/v3/slices"
)

const snapshotPrefix = "<!-- abgleich:followup:v1:"

var errPartialReviewBlockersUnavailable = errors.New("review: partial review blocker record unavailable")

type reviewSnapshot struct {
	HeadSHA  string             `json:"head"`
	BaseSHA  string             `json:"base"`
	Patches  map[string]string  `json:"patches,omitempty"`
	Sources  map[string]string  `json:"sources,omitempty"`
	Blockers []blockerReference `json:"blockers"`
}

type reviewFollowup struct {
	HeadSHA               string          `json:"previous_head"`
	BaseSHA               string          `json:"previous_base"`
	ChangedFiles          []string        `json:"changed_patches"`
	ChangedSource         []string        `json:"changed_source"`
	ComparisonUnavailable []string        `json:"comparison_unavailable,omitempty"`
	Blockers              []reviewBlocker `json:"open_blockers"`
	references            []blockerReference
}

func (c *Command) authenticatedAssessments(ctx context.Context, reviews []*github.PullRequestReview) ([]*github.PullRequestReview, error) {
	var candidates []*github.PullRequestReview
	for _, review := range reviews {
		if c.assessmentCandidate(review) {
			candidates = append(candidates, review)
		}
	}

	return c.github.AppReviews(ctx, candidates)
}

func lastAssessment(reviews []*github.PullRequestReview) *github.PullRequestReview {
	if len(reviews) == 0 {
		return nil
	}
	return reviews[len(reviews)-1]
}

func (c *Command) assessmentSnapshot(task *reviewTask, review *github.PullRequestReview) (*reviewSnapshot, error) {
	state := github.ReviewState(review.GetState())
	if review == nil || state == github.ReviewDismissed {
		return nil, nil
	}

	snapshot := decodeSnapshot(review.GetBody())
	marker := reviewThreadMarker.FindString(review.GetBody())
	partial := strings.Contains(marker, ":v2:partial:")
	if !c.matchesSnapshot(task, review, snapshot, marker, partial) {
		if partial && state == github.ReviewChangesRequested {
			return nil, errPartialReviewBlockersUnavailable
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

func (c *Command) matchesSnapshot(task *reviewTask, review *github.PullRequestReview, snapshot *reviewSnapshot, marker string, partial bool) bool {
	return snapshot != nil && snapshot.HeadSHA == review.GetCommitID() &&
		(!partial || len(snapshot.Blockers) > 0) && marker == c.resultMarkerAt(task, snapshot.HeadSHA, snapshot.BaseSHA, partial)
}

func (c *Command) assessmentCandidate(review *github.PullRequestReview) bool {
	marker := reviewThreadMarker.FindStringSubmatch(review.GetBody())
	if len(marker) != 2 || marker[1] != c.profile.Name {
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

func (r *reviewContext) followupContext(ctx context.Context, previous *reviewSnapshot, blockers []reviewBlocker) *reviewFollowup {
	followup := &reviewFollowup{
		HeadSHA: previous.HeadSHA, BaseSHA: previous.BaseSHA, Blockers: blockers, references: previous.Blockers,
	}
	if previous.Patches == nil {
		followup.ComparisonUnavailable = []string{"Previous patch and source fingerprints were unavailable."}
		return followup
	}
	followup.ChangedFiles = changedPatches(previous.Patches, r.patches)

	// A path is compared on the same side of the PR, even when that side moved.
	for _, source := range []struct{ side, revision string }{{"head", r.HeadSHA}, {"base", r.BaseSHA}} {
		changed, err := r.source.changedSourcePaths(ctx, source.revision, source.side+":", previous.Sources)
		if err != nil {
			followup.ComparisonUnavailable = append(followup.ComparisonUnavailable, "Could not compare previous source on the "+source.side+" revision.")
			continue
		}
		followup.ChangedSource = append(followup.ChangedSource, changed...)
	}
	slices.Sort(followup.ChangedSource)
	return followup
}

func (s *sourceSession) changedSourcePaths(ctx context.Context, revision, prefix string, previous map[string]string) ([]string, error) {
	names := make(map[string]string)
	for path, fingerprint := range previous {
		if name, ok := strings.CutPrefix(path, prefix); ok {
			names[name] = fingerprint
		}
	}
	if len(names) == 0 {
		return nil, nil
	}

	entries, err := s.treeEntries(ctx, revision)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		path := entry.GetPath()
		if names[path] == sourceFingerprint(entry) {
			delete(names, path)
		}
	}

	changed := make([]string, 0, len(names))
	for path := range names {
		changed = append(changed, prefix+path)
	}
	return changed, nil
}

func patchFingerprints(files []*github.CommitFile) map[string]string {
	patches := make(map[string]string, len(files))
	for _, file := range files {
		text := file.GetStatus() + "\x00" + file.GetPreviousFilename() + "\x00" + file.GetPatch()
		patches[file.GetFilename()] = fmt.Sprintf("%x", sha256.Sum256([]byte(text)))
	}
	return patches
}

func changedPatches(previous, current map[string]string) []string {
	var paths []string
	for path, fingerprint := range current {
		if previous[path] != fingerprint {
			paths = append(paths, path)
		}
	}
	for path := range previous {
		if _, exists := current[path]; !exists {
			paths = append(paths, path)
		}
	}
	slices.Sort(paths)
	return paths
}

func sourceFingerprint(entry *github.TreeEntry) string {
	return entry.GetMode() + ":" + entry.GetSHA()
}

func decodeSnapshot(body string) *reviewSnapshot {
	_, tail, found := strings.Cut(body, "\n"+snapshotPrefix)
	if !found {
		return nil
	}

	marker, _, _ := strings.Cut(tail, "\n")
	encoded, found := strings.CutSuffix(marker, " -->")
	if !found {
		return nil
	}

	raw, err := base64.RawStdEncoding.DecodeString(encoded)
	if err != nil {
		return nil
	}

	var snapshot reviewSnapshot
	if err := json.Unmarshal(raw, &snapshot); err != nil || snapshot.HeadSHA == "" || snapshot.BaseSHA == "" {
		return nil
	}
	return &snapshot
}

func (r *reviewContext) snapshot(result *reviewResult, partial bool) *reviewSnapshot {
	snapshot := &reviewSnapshot{HeadSHA: r.HeadSHA, BaseSHA: r.BaseSHA, Patches: r.patches, Sources: r.source.fingerprints}
	previous := make(map[string]blockerReference)
	if r.followup != nil {
		for _, ref := range r.followup.references {
			previous[ref.ID] = ref
		}
	}
	for _, finding := range result.Findings {
		if finding.Severity != severityP3 {
			ref, carried := previous[finding.ID]
			if !carried {
				ref = finding.reference
			}
			snapshot.Blockers = append(snapshot.Blockers, ref)
			delete(previous, finding.ID)
		}
	}
	if partial {
		for _, id := range slices.Sorted(maps.Keys(previous)) {
			snapshot.Blockers = append(snapshot.Blockers, previous[id])
		}
	}
	return snapshot
}

func (snapshot *reviewSnapshot) encode() (string, error) {
	raw, err := json.Marshal(snapshot)
	if err != nil {
		return "", err
	}
	return snapshotPrefix + base64.RawStdEncoding.EncodeToString(raw) + " -->", nil
}

func (r *reviewReply) reconcileFindings(result *reviewResult, followup *reviewFollowup, seed string) {
	previous := make(map[string]int64)
	if followup != nil {
		for _, blocker := range followup.Blockers {
			previous[blocker.ID] = blocker.reviewID
		}
	}

	references := linkPreviousFindings(result, previous, seed)
	r.Cleared = matchClearances(result.Cleared, previous, references)
	*result.Complete = *result.Complete && len(previous) == 0
	if !*result.Complete || len(r.Missing) > 0 {
		for _, finding := range r.Cleared {
			previous[finding.ID] = 0
		}
		r.Cleared = nil
	}
	r.Unverified = slices.Sorted(maps.Keys(previous))
}

func matchClearances(cleared []clearedFinding, previous map[string]int64, references map[string]int) []clearedFinding {
	var matched []clearedFinding
	for _, finding := range cleared {
		if previous[finding.ID] > 0 && references[finding.ID] == 1 && strings.TrimSpace(finding.Reason) != "" {
			matched = append(matched, finding)
			delete(previous, finding.ID)
		}
	}
	return matched
}

func linkPreviousFindings(result *reviewResult, previous map[string]int64, seed string) map[string]int {
	// A reference is usable only when the model accounted for it exactly once.
	references := make(map[string]int)
	used := make(map[string]bool, len(previous)+len(result.Findings))
	for id := range previous {
		used[id] = true
	}
	for _, finding := range result.Findings {
		references[finding.ID]++
	}
	for _, finding := range result.Cleared {
		references[finding.ID]++
	}

	for index := range result.Findings {
		finding := &result.Findings[index]
		id := finding.ID
		finding.ID = ""
		finding.ReviewID = 0

		if finding.Severity == severityP3 {
			continue
		}

		if origin := previous[id]; origin > 0 && references[id] == 1 {
			finding.ID = id
			finding.ReviewID = origin
			delete(previous, id)
			continue
		}
		finding.ID = newFindingID(seed, index, used)
	}
	return references
}

// A redelivered request can reuse the seed, so reserve every previous ID.
func newFindingID(seed string, index int, used map[string]bool) string {
	for attempt := 0; ; attempt++ {
		digest := sha256.Sum256(fmt.Appendf(nil, "%s\n%d\n%d", seed, index, attempt))
		id := fmt.Sprintf("F-%x", digest[:8])
		if !used[id] {
			used[id] = true
			return id
		}
	}
}
