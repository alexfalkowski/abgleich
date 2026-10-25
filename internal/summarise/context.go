package summarise

import "github.com/alexfalkowski/go-service/v2/context"

const (
	maxFiles             = 20
	maxPatchBytes        = 8 * 1024
	maxPatchContextBytes = 16 * 1024
)

// context uses the pull-request diff directly. Complete files, repository trees,
// and old-side snapshots are not needed for an advisory summary.
func (c *Command) context(ctx context.Context, task *summaryTask) (string, error) {
	files, err := c.github.ListPullRequestFiles(ctx, task.repository, task.pullRequest)
	if err != nil {
		return "", err
	}

	diff := pullRequestDiff{
		Title:            task.pullRequest.GetTitle(),
		Description:      task.pullRequest.GetBody(),
		HeadSHA:          task.pullRequest.GetHead().GetSHA(),
		BaseSHA:          task.pullRequest.GetBase().GetSHA(),
		OmittedFileCount: max(0, len(files)-maxFiles),
	}
	patchBytes := 0

	for _, file := range files[:min(len(files), maxFiles)] {
		change := changedFile{
			Filename:         file.GetFilename(),
			PreviousFilename: file.GetPreviousFilename(),
			Status:           file.GetStatus(),
		}

		patch := file.GetPatch()
		if len(patch) <= maxPatchBytes && patchBytes+len(patch) <= maxPatchContextBytes {
			change.Patch = patch
			patchBytes += len(patch)
		}
		diff.Files = append(diff.Files, change)
	}

	return render("context.txt.tmpl", diff)
}
