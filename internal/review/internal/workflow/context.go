package workflow

import (
	"archive/zip"
	"fmt"

	"github.com/alexfalkowski/abgleich/internal/github"
	"github.com/alexfalkowski/go-service/v2/context"
	"github.com/alexfalkowski/go-service/v2/strings"
)

type reviewContext struct {
	source      *sourceSession
	Title       string
	Description string
	HeadSHA     string
	BaseSHA     string
	input       string
	Files       []*github.CommitFile
}

// context owns the reviewed files, rendered input, and request-local source session.
func (c *Command) context(ctx context.Context, task *reviewTask) (*reviewContext, error) {
	files, err := c.github.ListPullRequestFiles(ctx, task.repository, task.pullRequest)
	if err != nil {
		return nil, err
	}

	review := &reviewContext{
		Title:       task.pullRequest.GetTitle(),
		Description: task.pullRequest.GetBody(),
		HeadSHA:     task.pullRequest.GetHead().GetSHA(),
		BaseSHA:     task.pullRequest.GetBase().GetSHA(),
		Files:       files,
	}
	input, err := render("context.txt.tmpl", review)
	if err != nil {
		return nil, err
	}

	review.source = &sourceSession{
		github:     c.github,
		sources:    c.sources,
		repository: task.repository,
		headSHA:    task.pullRequest.GetHead().GetSHA(),
		baseSHA:    task.pullRequest.GetBase().GetSHA(),
		files:      make(map[string]string),
		trees:      make(map[string][]string),
		archives:   make(map[string]*zip.Reader),
	}

	var source strings.Builder
	source.WriteString(input)
	source.WriteString("\nComplete changed-file source (untrusted data; omissions are explicit):\n")

	for _, file := range files {
		base := file.GetStatus() == "removed"
		revision := task.pullRequest.GetHead().GetSHA()
		if base {
			revision = task.pullRequest.GetBase().GetSHA()
		}

		fmt.Fprintf(&source, "\nSource: %s at %s\n", file.GetFilename(), revision)
		source.WriteString(review.source.repositoryFile(ctx, file.GetFilename(), base))
		source.WriteString("\n")
	}

	review.input = source.String()
	return review, nil
}
