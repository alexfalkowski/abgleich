package http

import (
	"github.com/alexfalkowski/abgleich/internal/commands"
	"github.com/alexfalkowski/abgleich/internal/github"
	"github.com/alexfalkowski/go-service/v2/context"
	"github.com/alexfalkowski/go-service/v2/errors"
	"github.com/alexfalkowski/go-service/v2/net/http"
	"github.com/alexfalkowski/go-service/v2/net/http/status"
	"github.com/alexfalkowski/go-service/v2/telemetry/logger"
)

// NewHandler keeps verification and command execution within each delivery's deadline.
// It returns nil when webhook deliveries are disabled.
func NewHandler(webhook *github.Webhook, dispatcher *commands.Dispatcher, log *logger.Logger) *Handler {
	if webhook == nil {
		return nil
	}

	return &Handler{webhook: webhook, dispatcher: dispatcher, logger: log}
}

// Handler verifies GitHub deliveries before passing comments to command policy.
// Processing remains synchronous; ignored and completed deliveries return HTTP 204.
type Handler struct {
	webhook    *github.Webhook
	dispatcher *commands.Dispatcher
	logger     *logger.Logger
}

// ServeHTTP bounds processing by the webhook budget. The shared HTTP transport
// enforces request body limits before this handler runs.
func (h *Handler) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	ctx, cancel := context.WithTimeout(req.Context(), h.webhook.GetTimeout())
	defer cancel()

	event, err := h.webhook.Read(req)
	if errors.Is(err, github.ErrIgnoredDelivery) {
		res.WriteHeader(http.StatusNoContent)
		return
	}

	if err != nil {
		h.writeError(req.Context(), res, err)
		return
	}

	if err := h.dispatcher.Handle(ctx, event); err != nil {
		h.writeError(req.Context(), res, err)
		return
	}

	res.WriteHeader(http.StatusNoContent)
}

func (h *Handler) writeError(ctx context.Context, res http.ResponseWriter, err error) {
	h.logger.Error("github comment command failed", "error", err)
	_ = status.WriteError(ctx, res, h.error(err))
}
