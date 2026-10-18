// Package github provides owner-wide discovery and repository access through
// Service, using go-github for REST reads and GraphQL rebase requests. Its client loads
// credentials at startup. GitHub performs rebases guarded by the expected PR head SHA.
package github
