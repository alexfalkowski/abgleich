// Package http registers the GitHub webhook on the shared HTTP server. It owns
// request deadlines and responses, delegates verification to github.Webhook,
// and passes verified comments to commands.Dispatcher. Body limits are enforced
// by go-service's transport configuration before the handler runs.
package http
