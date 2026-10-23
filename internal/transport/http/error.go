package http

import (
	"github.com/alexfalkowski/abgleich/internal/automerge"
	"github.com/alexfalkowski/go-service/v2/errors"
	"github.com/alexfalkowski/go-service/v2/net/http"
	"github.com/alexfalkowski/go-service/v2/net/http/status"
)

func (h *Handler) error(err error) error {
	if errors.Is(err, automerge.ErrCommandInProgress) {
		return status.Error(http.StatusServiceUnavailable, automerge.ErrCommandInProgress.Error())
	}

	if errors.Is(err, automerge.ErrWriteAccessRequired) {
		return status.Error(http.StatusForbidden, automerge.ErrWriteAccessRequired.Error())
	}

	if errors.Is(err, automerge.ErrPullRequestChanged) {
		return status.Error(http.StatusConflict, automerge.ErrPullRequestChanged.Error())
	}

	return status.SafeError(http.StatusServiceUnavailable, err)
}
