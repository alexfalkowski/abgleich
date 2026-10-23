package http

import (
	"github.com/alexfalkowski/abgleich/internal/commands"
	"github.com/alexfalkowski/go-service/v2/errors"
	"github.com/alexfalkowski/go-service/v2/net/http"
	"github.com/alexfalkowski/go-service/v2/net/http/status"
)

func (h *Handler) error(err error) error {
	if errors.Is(err, commands.ErrCommandInProgress) {
		return status.Error(http.StatusServiceUnavailable, commands.ErrCommandInProgress.Error())
	}

	if errors.Is(err, commands.ErrWriteAccessRequired) {
		return status.Error(http.StatusForbidden, commands.ErrWriteAccessRequired.Error())
	}

	if errors.Is(err, commands.ErrPullRequestChanged) {
		return status.Error(http.StatusConflict, commands.ErrPullRequestChanged.Error())
	}

	return status.SafeError(http.StatusServiceUnavailable, err)
}
