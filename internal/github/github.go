package github

import "github.com/google/go-github/v93/github"

// UserBot identifies a bot account in GitHub author metadata.
const UserBot = "Bot"

// UserHuman identifies a human account in GitHub author metadata.
const UserHuman = "User"

// Repository aliases [github.Repository] to keep SDK imports in this package.
type Repository = github.Repository

// Branch aliases [github.Branch] for branch cleanup snapshots.
type Branch = github.Branch

// RepositoryCommit aliases [github.RepositoryCommit] for commit age checks.
type RepositoryCommit = github.RepositoryCommit

// TreeEntry aliases [github.TreeEntry] to retain path and blob identity from a source tree.
type TreeEntry = github.TreeEntry

// RepositoryContent aliases [github.RepositoryContent] for source blob metadata.
type RepositoryContent = github.RepositoryContent

// PullRequest aliases [github.PullRequest] so consumers use the project-owned package.
type PullRequest = github.PullRequest

// PullRequestReview aliases [github.PullRequestReview] for native review history.
type PullRequestReview = github.PullRequestReview

// PullRequestReviewRequest aliases [github.PullRequestReviewRequest] for review publication.
type PullRequestReviewRequest = github.PullRequestReviewRequest

// DraftReviewComment aliases [github.DraftReviewComment] for inline findings.
type DraftReviewComment = github.DraftReviewComment

// Label aliases [github.Label] for shared label membership checks.
type Label = github.Label

// IssueEvent aliases [github.IssueEvent] for pull request automation history.
type IssueEvent = github.IssueEvent

// IssueComment aliases [github.IssueComment] for signed command requests.
type IssueComment = github.IssueComment

// IssueCommentEvent aliases [github.IssueCommentEvent] for verified deliveries.
type IssueCommentEvent = github.IssueCommentEvent

// User aliases [github.User] for current command-author permission checks.
type User = github.User

// CommitsComparison aliases [github.CommitsComparison] so consumers use the project-owned package.
type CommitsComparison = github.CommitsComparison

// CommitFile aliases [github.CommitFile] for pull request file snapshots.
type CommitFile = github.CommitFile

// CheckRun aliases [github.CheckRun] for current-head CI observations.
type CheckRun = github.CheckRun

// CommitStatus aliases [github.RepoStatus] for commit-status contexts.
type CommitStatus = github.RepoStatus
