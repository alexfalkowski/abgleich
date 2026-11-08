package github

import (
	"github.com/alexfalkowski/go-service/v2/strings"
	"github.com/google/go-github/v93/github"
)

// IsRepository compares the complete owner/repository name case-insensitively.
func (s *service) IsRepository(repo *Repository, name string) bool {
	return strings.ToLower(repo.GetFullName()) == strings.ToLower(name)
}

// IsSameRepository checks both PR branch repositories against the discovered
// repository using case-insensitive full names.
func (s *service) IsSameRepository(repo *Repository, pull *PullRequest) bool {
	name := strings.ToLower(repo.GetFullName())
	headRepository := strings.ToLower(pull.GetHead().GetRepo().GetFullName())

	return headRepository == name && s.IsBaseRepository(repo, pull)
}

// IsBaseRepository permits fork heads while requiring a matching base repository.
// Repository full names are compared case-insensitively.
func (s *service) IsBaseRepository(repo *Repository, pull *PullRequest) bool {
	return strings.ToLower(repo.GetFullName()) == strings.ToLower(pull.GetBase().GetRepo().GetFullName())
}

// IsSamePullRequestIdentity compares the PR number and node ID across snapshots
// and requires current to target repo. Repository names ignore case; branch refs
// and commit SHAs may differ.
func (s *service) IsSamePullRequestIdentity(repo *Repository, current, expected *PullRequest) bool {
	sameNumber := current.GetNumber() == expected.GetNumber()
	sameNodeID := current.GetNodeID() == expected.GetNodeID()
	sameRepository := s.IsBaseRepository(repo, current)

	return sameNumber && sameNodeID && sameRepository
}

// IsSamePullRequest detects changes to the PR identity, branch repositories,
// refs, or head SHA between snapshots. Repository names ignore case.
func (s *service) IsSamePullRequest(current, expected *PullRequest) bool {
	currentHead, expectedHead := current.GetHead(), expected.GetHead()
	currentBase, expectedBase := current.GetBase(), expected.GetBase()
	sameID := current.GetNodeID() == expected.GetNodeID() && current.GetNumber() == expected.GetNumber()
	sameHead := currentHead.GetRef() == expectedHead.GetRef() &&
		currentHead.GetSHA() == expectedHead.GetSHA() &&
		strings.ToLower(currentHead.GetRepo().GetFullName()) == strings.ToLower(expectedHead.GetRepo().GetFullName())
	sameBase := currentBase.GetRef() == expectedBase.GetRef() &&
		strings.ToLower(currentBase.GetRepo().GetFullName()) == strings.ToLower(expectedBase.GetRepo().GetFullName())

	return sameID && sameHead && sameBase
}

type repositoryIdentity interface {
	GetOwner() *github.User
	GetName() string
	GetFullName() string
}
