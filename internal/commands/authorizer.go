package commands

import (
	"github.com/alexfalkowski/abgleich/internal/github"
	"github.com/alexfalkowski/go-service/v2/context"
)

// NewAuthorizer constructs shared human-command consent checks without remote calls.
func NewAuthorizer(service github.Service) *Authorizer {
	return &Authorizer{github: service}
}

// Authorizer re-reads command comments and current permissions on every call.
// Callers own installation validation, PR eligibility, and when to recheck consent.
type Authorizer struct {
	github github.Service
}

// Authorize returns the current comment for an unedited human /ab command with
// write/admin access. Name is the exact command suffix. GetComment validates the
// expected comment's PR, author, and creation identity. Invalid comments return
// nil, nil; denied permissions return ErrWriteAccessRequired and read failures
// retain their error. No authorization result is cached.
func (a *Authorizer) Authorize(ctx context.Context, name string, repo *github.Repository, number int, expected *github.IssueComment) (*github.IssueComment, error) {
	comment, err := a.github.GetComment(ctx, repo, number, expected)
	if err != nil {
		return nil, err
	}

	matchesCommand := Parse(comment.GetBody()) == name
	isHuman := comment.GetUser().GetType() == "User"
	hasMetadata := github.HasCommentMetadata(comment)
	if !matchesCommand || !isHuman || !hasMetadata {
		return nil, nil
	}

	isEdited := comment.GetUpdatedAt().After(comment.GetCreatedAt().Time)
	if isEdited {
		return nil, nil
	}

	allowed, err := a.github.HasWritePermission(ctx, repo, comment.GetUser())
	if err != nil {
		return nil, err
	}

	if !allowed {
		return nil, ErrWriteAccessRequired
	}
	return comment, nil
}
