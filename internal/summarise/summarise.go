package summarise

import (
	"github.com/alexfalkowski/abgleich/internal/ai"
	"github.com/alexfalkowski/abgleich/internal/github"
)

// summaryTask groups a command comment with repository and PR snapshots for
// authorisation and revision checks throughout queued summary generation.
type summaryTask struct {
	repository     *github.Repository
	pullRequest    *github.PullRequest
	commandComment *github.IssueComment
}

// pullRequestDiff supplies PR metadata and every changed file to the context
// template, including patch text when GitHub provides it.
type pullRequestDiff struct {
	Title       string
	Description string
	HeadSHA     string
	BaseSHA     string
	Files       []changedFile
}

// changedFile retains file identity and status when GitHub provides no patch.
// An empty Patch renders that omission explicitly.
type changedFile struct {
	Filename         string
	PreviousFilename string
	Status           string
	Patch            string
}

// summaryComment ties generated text to the analysed head and command comment.
// The reply template escapes generated text and links CommandCommentID to the request.
type summaryComment struct {
	Completion       *ai.Completion
	HeadSHA          string
	Text             string
	Limits           string
	Changes          []string
	Locations        []summaryLocation
	CommandCommentID int64
}

// summaryLocation links an available changed-file patch to its explanation.
type summaryLocation struct {
	Path        string
	URL         string
	Description string
}
