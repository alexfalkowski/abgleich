package assessment

import (
	"crypto/sha256"
	"fmt"
	"maps"
	"strings"

	"github.com/alexfalkowski/abgleich/internal/review/internal/history"
	"github.com/alexfalkowski/go-service/v3/slices"
)

func (i Input) snapshot(result *reviewResult, partial bool) *history.Snapshot {
	snapshot := &history.Snapshot{HeadSHA: i.HeadSHA, BaseSHA: i.BaseSHA, Patches: i.Patches, Sources: i.Source.Fingerprints}
	previous := make(map[string]history.Reference)
	if i.Followup != nil {
		for _, ref := range i.Followup.References {
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

func (r *reviewReply) reconcileFindings(result *reviewResult, followup *history.Followup, seed string) {
	previous := make(map[string]int64)
	if followup != nil {
		for _, blocker := range followup.Blockers {
			previous[blocker.ID] = blocker.ReviewID
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
