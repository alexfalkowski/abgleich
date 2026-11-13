package ruby

import (
	"regexp"

	"github.com/alexfalkowski/go-service/v3/errors"
	"github.com/alexfalkowski/go-service/v3/strings"
)

var (
	gemName     = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]*$`)
	gemVersion  = regexp.MustCompile(`^[0-9]+(\.[A-Za-z0-9]+)*$`)
	gemChecksum = regexp.MustCompile(`^[a-fA-F0-9]{64}$`)
)

type lockedGem struct {
	name     string
	version  string
	platform string
	checksum string
}

// readLockedGem reads only the source blocks and checksum for the requested artifact.
// Unsupported sources and ambiguous entries are rejected rather than resolved like Bundler.
func readLockedGem(content, name, platform string) (*lockedGem, error) {
	section, remote, locked := strings.Empty, strings.Empty, strings.Empty
	remotes := 0

	for line := range strings.SplitSeq(content, "\n") {
		line = strings.TrimSuffix(line, "\r")
		if line == "CONTENT ADDRESSES" {
			return nil, errors.New("review: unsupported Ruby lockfile format")
		}

		if strings.TrimSpace(line) == line {
			section, remote, remotes = line, strings.Empty, 0
			continue
		}

		if strings.HasPrefix(line, "  remote:") {
			remote = strings.TrimRight(strings.TrimSpace(strings.TrimPrefix(line, "  remote:")), "/")
			remotes++
		}

		version := lockedGemVersion(line, name, platform)
		if strings.IsEmpty(version) {
			continue
		}

		if !publicGemSource(section, remote, remotes) {
			return nil, errors.New("review: Ruby dependency requires repository source or an unsupported registry")
		}

		if !strings.IsEmpty(locked) {
			return nil, errors.New("review: ambiguous locked Ruby dependency")
		}

		locked = version
	}

	if strings.IsEmpty(locked) {
		return nil, errors.New("review: Ruby dependency or platform is not locked")
	}

	artifact := locked
	if platform != "ruby" {
		artifact += "-" + platform
	}

	checksum, err := lockedGemChecksum(content, name+" ("+artifact+")")
	return &lockedGem{name: name, version: locked, platform: platform, checksum: checksum}, err
}

func publicGemSource(section, remote string, count int) bool {
	return section == "GEM" && remote == "https://rubygems.org" && count == 1
}

func lockedGemVersion(line, name, platform string) string {
	prefix := "    " + name + " ("
	if !strings.HasPrefix(line, prefix) || !strings.HasSuffix(line, ")") {
		return strings.Empty
	}

	version := strings.TrimSuffix(strings.TrimPrefix(line, prefix), ")")
	if platform != "ruby" {
		if !strings.HasSuffix(version, "-"+platform) {
			return strings.Empty
		}

		version = strings.TrimSuffix(version, "-"+platform)
	}

	if !gemVersion.MatchString(version) {
		return strings.Empty
	}

	return version
}

func lockedGemChecksum(content, artifact string) (string, error) {
	section := strings.Empty
	seen := false
	checksum := strings.Empty

	for line := range strings.SplitSeq(content, "\n") {
		line = strings.TrimSuffix(line, "\r")
		if !strings.IsEmpty(line) && strings.TrimSpace(line) == line {
			section = line
		}

		prefix := "  " + artifact
		if section != "CHECKSUMS" || (line != prefix && !strings.HasPrefix(line, prefix+" ")) {
			continue
		}

		if seen {
			return strings.Empty, errors.New("review: ambiguous Ruby checksum")
		}

		seen = true

		var err error
		checksum, err = parseGemChecksum(strings.TrimSpace(strings.TrimPrefix(line, prefix)))
		if err != nil {
			return strings.Empty, err
		}
	}

	return checksum, nil
}

func parseGemChecksum(content string) (string, error) {
	checksum := strings.Empty

	for field := range strings.SplitSeq(content, ",") {
		if field = strings.TrimSpace(field); strings.HasPrefix(field, "sha256=") {
			digest := strings.TrimPrefix(field, "sha256=")
			if !gemChecksum.MatchString(digest) || !strings.IsEmpty(checksum) {
				return strings.Empty, errors.New("review: invalid Ruby checksum")
			}

			checksum = strings.ToLower(digest)
		}
	}

	return checksum, nil
}
