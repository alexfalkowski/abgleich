package github

import (
	"github.com/alexfalkowski/go-service/v2/context"
	"github.com/alexfalkowski/go-service/v2/errors"
	"github.com/alexfalkowski/go-service/v2/net/http"
	"github.com/alexfalkowski/go-service/v2/net/http/media"
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

// ListPullRequestAutomationEvents collects and validates every timeline page
// before returning history. Any invalid page or stalled cursor rejects the history.
func (s *service) ListPullRequestAutomationEvents(ctx context.Context, pull *PullRequest) ([]*IssueEvent, error) {
	body := PullRequestAutomationEventsRequest{Query: automationEventsQuery}
	body.Variables.ID = pull.GetNodeID()

	var events []*IssueEvent
	for {
		request, err := s.client.NewRequest(ctx, http.MethodPost, "/graphql", body)
		if err != nil {
			return nil, err
		}

		request.Header.Set(http.AcceptKey, media.JSON)

		var response PullRequestAutomationEventsResponse
		if _, err := s.client.Do(request, &response); err != nil {
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
		if cursor == nil || *cursor == "" || (body.Variables.After != nil && *cursor == *body.Variables.After) {
			return nil, errors.New("github automation events returned no advancing cursor")
		}

		body.Variables.After = cursor
	}
}
