package github

import (
	"github.com/alexfalkowski/go-service/v3/context"
	"github.com/alexfalkowski/go-service/v3/errors"
	"github.com/alexfalkowski/go-service/v3/strings"
	"github.com/google/go-github/v93/github"
)

const submoduleURLPrefix = "git@github.com:"

// GetSubmoduleRepository reads the mount's metadata at the parent revision and
// requires a public target. The pinned commit remains entry.GetSHA().
func (s *service) GetSubmoduleRepository(ctx context.Context, repo *Repository, revision string, entry *TreeEntry) (*Repository, error) {
	if !IsSubmodule(entry) || strings.IsEmpty(entry.GetSHA()) || strings.IsEmpty(revision) {
		return nil, errors.New("github: invalid submodule entry")
	}

	opts := &github.RepositoryContentGetOptions{Ref: revision}
	content, _, _, err := s.client.installation.Repositories.GetContents(ctx, repo.GetOwner().GetLogin(), repo.GetName(), entry.GetPath(), opts)
	if err != nil {
		return nil, errors.New("github: submodule metadata unavailable")
	}

	if content.GetPath() != entry.GetPath() || content.GetSHA() != entry.GetSHA() {
		return nil, errors.New("github: submodule metadata does not match its tree entry")
	}

	name := submoduleRepositoryName(repo, content.GetSubmoduleGitURL())
	if strings.IsEmpty(name) {
		return nil, errors.New("github: submodule must use a same-owner git@github.com URL")
	}

	return s.publicSubmoduleRepository(ctx, name)
}

func (s *service) publicSubmoduleRepository(ctx context.Context, name string) (*Repository, error) {
	target, err := s.FindRepository(ctx, name)
	if err != nil {
		return nil, err
	}

	if target == nil {
		return nil, errors.New("github: submodule repository unavailable to the installation")
	}

	if target.Private == nil || target.GetPrivate() {
		return nil, errors.New("github: submodule repository must be public")
	}

	return target, nil
}

func submoduleRepositoryName(repo *Repository, raw string) string {
	if !strings.HasPrefix(raw, submoduleURLPrefix) {
		return strings.Empty
	}

	name := strings.TrimSuffix(strings.TrimPrefix(raw, submoduleURLPrefix), ".git")
	owner, _, _ := strings.Cut(name, "/")

	if strings.ToLower(owner) != strings.ToLower(repo.GetOwner().GetLogin()) {
		return strings.Empty
	}

	return name
}
