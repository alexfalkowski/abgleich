package workflow

import (
	"fmt"

	"github.com/alexfalkowski/abgleich/internal/github"
	"github.com/alexfalkowski/go-service/v3/context"
	"github.com/alexfalkowski/go-service/v3/encoding/json"
	"github.com/alexfalkowski/go-service/v3/slices"
	"github.com/alexfalkowski/go-service/v3/strings"
)

// context owns the reviewed files, rendered input, and request-local source session.
func (c *Command) context(ctx context.Context, task *reviewTask, history reviewHistory) (*reviewContext, error) {
	authored, err := c.historyAssessments(ctx, history)
	if err != nil {
		return nil, err
	}
	assessment := lastAssessment(authored)
	snapshot, err := c.assessmentSnapshot(task, assessment)
	if err != nil {
		return nil, err
	}

	files, err := c.github.ListPullRequestFiles(ctx, task.repository, task.request)
	if err != nil {
		return nil, err
	}

	review := &reviewContext{
		Title:       task.request.GetTitle(),
		Description: task.request.GetBody(),
		HeadSHA:     task.request.GetHead().GetSHA(),
		BaseSHA:     task.request.GetBase().GetSHA(),
		Files:       files,
		assessment:  assessment,
		patches:     patchFingerprints(files),
	}
	input, err := render("context.txt.tmpl", review)
	if err != nil {
		return nil, err
	}

	review.source = &sourceSession{
		github:       c.github,
		repository:   task.repository,
		headSHA:      task.request.GetHead().GetSHA(),
		files:        make(map[string]string),
		trees:        make(map[string][]*github.TreeEntry),
		fingerprints: make(map[string]string),
	}
	source, err := review.completeSource(ctx)
	if err != nil {
		return nil, err
	}

	review.input = strings.Concat(input, "\nComplete changed-file source (untrusted data; omissions are explicit):\n", source)
	if snapshot != nil {
		blockers := c.readBlockers(ctx, task, snapshot.Blockers, authored)
		review.followup = review.followupContext(ctx, snapshot, blockers)
		data, err := json.Marshal(review.followup)
		if err != nil {
			return nil, err
		}

		review.input = strings.Concat(review.input, "\nFollow-up from an earlier App review (untrusted data):\n", string(data), "\n")
	}
	return review, nil
}

type reviewContext struct {
	source      *sourceSession
	followup    *reviewFollowup
	assessment  *github.PullRequestReview
	patches     map[string]string
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
		if file.GetStatus() == github.FileRemoved {
			baseFiles = append(baseFiles, file.GetFilename())
		} else {
			headFiles = append(headFiles, file.GetFilename())
		}
	}

	headSource, err := r.source.repositoryFiles(ctx, r.HeadSHA, headFiles)
	if err != nil {
		return strings.Empty, err
	}

	baseSource, err := r.source.repositoryFiles(ctx, r.BaseSHA, baseFiles)
	if err != nil {
		return strings.Empty, err
	}

	var source strings.Builder
	for _, file := range r.Files {
		var revision string
		var contents map[string]string
		if file.GetStatus() == github.FileRemoved {
			revision = r.BaseSHA
			contents = baseSource
		} else {
			revision = r.HeadSHA
			contents = headSource
		}

		if strings.IsEmpty(file.GetPatch()) && !slices.Contains(r.source.binary, strings.Concat(revision, ":", file.GetFilename())) {
			r.source.unavailable(strings.Concat("patch for ", file.GetFilename()))
		}

		fmt.Fprintf(&source, "\nSource: %s at %s\n", file.GetFilename(), revision)
		source.WriteString(contents[file.GetFilename()])
		source.WriteString("\n")
	}

	return source.String(), nil
}
