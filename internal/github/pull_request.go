package github

import (
	"github.com/alexfalkowski/go-service/v2/context"
	"github.com/alexfalkowski/go-service/v2/errors"
	"github.com/alexfalkowski/go-service/v2/strings"
)

// PullOpen identifies an open pull request in REST responses and filters.
const PullOpen = "open"

// PullClosed includes merged pull requests in REST responses.
const PullClosed = "closed"

// MergeableClean indicates that GitHub reports no mergeability blockers.
const MergeableClean = "clean"

// MergeableUnstable indicates non-passing checks while the pull request remains mergeable.
const MergeableUnstable = "unstable"

const (
	updateRebase    = "REBASE"
	autoMergeSquash = "SQUASH"
	squashMethod    = "squash"
)

const (
	// GitHub exposes server-side rebases through GraphQL, while the REST branch
	// update only merges. Keep using go-github's authenticated request transport.
	// https://docs.github.com/en/graphql/reference/pulls#updatepullrequestbranch
	rebaseMutation = `mutation RebasePullRequest($input: UpdatePullRequestBranchInput!) {
  updatePullRequestBranch(input: $input) {
    pullRequest { id }
  }
}`

	// GitHub enables auto-merge through GraphQL and merges once repository
	// requirements pass. Keep using go-github's authenticated request transport.
	// https://docs.github.com/en/graphql/reference/pulls#enablepullrequestautomerge
	autoMergeMutation = `mutation EnableAutoMerge($input: EnablePullRequestAutoMergeInput!) {
  enablePullRequestAutoMerge(input: $input) {
    pullRequest { id }
  }
}`
)

// HasPullRequestNodeID checks the identity required by GraphQL operations.
// A numeric pull request ID does not substitute for its node ID.
func HasPullRequestNodeID(pull *PullRequest) bool {
	return !strings.IsEmpty(pull.GetNodeID())
}

// ValidatePullRequestMetadata requires a node ID, head SHA, branch references,
// and repository names before guarded automation. Callers decide which PRs need
// this validation, so incomplete snapshots of ineligible PRs can be ignored.
func ValidatePullRequestMetadata(pull *PullRequest) error {
	head, base := pull.GetHead(), pull.GetBase()
	if strings.IsAnyEmpty(
		pull.GetNodeID(),
		head.GetSHA(),
		head.GetRef(),
		head.GetRepo().GetFullName(),
		base.GetRef(),
		base.GetRepo().GetFullName(),
	) {
		return errors.New("github: pull request is missing branch metadata")
	}

	return nil
}

// UpdatePullRequestBranchInput supplies GitHub's branch-update arguments.
// ExpectedHeadOID rejects concurrent head changes; UpdateMethod selects REBASE
// or MERGE.
type UpdatePullRequestBranchInput struct {
	// PullRequestID identifies the target PR by its GraphQL node ID, rather than its numeric ID.
	PullRequestID string `json:"pullRequestId"`

	// ExpectedHeadOID guards the mutation against concurrent changes to the PR head commit.
	ExpectedHeadOID string `json:"expectedHeadOid"`

	// UpdateMethod selects the GitHub update strategy; the worker requests REBASE.
	UpdateMethod string `json:"updateMethod"`
}

// UpdatePullRequestBranchRequest encodes a branch-update mutation. Query must
// declare $input with the GraphQL type UpdatePullRequestBranchInput.
type UpdatePullRequestBranchRequest struct {
	// Query contains the GraphQL operation submitted with these variables.
	Query string `json:"query"`

	// Variables binds the operation's GraphQL arguments.
	Variables struct {
		// Input supplies the mutation arguments, including the target identity.
		Input UpdatePullRequestBranchInput `json:"input"`
	} `json:"variables"`
}

// UpdatePullRequestBranchResponse decodes GitHub's branch-update mutation.
// HTTP success can include GraphQL errors or incomplete data; call Validate
// before treating an update as accepted.
type UpdatePullRequestBranchResponse struct {
	// Data contains returned operation data, which still requires error and identity validation.
	Data struct {
		// Update contains the branch-update mutation's acceptance payload.
		Update struct {
			// PullRequest identifies the PR accepted for a branch update.
			PullRequest struct {
				// ID must match the requested PR node ID to confirm acceptance.
				ID string `json:"id"`
			} `json:"pullRequest"`
		} `json:"updatePullRequestBranch"`
	} `json:"data"`

	// Errors retains GraphQL failures even when the HTTP response succeeds.
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

// Rebase requests a server-side rebase using pull's node ID and expected head
// SHA. HTTP failures, GraphQL errors, and mismatched response IDs are errors.
// Success confirms acceptance, not completion; the mutation is not retried.
func (s *service) Rebase(ctx context.Context, pull *PullRequest) error {
	nodeID := pull.GetNodeID()
	body := UpdatePullRequestBranchRequest{Query: rebaseMutation}
	body.Variables.Input = UpdatePullRequestBranchInput{
		PullRequestID: nodeID, ExpectedHeadOID: pull.GetHead().GetSHA(), UpdateMethod: updateRebase,
	}

	var response UpdatePullRequestBranchResponse
	if err := s.graphql(ctx, body, &response); err != nil {
		return err
	}

	return response.Validate(nodeID)
}

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
