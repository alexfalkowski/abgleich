package codereview

import (
	"github.com/alexfalkowski/abgleich/internal/github"
	"github.com/alexfalkowski/go-service/v2/context"
)

type reviewContext struct {
	Title       string
	Description string
	HeadSHA     string
	BaseSHA     string
	Files       []*github.CommitFile
}

// context includes every file and available patch returned by GitHub.
func (c *Command) context(ctx context.Context, task *reviewTask) (string, error) {
	files, err := c.github.ListPullRequestFiles(ctx, task.repository, task.pullRequest)
	if err != nil {
		return "", err
	}

	return render("context.txt.tmpl", reviewContext{
		Title:       task.pullRequest.GetTitle(),
		Description: task.pullRequest.GetBody(),
		HeadSHA:     task.pullRequest.GetHead().GetSHA(),
		BaseSHA:     task.pullRequest.GetBase().GetSHA(),
		Files:       files,
	})
}
