package github

import (
	"github.com/alexfalkowski/go-service/v2/context"
	"github.com/alexfalkowski/go-service/v2/errors"
	"github.com/alexfalkowski/go-service/v2/net/http"
	"github.com/alexfalkowski/go-service/v2/net/http/media"
	"github.com/alexfalkowski/go-service/v2/strings"
	"github.com/alexfalkowski/go-service/v2/time"
)

// Alias GraphQL fields onto SDK issue events. AutoMergeDisabledEvent is exposed
// by GraphQL, so REST issue events alone cannot preserve a disabled auto-merge.
// https://docs.github.com/en/graphql/reference/pulls#automergedisabledevent
const automationEventsQuery = `query PullRequestAutomationEvents($id: ID!, $after: String) {
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

// LatestAutomationEventTimes returns the latest label application, removal, and
// auto-merge disable timestamps, in that order. Missing events return zero times.
// History must be validated, and label matching is exact. Disabling auto-merge
// is independent of the label; callers decide whether these facts revoke access.
func LatestAutomationEventTimes(events []*IssueEvent, label string) (time.Time, time.Time, time.Time) {
	var labeledAt, removedAt, disabledAt time.Time

	for _, event := range events {
		occurredAt := event.GetCreatedAt().Time

		switch event.GetEvent() {
		case "LabeledEvent":
			if event.GetLabel().GetName() == label && occurredAt.After(labeledAt) {
				labeledAt = occurredAt
			}

		case "UnlabeledEvent":
			if event.GetLabel().GetName() == label && occurredAt.After(removedAt) {
				removedAt = occurredAt
			}

		case "AutoMergeDisabledEvent":
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
		request, err := s.client.installation.NewRequest(ctx, http.MethodPost, "graphql", body)
		if err != nil {
			return nil, err
		}

		request.Header.Set(http.AcceptKey, media.JSON)

		var response PullRequestAutomationEventsResponse
		if _, err := s.client.installation.Do(request, &response); err != nil {
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
