package http

import (
	"github.com/alexfalkowski/go-service/v3/net/http"
	"github.com/alexfalkowski/go-signal/v2"
)

// Module wires inbound GitHub webhook handling into the shared HTTP server.
var Module = signal.Module(
	signal.Constructor(NewHandler),
	signal.Register(Register),
)

// Register exposes the webhook only when configured. GitHub signatures protect
// the route, which bypasses the service's transport token authentication.
func Register(router *http.Router, handler *Handler) {
	if handler != nil {
		router.HandleRoute("POST /webhooks/github", handler, http.WithRouteUnauthenticated())
	}
}
