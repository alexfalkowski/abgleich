package summarise

import "github.com/alexfalkowski/abgleich/internal/github"

// summaryTask groups a command comment with repository and PR snapshots for
// authorisation and revision checks throughout queued summary generation.
type summaryTask struct {
	repository     *github.Repository
	pullRequest    *github.PullRequest
	commandComment *github.IssueComment
}

// pullRequestDiff supplies PR metadata and selected patches to the context
// template. OmittedFileCount records changed files outside that selection.
type pullRequestDiff struct {
	Title            string
	Description      string
	HeadSHA          string
	BaseSHA          string
	Files            []changedFile
	OmittedFileCount int
}

// changedFile retains file identity and status when its patch is unavailable or
// excluded by a size budget. An empty Patch renders an explicit omission.
type changedFile struct {
	Filename         string
	PreviousFilename string
	Status           string
	Patch            string
}

// summaryComment ties generated text to the analysed head and command comment.
// The reply template escapes Text and links CommandCommentID to the request.
type summaryComment struct {
	HeadSHA          string
	Text             string
	CommandCommentID int64
}
