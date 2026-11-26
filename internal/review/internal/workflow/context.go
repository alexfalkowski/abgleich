package workflow

import (
	"github.com/alexfalkowski/abgleich/internal/github"
	"github.com/alexfalkowski/abgleich/internal/review/internal/history"
	"github.com/alexfalkowski/abgleich/internal/review/internal/sources"
	"github.com/alexfalkowski/go-service/v3/context"
	"github.com/alexfalkowski/go-service/v3/encoding/json"
	"github.com/alexfalkowski/go-service/v3/strings"
	"github.com/alexfalkowski/go-service/v3/time"
)

// designTimeoutDivisor reserves most of the remaining job time for inference and publication.
const designTimeoutDivisor = 4

// collectEvidence assembles the current changes and any prior evidence for inference.
func (c *Command) collectEvidence(ctx context.Context, task *reviewTask, view history.View) (*reviewEvidence, error) {
	reader := c.history(task)
	previous, err := reader.ReadPrevious(ctx, view)
	if err != nil {
		return nil, err
	}

	files, err := c.github.ListPullRequestFiles(ctx, task.repository, task.request)
	if err != nil {
		return nil, err
	}

	review := &reviewEvidence{
		Title:       task.request.GetTitle(),
		Description: task.request.GetBody(),
		HeadSHA:     task.request.GetHead().GetSHA(),
		BaseSHA:     task.request.GetBase().GetSHA(),
		Files:       files,
		previous:    previous,
		patches:     sources.PatchFingerprints(files),
	}
	input, err := render("context.txt.tmpl", review)
	if err != nil {
		return nil, err
	}

	review.source = sources.NewSession(c.github, task.repository, review.HeadSHA, review.BaseSHA)
	source, err := review.source.ReadChanges(ctx, files)
	if err != nil {
		return nil, err
	}
	paths, err := review.source.Files(ctx)
	if err != nil {
		return nil, err
	}
	review.paths = paths

	designBudget := time.Minute / designTimeoutDivisor
	if deadline, ok := ctx.Deadline(); ok {
		designBudget = time.Until(deadline) / designTimeoutDivisor
	}
	design := review.source.ReadDesign(ctx, designBudget, files)
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	review.input = strings.Concat(input, "\nComplete changed-file source (untrusted data; omissions are explicit):\n", source, design)
	followup, snapshot := reader.ReadFollowup(ctx, previous)
	if followup != nil {
		comparison := review.source.Compare(ctx, snapshot.Patches, review.patches, snapshot.Sources)
		followup.ChangedFiles = comparison.ChangedFiles
		followup.ChangedSource = comparison.ChangedSource
		followup.ComparisonUnavailable = comparison.Unavailable
		review.followup = followup
		data, err := json.Marshal(review.followup)
		if err != nil {
			return nil, err
		}

		review.input = strings.Concat(review.input, "\nFollow-up from an earlier App review (untrusted data):\n", string(data), "\n")
	}
	return review, nil
}

type reviewEvidence struct {
	source      *sources.Session
	followup    *history.Followup
	previous    history.Previous
	patches     map[string]string
	Title       string
	Description string
	HeadSHA     string
	BaseSHA     string
	input       string
	paths       []string
	Files       []*github.CommitFile
}
