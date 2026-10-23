package github

import (
	"github.com/alexfalkowski/go-service/v2/context"
	"github.com/alexfalkowski/go-service/v2/errors"
	"github.com/alexfalkowski/go-service/v2/strings"
)

// ValidateInstallation rejects an installation for another App or account and
// suspended installations. Each sweep and comment command checks fresh state.
func (s *service) ValidateInstallation(ctx context.Context) error {
	installation, _, err := s.client.application.Apps.GetInstallation(ctx, s.config.App.InstallationID)
	if err != nil {
		return errors.Prefix("github installation", err)
	}

	if installation.GetID() != s.config.App.InstallationID || installation.GetAppID() != s.config.App.ID ||
		strings.ToLower(installation.GetAccount().GetLogin()) != strings.ToLower(s.config.Owner) || installation.SuspendedAt != nil {
		return errors.New("github installation does not match the configured app and owner or is suspended")
	}

	return nil
}
