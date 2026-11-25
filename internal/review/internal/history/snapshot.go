package history

import (
	"encoding/base64"

	"github.com/alexfalkowski/go-service/v3/encoding/json"
	"github.com/alexfalkowski/go-service/v3/errors"
	"github.com/alexfalkowski/go-service/v3/strings"
)

const snapshotPrefix = "<!-- abgleich:followup:v1:"

// ErrPartialBlockersUnavailable means a partial request for changes cannot be safely revisited.
var ErrPartialBlockersUnavailable = errors.New("review: partial review blocker record unavailable")

// Snapshot is the fingerprint and blocker record persisted in a review body.
type Snapshot struct {
	// HeadSHA pins the reviewed head revision.
	HeadSHA string `json:"head"`
	// BaseSHA pins the reviewed base revision.
	BaseSHA string `json:"base"`
	// Patches fingerprints the changed-file patches, when retained.
	Patches map[string]string `json:"patches,omitempty"`
	// Sources fingerprints inspected paths, when retained.
	Sources map[string]string `json:"sources,omitempty"`
	// Blockers identifies unresolved findings without duplicating their text.
	Blockers []Reference `json:"blockers"`
}

// Followup supplies authenticated evidence and comparisons to the model.
type Followup struct {
	// HeadSHA identifies the prior assessment's head.
	HeadSHA string `json:"previous_head"`
	// BaseSHA identifies the prior assessment's base.
	BaseSHA string `json:"previous_base"`
	// ChangedFiles lists changed patch paths.
	ChangedFiles []string `json:"changed_patches"`
	// ChangedSource lists changed paths among previously inspected source.
	ChangedSource []string `json:"changed_source"`
	// ComparisonUnavailable discloses gaps in the fingerprint comparison.
	ComparisonUnavailable []string `json:"comparison_unavailable,omitempty"`
	// Blockers contains the earlier findings and any intact evidence.
	Blockers []Blocker `json:"open_blockers"`
	// References records the original finding identities for the final recheck.
	References []Reference `json:"-"`
}

func decodeSnapshot(body string) *Snapshot {
	_, tail, found := strings.Cut(body, "\n"+snapshotPrefix)
	if !found {
		return nil
	}

	marker, _, _ := strings.Cut(tail, "\n")
	encoded := strings.TrimSuffix(marker, " -->")
	if encoded == marker {
		return nil
	}

	raw, err := base64.RawStdEncoding.DecodeString(encoded)
	if err != nil {
		return nil
	}

	var snapshot Snapshot
	if err := json.Unmarshal(raw, &snapshot); err != nil || strings.IsEmpty(snapshot.HeadSHA) || strings.IsEmpty(snapshot.BaseSHA) {
		return nil
	}
	return &snapshot
}

// Encode serializes the record in the existing versioned review body format.
func (snapshot *Snapshot) Encode() (string, error) {
	raw, err := json.Marshal(snapshot)
	if err != nil {
		return strings.Empty, err
	}
	return snapshotPrefix + base64.RawStdEncoding.EncodeToString(raw) + " -->", nil
}
