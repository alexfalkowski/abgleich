package summarise

import (
	"html"
	"net/url"

	"github.com/alexfalkowski/go-service/v2/slices"
	"github.com/alexfalkowski/go-service/v2/strings"
)

const (
	maxSummaryChanges   = 3
	maxSummaryLocations = 3
)

func reply(task *summaryTask, text string, files []changedFile) (string, error) {
	text, limits, _ := strings.Cut(text, "\nLimits:")
	text, locations := summaryLocations(text, files, task.repository.GetFullName(), task.pullRequest.GetHead().GetSHA())
	text, changes := summaryChanges(text)

	return render("summary.md.tmpl", summaryComment{
		HeadSHA:          task.pullRequest.GetHead().GetSHA(),
		CommandCommentID: task.commandComment.GetID(),
		Text:             text,
		Changes:          changes,
		Locations:        locations,
		Limits:           strings.TrimSpace(limits),
	})
}

func summaryChanges(text string) (string, []string) {
	intro, section, found := strings.Cut(strings.TrimSpace(text), "\nChanges:\n")
	if !found {
		return text, nil
	}

	changes := slices.Collect(strings.SplitSeq(strings.TrimSpace(section), "\n"))
	if len(changes) > maxSummaryChanges {
		return text, nil
	}
	for i, change := range changes {
		changes[i] = strings.TrimSpace(change)
		if strings.IsEmpty(changes[i]) {
			return text, nil
		}
	}

	return strings.TrimSpace(intro), changes
}

func summaryLocations(text string, files []changedFile, repository, sha string) (string, []summaryLocation) {
	intro, section, found := strings.Cut(strings.TrimSpace(text), "Where to look:")
	if !found {
		return text, nil
	}

	available := make(map[string]struct{}, len(files))
	for _, file := range files {
		available[file.Filename] = struct{}{}
	}

	lines := slices.Collect(strings.SplitSeq(strings.TrimSpace(section), "\n"))
	if len(lines) == 0 || len(lines) > maxSummaryLocations {
		return text, nil
	}

	locations := make([]summaryLocation, 0, len(lines))
	for _, line := range lines {
		path, description, ok := strings.Cut(strings.TrimSpace(line), ": ")
		if !ok || strings.IsEmpty(path) || strings.IsEmpty(strings.TrimSpace(description)) {
			return text, nil
		}
		if _, ok := available[path]; !ok {
			return text, nil
		}

		link := url.URL{
			Scheme: "https",
			Host:   "github.com",
			Path:   "/" + repository + "/blob/" + sha + "/" + path,
		}
		linkURL := strings.NewReplacer("(", "%28", ")", "%29").Replace(link.String())
		locations = append(locations, summaryLocation{
			Path:        path,
			URL:         linkURL,
			Description: strings.TrimSpace(description),
		})
	}

	return strings.TrimSpace(intro), locations
}

// Markdown syntax and GitHub mentions need escaping in addition to HTML text.
func markdownText(text string) string {
	return html.EscapeString(strings.NewReplacer(
		"@", "@\u200b",
		"\\", "\\\\",
		"`", "\\`",
		"*", "\\*",
		"_", "\\_",
		"[", "\\[",
		"]", "\\]",
		"(", "\\(",
		")", "\\)",
		"#", "\\#",
		"!", "\\!",
		"|", "\\|",
		"~", "\\~",
		"-", "\\-",
		"+", "\\+",
		"=", "\\=",
	).Replace(text))
}
