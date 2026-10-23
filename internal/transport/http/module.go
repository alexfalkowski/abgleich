package http

import (
	"github.com/alexfalkowski/go-service/v2/di"
	"github.com/alexfalkowski/go-service/v2/net/http"
)

// Module wires inbound GitHub webhook handling into the shared HTTP server.
var Module = di.Module(
	di.Constructor(NewHandler),
	di.Register(Register),
)

// Register exposes the webhook only when configured. GitHub signatures protect
// the route, which bypasses the service's transport token authentication.
func Register(router *http.Router, handler *Handler) {
	if handler != nil {
		router.HandleRoute("POST /webhooks/github", handler, http.WithRouteUnauthenticated())
	}
}
