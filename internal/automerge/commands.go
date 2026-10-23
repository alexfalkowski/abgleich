package automerge

import (
	"github.com/alexfalkowski/abgleich/internal/automerge/internal/strategy"
	"github.com/alexfalkowski/abgleich/internal/github"
	"github.com/alexfalkowski/go-service/v2/context"
	"github.com/alexfalkowski/go-service/v2/strings"
	"github.com/alexfalkowski/go-service/v2/time"
	"github.com/alexfalkowski/go-sync"
)

const (
	autoMergeCommand   = "@abgleich automerge"
	maxReactionTimeout = time.Second

	// Limit each reaction to a quarter of the remaining command time, leaving
	// most of the deadline for permission checks and labeling.
	reactionTimeoutDivisor = 4
)

// NewCommands constructs synchronous PR comment handling. Concurrent commands
// are rejected while one is running; no work is queued.
func NewCommands(service github.Service) *Commands {
	return &Commands{github: service}
}

// Commands requests labels and leaves merge decisions to scheduled reconciliation.
// Remote PR history, rather than a local delivery cache, handles stale requests.
type Commands struct {
	github github.Service
	mutex  sync.Mutex
}

// Handle processes a verified PR comment within the caller's context. Unrelated
// comments are ignored. Rejections return ErrCommandInProgress,
// ErrWriteAccessRequired, or ErrPullRequestChanged for the transport to classify.
func (c *Commands) Handle(ctx context.Context, event *github.IssueCommentEvent) error {
	if strings.TrimSpace(event.GetComment().GetBody()) != autoMergeCommand {
		return nil
	}

	if !c.mutex.TryLock() {
		return ErrCommandInProgress
	}
	defer c.mutex.Unlock()

	if err := c.github.ValidateInstallation(ctx); err != nil {
		return err
	}

	c.react(ctx, event, "eyes")

	accepted, err := c.requestLabel(ctx, event)
	reaction := "confused"
	if accepted {
		reaction = "+1"
	}
	c.react(ctx, event, reaction)

	return err
}

// Feedback is best effort and bounded by the reaction limit and command deadline.
func (c *Commands) react(ctx context.Context, event *github.IssueCommentEvent, content string) {
	if ctx.Err() != nil {
		return
	}

	timeout := maxReactionTimeout
	if deadline, ok := ctx.Deadline(); ok {
		timeout = min(timeout, time.Until(deadline)/reactionTimeoutDivisor)
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// The service logs feedback failures; they do not change command outcomes.
	_ = c.github.AddCommentReaction(ctx, event.GetRepo(), event.GetComment(), content)
}

func (c *Commands) requestLabel(ctx context.Context, event *github.IssueCommentEvent) (bool, error) {
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

func (c *Commands) authorize(ctx context.Context, repo *github.Repository, pull *github.PullRequest, comment *github.IssueComment) (bool, error) {
	if strings.TrimSpace(comment.GetBody()) != autoMergeCommand || comment.GetUser().GetType() != "User" ||
		!github.HasCommentMetadata(comment) ||
		comment.GetUpdatedAt().After(comment.GetCreatedAt().Time) {
		return false, nil
	}

	allowed, err := c.github.HasWritePermission(ctx, repo, comment.GetUser())
	if err != nil {
		return false, err
	}

	if !allowed {
		return false, ErrWriteAccessRequired
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

func (c *Commands) addLabel(ctx context.Context, repo *github.Repository, pull *github.PullRequest) (bool, error) {
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
		return false, ErrPullRequestChanged
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

func (c *Commands) eligibleForCommand(repo *github.Repository, pull *github.PullRequest) bool {
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
