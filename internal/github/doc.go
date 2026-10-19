// Package github provides owner-wide discovery and repository access through
// Service, using go-github for REST reads and merges, and GraphQL rebase and
// auto-merge requests. Its client loads credentials at startup. Rebases and
// immediate merges are guarded by the expected PR head SHA.
package github
