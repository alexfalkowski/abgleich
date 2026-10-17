// Package github owns GitHub repository access and pull request snapshots, using
// go-github for REST reads and GraphQL rebase requests. Its reusable client loads
// credentials at startup. GitHub performs rebases guarded by the expected PR head SHA.
package github
