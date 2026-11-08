package github

import (
	_ "embed"

	"github.com/alexfalkowski/go-service/v2/context"
	"github.com/alexfalkowski/go-service/v2/errors"
	"github.com/alexfalkowski/go-service/v2/strings"
)

const deletedOID = "0000000000000000000000000000000000000000"

// updateRefs supports deleting a reference while checking its previous OID.
// https://docs.github.com/en/graphql/reference/git#updaterefs
//
//go:embed graphql/delete_branch.gql
var deleteBranchMutation string

// DeleteBranch uses updateRefs with beforeOid so a concurrent push rejects deletion.
func (s *service) DeleteBranch(ctx context.Context, repo *Repository, branch *Branch) error {
	if err := validateBranch(branch); err != nil {
		return err
	}

	if strings.IsAnyEmpty(repo.GetNodeID(), repo.GetDefaultBranch()) ||
		branch.GetProtected() || branch.GetName() == repo.GetDefaultBranch() {
		return errors.New("github: branch cannot be deleted")
	}

	body := UpdateRefsRequest{Query: deleteBranchMutation}
	input := &body.Variables.Input
	input.RepositoryID = repo.GetNodeID()
	input.ClientMutationID = strings.Concat(repo.GetNodeID(), ":", branch.GetName(), ":", branch.GetCommit().GetSHA())
	input.RefUpdates = []RefUpdate{
		{Name: branchPrefix + branch.GetName(), BeforeOID: branch.GetCommit().GetSHA(), AfterOID: deletedOID},
	}

	var response UpdateRefsResponse
	if err := s.graphql(ctx, body, &response); err != nil {
		return err
	}

	return response.Validate(input.ClientMutationID)
}

// RefUpdate carries the expected and replacement OIDs for an atomic ref update.
type RefUpdate struct {
	// Name is the fully qualified reference name, such as refs/heads/feature.
	Name string `json:"name"`

	// BeforeOID requires the reference to still point at this commit before it can change.
	BeforeOID string `json:"beforeOid"`

	// AfterOID sets the replacement commit; the all-zero OID deletes the reference.
	AfterOID string `json:"afterOid"`
}

// UpdateRefsRequest deletes one branch with an atomic expected-commit guard.
// The zero AfterOID deletes the reference; ClientMutationID must be echoed.
type UpdateRefsRequest struct {
	// Query contains the GraphQL operation submitted with these variables.
	Query string `json:"query"`

	// Variables binds the operation's GraphQL arguments.
	Variables struct {
		// Input supplies the mutation arguments, including the target identity.
		Input struct {
			// RepositoryID identifies the repository by its GraphQL node ID.
			RepositoryID string `json:"repositoryId"`

			// ClientMutationID correlates the mutation request with its acceptance response.
			ClientMutationID string `json:"clientMutationId"`

			// RefUpdates contains the guarded reference changes; cleanup submits one deletion at a time.
			RefUpdates []RefUpdate `json:"refUpdates"`
		} `json:"input"`
	} `json:"variables"`
}

// UpdateRefsResponse requires a matching mutation ID and no GraphQL errors.
type UpdateRefsResponse struct {
	// Data contains returned operation data, which still requires error and identity validation.
	Data struct {
		// Update confirms an accepted updateRefs mutation; nil cannot confirm success.
		Update *struct {
			// ClientMutationID correlates the mutation request with its acceptance response.
			ClientMutationID string `json:"clientMutationId"`
		} `json:"updateRefs"`
	} `json:"data"`

	// Errors retains GraphQL failures even when the HTTP response succeeds.
	Errors Errors `json:"errors"`
}

// Validate rejects absent, unconfirmed, or failed reference updates.
func (r *UpdateRefsResponse) Validate(expectedID string) error {
	if err := r.Errors.Validate(); err != nil {
		return errors.Prefix("github: delete branch", err)
	}

	if r.Data.Update == nil || r.Data.Update.ClientMutationID != expectedID {
		return errors.New("github: delete branch returned no matching mutation")
	}

	return nil
}
