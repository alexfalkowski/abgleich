package github

import (
	"github.com/alexfalkowski/go-service/v2/context"
	"github.com/alexfalkowski/go-service/v2/errors"
)

const (
	permissionWrite = "write"
	permissionAdmin = "admin"
)

// HasWritePermission reads effective permissions, including maintain and admin.
func (s *service) HasWritePermission(ctx context.Context, repo *Repository, author *User) (bool, error) {
	permission, _, err := s.client.installation.Repositories.GetPermissionLevel(ctx, repo.GetOwner().GetLogin(), repo.GetName(),
		author.GetLogin())
	if err != nil {
		return false, err
	}

	if permission.GetUser().GetID() != author.GetID() {
		return false, errors.New("github: permissions returned a different author")
	}

	return permission.GetPermission() == permissionWrite || permission.GetPermission() == permissionAdmin, nil
}
