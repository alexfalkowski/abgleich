package automerge

import (
	"github.com/alexfalkowski/abgleich/internal/automerge/internal/strategy"
	"github.com/alexfalkowski/abgleich/internal/commands"
	"github.com/alexfalkowski/abgleich/internal/github"
	"github.com/alexfalkowski/go-service/v2/context"
	"github.com/alexfalkowski/go-service/v2/time"
)

const autoMergeCommand = "automerge"

var _ commands.Handler = (*Command)(nil)

// NewCommand constructs the automerge comment command's authorization and action.
func NewCommand(service github.Service) *Command {
	return &Command{github: service}
}

// Command requests the automerge label and leaves merge decisions to scheduled
// reconciliation. Remote PR history handles stale requests across restarts.
type Command struct {
	github github.Service
}

// Handle re-reads the comment, permissions, and PR state before requesting a
// label. It returns true for accepted or already-enabled automation. The shared
// dispatcher owns command selection, installation validation, and feedback.
func (c *Command) Handle(ctx context.Context, event *github.IssueCommentEvent) (bool, error) {
	repo, err := c.github.GetRepository(ctx, event.GetRepo())
	if err != nil {
		return false, err
	}

	pull, err := c.github.GetPullRequest(ctx, repo, event.GetIssue().GetNumber())
	if err != nil {
		return false, err
	}

	if !c.eligibleForCommand(repo, pull) {
		return false, nil
	}

	comment, err := c.github.GetComment(ctx, repo, pull.GetNumber(), event.GetComment())
	if err != nil {
		return false, err
	}

	return c.authorize(ctx, repo, pull, comment)
}

func (c *Command) authorize(ctx context.Context, repo *github.Repository, pull *github.PullRequest, comment *github.IssueComment) (bool, error) {
	if commands.Parse(comment.GetBody()) != autoMergeCommand || comment.GetUser().GetType() != "User" ||
		!github.HasCommentMetadata(comment) ||
		comment.GetUpdatedAt().After(comment.GetCreatedAt().Time) {
		return false, nil
	}

	allowed, err := c.github.HasWritePermission(ctx, repo, comment.GetUser())
	if err != nil {
		return false, err
	}

	if !allowed {
		return false, commands.ErrWriteAccessRequired
	}

	if automationAlreadyRequested(pull) {
		return true, nil
	}

	events, err := c.github.ListPullRequestAutomationEvents(ctx, pull)
	if err != nil {
		return false, err
	}

	if !commandAuthorizesAutoMerge(events, comment.GetCreatedAt().Time) {
		return false, nil
	}

	return c.addLabel(ctx, repo, pull)
}

func (c *Command) addLabel(ctx context.Context, repo *github.Repository, pull *github.PullRequest) (bool, error) {
	currentRepo, err := c.github.GetRepository(ctx, repo)
	if err != nil {
		return false, err
	}

	current, err := c.github.GetPullRequest(ctx, currentRepo, pull.GetNumber())
	if err != nil {
		return false, err
	}

	if !c.eligibleForCommand(currentRepo, current) {
		return false, nil
	}

	if !c.github.IsSamePullRequest(current, pull) {
		return false, commands.ErrPullRequestChanged
	}

	if automationAlreadyRequested(current) {
		return true, nil
	}

	if err := ctx.Err(); err != nil {
		return false, err
	}

	if err := c.github.AddLabel(ctx, currentRepo, current.GetNumber(), strategy.AutoMergeLabel); err != nil {
		return false, err
	}

	return true, nil
}

func (c *Command) eligibleForCommand(repo *github.Repository, pull *github.PullRequest) bool {
	return eligibleRepository(repo) && repo.GetAllowSquashMerge() && strategy.Eligible(pull) &&
		c.github.IsBaseRepository(repo, pull) && github.HasPullRequestNodeID(pull)
}

func automationAlreadyRequested(pull *github.PullRequest) bool {
	return pull.GetAutoMerge() != nil || github.HasLabel(pull.GetLabels(), strategy.AutoMergeLabel)
}

// commandAuthorizesAutoMerge requires the original command to be strictly newer
// than every label removal and auto-merge disablement. Equal timestamps defer.
func commandAuthorizesAutoMerge(events []*github.IssueEvent, requestedAt time.Time) bool {
	_, removedAt, disabledAt := github.LatestAutomationEventTimes(events, strategy.AutoMergeLabel)

	return !requestedAt.IsZero() && requestedAt.After(removedAt) && requestedAt.After(disabledAt)
}
