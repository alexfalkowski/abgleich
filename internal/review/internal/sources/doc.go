// Package sources reads repository and submodule source at pinned revisions,
// selects provider guides and tools, and records evidence for each review.
// Tool calls use a consent and revision guard supplied by the workflow.
// Providers retain shared transports and keep lookup caches per review.
package sources
