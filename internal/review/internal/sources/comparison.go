package sources

import (
	"crypto/sha256"
	"fmt"

	"github.com/alexfalkowski/abgleich/internal/github"
	"github.com/alexfalkowski/go-service/v3/context"
	"github.com/alexfalkowski/go-service/v3/slices"
	"github.com/alexfalkowski/go-service/v3/strings"
)

// Comparison describes changes since the preceding assessment.
type Comparison struct {
	// ChangedFiles lists changed patch paths.
	ChangedFiles []string

	// ChangedSource lists inspected paths that changed on their original PR side.
	ChangedSource []string

	// Unavailable records comparisons that could not be made.
	Unavailable []string
}

// Compare compares current patches and source against the previous assessment's
// fingerprints. Unavailable evidence limits the comparison.
func (s *Session) Compare(ctx context.Context, previousPatches, currentPatches, previousSource map[string]string) Comparison {
	var comparison Comparison
	switch {
	case previousPatches == nil && previousSource == nil:
		comparison.Unavailable = append(comparison.Unavailable, "Previous patch and source fingerprints were unavailable.")
	case previousPatches == nil:
		comparison.Unavailable = append(comparison.Unavailable, "Previous patch fingerprints were unavailable.")
	case previousSource == nil:
		comparison.Unavailable = append(comparison.Unavailable, "Previous source fingerprints were unavailable.")
	}
	if previousPatches != nil {
		comparison.ChangedFiles = changedPatches(previousPatches, currentPatches)
	}

	type revisionComparison struct {
		side     string
		revision string
	}
	head := revisionComparison{side: "head", revision: s.headSHA}
	base := revisionComparison{side: "base", revision: s.baseSHA}

	// A path is compared on the same side of the PR, even when that side moved.
	for _, source := range []revisionComparison{head, base} {
		changed, err := s.changedSourcePaths(ctx, source.revision, source.side+":", previousSource)
		if err != nil {
			comparison.Unavailable = append(comparison.Unavailable, "Could not compare previous source on the "+source.side+" revision.")
			continue
		}
		comparison.ChangedSource = append(comparison.ChangedSource, changed...)
	}
	slices.Sort(comparison.ChangedSource)
	return comparison
}

func (s *Session) changedSourcePaths(ctx context.Context, revision, prefix string, previous map[string]string) ([]string, error) {
	names := make(map[string]string)
	for path, fingerprint := range previous {
		if strings.HasPrefix(path, prefix) {
			names[strings.TrimPrefix(path, prefix)] = fingerprint
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
		fingerprint := sourceFingerprint(entry)
		if names[path] == fingerprint {
			// Keep tracking unchanged related paths across focused follow-ups.
			s.fingerprints[prefix+path] = fingerprint
			delete(names, path)
		}
	}

	changed := make([]string, 0, len(names))
	for path := range names {
		changed = append(changed, prefix+path)
	}
	return changed, nil
}

// PatchFingerprints identifies changed patches across review attempts.
func PatchFingerprints(files []*github.CommitFile) map[string]string {
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
