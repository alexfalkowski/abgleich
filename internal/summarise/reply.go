package summarise

import (
	"html"
	"net/url"
	"strings"
)

const maxSummaryLocations = 3

func reply(task *summaryTask, text string, files []changedFile) (string, error) {
	text, locations := summaryLocations(text, files, task.repository.GetFullName(), task.pullRequest.GetHead().GetSHA())

	return render("summary.md.tmpl", summaryComment{
		HeadSHA:          task.pullRequest.GetHead().GetSHA(),
		CommandCommentID: task.commandComment.GetID(),
		Text:             text,
		Locations:        locations,
	})
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

	lines := strings.Split(strings.TrimSpace(section), "\n")
	if len(lines) == 0 || len(lines) > maxSummaryLocations {
		return text, nil
	}

	locations := make([]summaryLocation, 0, len(lines))
	for _, line := range lines {
		path, description, ok := strings.Cut(strings.TrimSpace(line), ": ")
		if !ok || path == "" || strings.TrimSpace(description) == "" {
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
