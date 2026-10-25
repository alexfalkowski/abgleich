package github

import (
	"github.com/alexfalkowski/go-service/v2/errors"
	"github.com/alexfalkowski/go-service/v2/strings"
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
