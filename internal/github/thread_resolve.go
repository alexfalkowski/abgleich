package github

import (
	_ "embed"

	"github.com/alexfalkowski/go-service/v2/context"
	"github.com/alexfalkowski/go-service/v2/errors"
)

// resolveReviewThreadMutation returns the thread ID and resolved state so the
// service can confirm acceptance. GitHub has no expected-state guard; callers
// refresh eligibility beforehand, but a change after that read can still race.
// https://docs.github.com/en/graphql/reference/pulls#resolvereviewthread
//
//go:embed graphql/resolve_review_thread.gql
var resolveReviewThreadMutation string

// ResolveReviewThread confirms matching resolved state without retrying failures.
// Callers own eligibility and must refresh the PR and thread before mutation.
func (s *service) ResolveReviewThread(ctx context.Context, _ *Repository, _ *PullRequest, thread *ReviewThread) error {
	body := ResolveReviewThreadRequest{Query: resolveReviewThreadMutation}
	body.Variables.Input.ThreadID = thread.ID

	var response ResolveReviewThreadResponse
	if err := s.graphql(ctx, body, &response); err != nil {
		return err
	}
	if err := response.Errors.Validate(); err != nil {
		return err
	}

	resolved := response.Data.Resolve.Thread
	sameThread := resolved != nil && resolved.ID == thread.ID
	isResolved := resolved != nil && resolved.IsResolved != nil && *resolved.IsResolved

	if !sameThread || !isResolved {
		return errors.New("github: resolve review thread returned no matching resolved thread")
	}

	return nil
}

// ResolveReviewThreadRequest resolves a thread by node ID without an
// expected-state guard; callers must refresh eligibility before submission.
type ResolveReviewThreadRequest struct {
	// Query contains the GraphQL operation submitted with these variables.
	Query string `json:"query"`

	// Variables binds the operation's GraphQL arguments.
	Variables struct {
		// Input supplies the mutation arguments, including the target identity.
		Input struct {
			// ThreadID identifies the thread to resolve after final eligibility rechecks.
			ThreadID string `json:"threadId"`
		} `json:"input"`
	} `json:"variables"`
}

// ResolveReviewThreadResponse confirms a matching thread's resolved state.
// GraphQL errors or missing confirmation must not be treated as acceptance.
type ResolveReviewThreadResponse struct {
	// Data contains returned operation data, which still requires error and identity validation.
	Data struct {
		// Resolve contains the resolveReviewThread acceptance payload.
		Resolve struct {
			// Thread confirms the target thread and its resolved state; nil cannot confirm acceptance.
			Thread *struct {
				// ID must match the requested thread node ID to confirm acceptance.
				ID string `json:"id"`

				// IsResolved must be present and true to confirm that the mutation resolved the thread.
				IsResolved *bool `json:"isResolved"`
			} `json:"thread"`
		} `json:"resolveReviewThread"`
	} `json:"data"`

	// Errors retains GraphQL failures even when the HTTP response succeeds.
	Errors Errors `json:"errors"`
}
