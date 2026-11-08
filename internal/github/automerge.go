package github

import (
	_ "embed"

	"github.com/alexfalkowski/go-service/v2/context"
	"github.com/alexfalkowski/go-service/v2/errors"
)

const autoMergeSquash = "SQUASH"

// GitHub enables auto-merge through GraphQL and merges once repository
// requirements pass. Keep using go-github's authenticated request transport.
// https://docs.github.com/en/graphql/reference/pulls#enablepullrequestautomerge
//
//go:embed graphql/auto_merge.gql
var autoMergeMutation string

// EnableAutoMerge enables squash auto-merge using the PR's node ID and expected
// head SHA. GitHub enforces repository rules. Failures are not retried.
func (s *service) EnableAutoMerge(ctx context.Context, pull *PullRequest) error {
	body := EnablePullRequestAutoMergeRequest{Query: autoMergeMutation}
	body.Variables.Input = EnablePullRequestAutoMergeInput{
		PullRequestID: pull.GetNodeID(), ExpectedHeadOID: pull.GetHead().GetSHA(), MergeMethod: autoMergeSquash,
	}

	var response EnablePullRequestAutoMergeResponse
	if err := s.graphql(ctx, body, &response); err != nil {
		return err
	}

	return response.Validate(pull.GetNodeID())
}

// EnablePullRequestAutoMergeInput selects a PR and merge method for auto-merge,
// guarded by ExpectedHeadOID against concurrent pushes.
type EnablePullRequestAutoMergeInput struct {
	// PullRequestID identifies the target PR by its GraphQL node ID, rather than its numeric ID.
	PullRequestID string `json:"pullRequestId"`

	// ExpectedHeadOID guards the mutation against concurrent changes to the PR head commit.
	ExpectedHeadOID string `json:"expectedHeadOid"`

	// MergeMethod selects the eventual merge strategy; the worker requests SQUASH.
	MergeMethod string `json:"mergeMethod"`
}

// EnablePullRequestAutoMergeRequest encodes GitHub's auto-merge mutation.
type EnablePullRequestAutoMergeRequest struct {
	// Query contains the GraphQL operation submitted with these variables.
	Query string `json:"query"`

	// Variables binds the operation's GraphQL arguments.
	Variables struct {
		// Input supplies the mutation arguments, including the target identity.
		Input EnablePullRequestAutoMergeInput `json:"input"`
	} `json:"variables"`
}

// EnablePullRequestAutoMergeResponse requires validation even on HTTP success.
type EnablePullRequestAutoMergeResponse struct {
	// Data contains returned operation data, which still requires error and identity validation.
	Data struct {
		// Enable contains the enablePullRequestAutoMerge acceptance payload.
		Enable struct {
			// PullRequest identifies the PR accepted for auto-merge enablement.
			PullRequest struct {
				// ID must match the requested PR node ID to confirm acceptance.
				ID string `json:"id"`
			} `json:"pullRequest"`
		} `json:"enablePullRequestAutoMerge"`
	} `json:"data"`

	// Errors retains GraphQL failures even when the HTTP response succeeds.
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
