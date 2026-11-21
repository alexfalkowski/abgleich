package sources

import (
	"crypto/sha256"
	"fmt"
	"strings"

	"github.com/alexfalkowski/abgleich/internal/github"
	"github.com/alexfalkowski/go-service/v3/context"
	"github.com/alexfalkowski/go-service/v3/slices"
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

// Compare compares previously reviewed patches and source on the same side of
// the pull request. Unreadable trees are reported as comparison limitations.
func (s *Session) Compare(ctx context.Context, previousPatches, currentPatches, previousSource map[string]string) Comparison {
	if previousPatches == nil {
		return Comparison{Unavailable: []string{"Previous patch and source fingerprints were unavailable."}}
	}
	comparison := Comparison{ChangedFiles: changedPatches(previousPatches, currentPatches)}

	// A path is compared on the same side of the PR, even when that side moved.
	for _, source := range []struct{ side, revision string }{{"head", s.headSHA}, {"base", s.baseSHA}} {
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
