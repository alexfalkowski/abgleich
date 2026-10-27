package github

import "github.com/alexfalkowski/go-service/v2/strings"

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
