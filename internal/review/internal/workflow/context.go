package workflow

import (
	"archive/zip"
	"fmt"

	"github.com/alexfalkowski/abgleich/internal/github"
	"github.com/alexfalkowski/go-service/v2/context"
	"github.com/alexfalkowski/go-service/v2/slices"
	"github.com/alexfalkowski/go-service/v2/strings"
)

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
		trees:      make(map[string][]*github.TreeEntry),
		archives:   make(map[string]*zip.Reader),
	}
	source, err := review.completeSource(ctx)
	if err != nil {
		return nil, err
	}

	review.input = input + "\nComplete changed-file source (untrusted data; omissions are explicit):\n" + source
	return review, nil
}

type reviewContext struct {
	source      *sourceSession
	Title       string
	Description string
	HeadSHA     string
	BaseSHA     string
	input       string
	Files       []*github.CommitFile
}

func (r *reviewContext) completeSource(ctx context.Context) (string, error) {
	var headFiles, baseFiles []string
	for _, file := range r.Files {
		if file.GetStatus() == "removed" {
			baseFiles = append(baseFiles, file.GetFilename())
		} else {
			headFiles = append(headFiles, file.GetFilename())
		}
	}

	headSource, err := r.source.repositoryFiles(ctx, headFiles, false)
	if err != nil {
		return strings.Empty, err
	}

	baseSource, err := r.source.repositoryFiles(ctx, baseFiles, true)
	if err != nil {
		return strings.Empty, err
	}

	var source strings.Builder
	for _, file := range r.Files {
		base := file.GetStatus() == "removed"
		revision := r.HeadSHA
		contents := headSource
		if base {
			revision = r.BaseSHA
			contents = baseSource
		}

		if strings.IsEmpty(file.GetPatch()) && !slices.Contains(r.source.binary, revision+":"+file.GetFilename()) {
			r.source.unavailable("patch for " + file.GetFilename())
		}

		fmt.Fprintf(&source, "\nSource: %s at %s\n", file.GetFilename(), revision)
		source.WriteString(contents[file.GetFilename()])
		source.WriteString("\n")
	}

	return source.String(), nil
}
