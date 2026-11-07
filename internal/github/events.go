package github

import (
	"github.com/alexfalkowski/go-service/v2/context"
	"github.com/alexfalkowski/go-service/v2/errors"
	"github.com/alexfalkowski/go-service/v2/strings"
	"github.com/alexfalkowski/go-service/v2/time"
)

const (
	eventLabeled           = "LabeledEvent"
	eventUnlabeled         = "UnlabeledEvent"
	eventAutoMergeDisabled = "AutoMergeDisabledEvent"

	// Alias GraphQL fields onto SDK issue events. AutoMergeDisabledEvent is exposed
	// by GraphQL, so REST issue events alone cannot preserve a disabled auto-merge.
	// https://docs.github.com/en/graphql/reference/pulls#automergedisabledevent
	automationEventsQuery = `query PullRequestAutomationEvents($id: ID!, $after: String) {
  node(id: $id) {
    ... on PullRequest {
      id
      timelineItems(first: 100, after: $after,
        itemTypes: [LABELED_EVENT, UNLABELED_EVENT, AUTO_MERGE_DISABLED_EVENT]) {
        nodes {
          event: __typename
          ... on LabeledEvent { created_at: createdAt label { name } }
          ... on UnlabeledEvent { created_at: createdAt label { name } }
          ... on AutoMergeDisabledEvent { created_at: createdAt }
        }
        pageInfo { endCursor hasNextPage }
      }
    }
  }
}`
)

// LatestAutomationEventTimes returns the latest label application, removal, and
// auto-merge disable timestamps, in that order. Missing events return zero times.
// History must be validated, and label matching is exact. Disabling auto-merge
// is independent of the label; callers decide whether these facts revoke access.
func LatestAutomationEventTimes(events []*IssueEvent, label string) (time.Time, time.Time, time.Time) {
	var labeledAt, removedAt, disabledAt time.Time

	for _, event := range events {
		occurredAt := event.GetCreatedAt().Time

		switch event.GetEvent() {
		case eventLabeled:
			if event.GetLabel().GetName() == label && occurredAt.After(labeledAt) {
				labeledAt = occurredAt
			}

		case eventUnlabeled:
			if event.GetLabel().GetName() == label && occurredAt.After(removedAt) {
				removedAt = occurredAt
			}

		case eventAutoMergeDisabled:
			if occurredAt.After(disabledAt) {
				disabledAt = occurredAt
			}
		}
	}

	return labeledAt, removedAt, disabledAt
}

// ListPullRequestAutomationEvents collects and validates every timeline page
// before returning history. Any invalid page or stalled cursor rejects the history.
func (s *service) ListPullRequestAutomationEvents(ctx context.Context, pull *PullRequest) ([]*IssueEvent, error) {
	body := PullRequestAutomationEventsRequest{Query: automationEventsQuery}
	body.Variables.ID = pull.GetNodeID()

	var events []*IssueEvent

	for {
		var response PullRequestAutomationEventsResponse
		if err := s.graphql(ctx, body, &response); err != nil {
			return nil, err
		}
		if err := response.Validate(pull.GetNodeID()); err != nil {
			return nil, err
		}

		page := response.Data.Node.TimelineItems
		events = append(events, page.Nodes...)

		if !*page.PageInfo.HasNextPage {
			return events, nil
		}

		cursor := page.PageInfo.EndCursor
		if cursor == nil || strings.IsEmpty(*cursor) || (body.Variables.After != nil && *cursor == *body.Variables.After) {
			return nil, errors.New("github: automation events returned no advancing cursor")
		}

		body.Variables.After = cursor
	}
}

// PullRequestAutomationEventsRequest identifies a PR by node ID and supplies the cursor
// for the next page of its automation history.
type PullRequestAutomationEventsRequest struct {
	// Variables binds the operation's GraphQL arguments.
	Variables struct {
		// ID identifies the PR node whose complete automation history is requested.
		ID string `json:"id"`

		// After selects the next page; nil starts pagination.
		After *string `json:"after"`
	} `json:"variables"`

	// Query contains the GraphQL operation submitted with these variables.
	Query string `json:"query"`
}

// PullRequestAutomationEventsResponse decodes timeline history into SDK issue-event snapshots.
// Call Validate before using the history, even when the HTTP request succeeds.
type PullRequestAutomationEventsResponse struct {
	// Data contains returned operation data, which still requires error and identity validation.
	Data struct {
		// Node contains the PR identity and requested automation timeline page.
		Node struct {
			// ID must match the requested PR node ID before its history is used.
			ID string `json:"id"`

			// TimelineItems contains label and auto-merge-disable history used to establish consent.
			TimelineItems struct {
				// Nodes contains the page's automation events; nil indicates an incomplete page.
				Nodes []*IssueEvent `json:"nodes"`

				// PageInfo reports whether discovery must continue before the results are complete.
				PageInfo struct {
					// EndCursor selects the following page when HasNextPage is true.
					EndCursor *string `json:"endCursor"`

					// HasNextPage reports whether more results remain; nil indicates an incomplete response.
					HasNextPage *bool `json:"hasNextPage"`
				} `json:"pageInfo"`
			} `json:"timelineItems"`
		} `json:"node"`
	} `json:"data"`

	// Errors retains GraphQL failures even when the HTTP response succeeds.
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

// validateAutomationEvents requires timestamps, supported event types, and label
// names for label changes so incomplete history cannot authorize automation.
func validateAutomationEvents(events []*IssueEvent) error {
	for _, event := range events {
		if event.GetCreatedAt().IsZero() {
			return errors.New("github: automation event is missing its timestamp")
		}

		switch event.GetEvent() {
		case eventLabeled, eventUnlabeled:
			if strings.IsEmpty(event.GetLabel().GetName()) {
				return errors.New("github: automation event is missing its label")
			}
		case eventAutoMergeDisabled:
		default:
			return errors.New("github: automation event has an unexpected type")
		}
	}

	return nil
}
