// Package http registers the GitHub webhook on the shared HTTP server. It owns
// request deadlines and responses, delegates verification to github.Webhook,
// and passes verified comments to commands.Dispatcher or repository notifications
// to automerge.Coordinator. Body limits are enforced
// by go-service's transport configuration before the handler runs.
package http
