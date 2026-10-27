package http

import (
	"github.com/alexfalkowski/abgleich/internal/automerge"
	"github.com/alexfalkowski/abgleich/internal/commands"
	"github.com/alexfalkowski/abgleich/internal/github"
	"github.com/alexfalkowski/go-service/v2/context"
	"github.com/alexfalkowski/go-service/v2/di"
	"github.com/alexfalkowski/go-service/v2/errors"
	"github.com/alexfalkowski/go-service/v2/net/http"
	"github.com/alexfalkowski/go-service/v2/net/http/status"
	"github.com/alexfalkowski/go-service/v2/telemetry/logger"
)

// HandlerParams supplies webhook verification, delivery routing, and logging dependencies.
type HandlerParams struct {
	di.In

	Webhook     *github.Webhook
	Dispatcher  *commands.Dispatcher
	Coordinator *automerge.Coordinator
	Logger      *logger.Logger
}

// NewHandler keeps verification and command execution within each delivery's deadline.
// It returns nil when webhook deliveries are disabled.
func NewHandler(params HandlerParams) *Handler {
	if params.Webhook == nil {
		return nil
	}

	return &Handler{webhook: params.Webhook, dispatcher: params.Dispatcher, coordinator: params.Coordinator, logger: params.Logger}
}

// Handler verifies GitHub deliveries before dispatching comments or queueing
// repository checks. Ignored, queued, and completed deliveries return HTTP 204.
type Handler struct {
	webhook     *github.Webhook
	dispatcher  *commands.Dispatcher
	coordinator *automerge.Coordinator
	logger      *logger.Logger
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

	if err := h.handle(ctx, event); err != nil {
		h.writeError(req.Context(), res, err)
		return
	}

	res.WriteHeader(http.StatusNoContent)
}

func (h *Handler) handle(ctx context.Context, delivery *github.Delivery) error {
	if delivery.Comment != nil {
		return h.dispatcher.Handle(ctx, delivery.Comment)
	}

	h.coordinator.Enqueue(delivery.Repository)
	return nil
}

func (h *Handler) writeError(ctx context.Context, res http.ResponseWriter, err error) {
	h.logger.Error("github webhook failed", "error", err)
	_ = status.WriteError(ctx, res, h.error(err))
}
