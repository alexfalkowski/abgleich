package summarise

import (
	"github.com/alexfalkowski/go-service/v2/context"
	"github.com/alexfalkowski/go-service/v2/strings"
)

// context uses the pull-request diff directly. Complete files, repository trees,
// and old-side snapshots are not needed for an advisory summary.
func (c *Command) context(ctx context.Context, task *summaryTask) (string, []changedFile, error) {
	files, err := c.github.ListPullRequestFiles(ctx, task.repository, task.pullRequest)
	if err != nil {
		return "", nil, err
	}

	diff := pullRequestDiff{
		Title:       task.pullRequest.GetTitle(),
		Description: task.pullRequest.GetBody(),
		HeadSHA:     task.pullRequest.GetHead().GetSHA(),
		BaseSHA:     task.pullRequest.GetBase().GetSHA(),
	}
	available := make([]changedFile, 0, len(files))

	for _, file := range files {
		change := changedFile{
			Filename:         file.GetFilename(),
			PreviousFilename: file.GetPreviousFilename(),
			Status:           file.GetStatus(),
			Patch:            file.GetPatch(),
		}

		diff.Files = append(diff.Files, change)
		if !strings.IsEmpty(change.Patch) && change.Status != "removed" {
			available = append(available, change)
		}
	}

	input, err := render("context.txt.tmpl", diff)
	return input, available, err
}
