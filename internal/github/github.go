package github

import (
	"github.com/alexfalkowski/go-service/v2/errors"
	"github.com/alexfalkowski/go-service/v2/strings"
	"github.com/google/go-github/v93/github"
)

// Repository aliases [github.Repository] to keep SDK imports in this package.
type Repository = github.Repository

// PullRequest aliases [github.PullRequest] so consumers use the project-owned package.
type PullRequest = github.PullRequest

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

// UpdatePullRequestBranchInput supplies GitHub's branch-update arguments.
// ExpectedHeadOID rejects concurrent head changes; UpdateMethod selects REBASE
// or MERGE.
type UpdatePullRequestBranchInput struct {
	PullRequestID   string `json:"pullRequestId"`
	ExpectedHeadOID string `json:"expectedHeadOid"`
	UpdateMethod    string `json:"updateMethod"`
}

// UpdatePullRequestBranchRequest encodes a branch-update mutation. Query must
// declare $input with the GraphQL type UpdatePullRequestBranchInput.
type UpdatePullRequestBranchRequest struct {
	Query     string `json:"query"`
	Variables struct {
		Input UpdatePullRequestBranchInput `json:"input"`
	} `json:"variables"`
}

// UpdatePullRequestBranchResponse decodes GitHub's branch-update mutation.
// HTTP success can include GraphQL errors or incomplete data; call Validate
// before treating an update as accepted.
type UpdatePullRequestBranchResponse struct {
	Data struct {
		Update struct {
			PullRequest struct {
				ID string `json:"id"`
			} `json:"pullRequest"`
		} `json:"updatePullRequestBranch"`
	} `json:"data"`
	Errors Errors `json:"errors"`
}

// Validate returns joined GraphQL errors or an error when the payload does not
// identify expectedID. A matching payload confirms acceptance, not completion.
func (r *UpdatePullRequestBranchResponse) Validate(expectedID string) error {
	if err := r.Errors.Validate(); err != nil {
		return errors.Prefix("github: rebase", err)
	}

	if r.Data.Update.PullRequest.ID != expectedID {
		return errors.New("github: rebase returned no matching pull request")
	}

	return nil
}

// EnablePullRequestAutoMergeInput selects a PR and merge method for auto-merge,
// guarded by ExpectedHeadOID against concurrent pushes.
type EnablePullRequestAutoMergeInput struct {
	PullRequestID   string `json:"pullRequestId"`
	ExpectedHeadOID string `json:"expectedHeadOid"`
	MergeMethod     string `json:"mergeMethod"`
}

// EnablePullRequestAutoMergeRequest encodes GitHub's auto-merge mutation.
type EnablePullRequestAutoMergeRequest struct {
	Query     string `json:"query"`
	Variables struct {
		Input EnablePullRequestAutoMergeInput `json:"input"`
	} `json:"variables"`
}

// EnablePullRequestAutoMergeResponse requires validation even on HTTP success.
type EnablePullRequestAutoMergeResponse struct {
	Data struct {
		Enable struct {
			PullRequest struct {
				ID string `json:"id"`
			} `json:"pullRequest"`
		} `json:"enablePullRequestAutoMerge"`
	} `json:"data"`
	Errors Errors `json:"errors"`
}

// Validate rejects GraphQL errors and responses identifying a different PR.
func (r *EnablePullRequestAutoMergeResponse) Validate(expectedID string) error {
	if err := r.Errors.Validate(); err != nil {
		return errors.Prefix("github: auto-merge", err)
	}

	if r.Data.Enable.PullRequest.ID != expectedID {
		return errors.New("github: auto-merge returned no matching pull request")
	}

	return nil
}

// PullRequestAutomationEventsRequest identifies a PR by node ID and supplies the cursor
// for the next page of its automation history.
type PullRequestAutomationEventsRequest struct {
	Variables struct {
		ID    string  `json:"id"`
		After *string `json:"after"`
	} `json:"variables"`
	Query string `json:"query"`
}

// PullRequestAutomationEventsResponse decodes timeline history into SDK issue-event snapshots.
// Call Validate before using the history, even when the HTTP request succeeds.
type PullRequestAutomationEventsResponse struct {
	Data struct {
		Node struct {
			ID            string `json:"id"`
			TimelineItems struct {
				Nodes    []*IssueEvent `json:"nodes"`
				PageInfo struct {
					EndCursor   *string `json:"endCursor"`
					HasNextPage *bool   `json:"hasNextPage"`
				} `json:"pageInfo"`
			} `json:"timelineItems"`
		} `json:"node"`
	} `json:"data"`
	Errors Errors `json:"errors"`
}

// Validate rejects GraphQL errors, a PR node ID different from expectedID,
// incomplete pages, and events that cannot establish valid automation history.
func (r *PullRequestAutomationEventsResponse) Validate(expectedID string) error {
	if err := r.Errors.Validate(); err != nil {
		return errors.Prefix("github: automation events", err)
	}

	node := r.Data.Node
	if node.ID != expectedID {
		return errors.New("github: automation events returned no matching pull request")
	}

	if node.TimelineItems.Nodes == nil || node.TimelineItems.PageInfo.HasNextPage == nil {
		return errors.New("github: automation events returned an incomplete page")
	}

	return validateAutomationEvents(node.TimelineItems.Nodes)
}

// Errors contains GraphQL failures that may accompany data even on HTTP success.
type Errors []struct {
	Message string `json:"message"`
	Type    string `json:"type"`
}

// Validate joins reported failures and includes [ErrRateLimited] for rate limits.
// It returns nil when Errors is empty or nil.
func (e Errors) Validate() error {
	var failures error
	for _, failure := range e {
		failures = errors.Join(failures, errors.New("github: "+failure.Message))

		if failure.Type == "RATE_LIMITED" || rateLimitMessage(failure.Message) {
			failures = errors.Join(failures, ErrRateLimited)
		}
	}

	return failures
}

// validateAutomationEvents requires timestamps, supported event types, and label
// names for label changes so incomplete history cannot authorize automation.
func validateAutomationEvents(events []*IssueEvent) error {
	for _, event := range events {
		if event.GetCreatedAt().IsZero() {
			return errors.New("github: automation event is missing its timestamp")
		}

		switch event.GetEvent() {
		case "LabeledEvent", "UnlabeledEvent":
			if strings.IsEmpty(event.GetLabel().GetName()) {
				return errors.New("github: automation event is missing its label")
			}
		case "AutoMergeDisabledEvent":
		default:
			return errors.New("github: automation event has an unexpected type")
		}
	}

	return nil
}
