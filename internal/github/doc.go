// Package github implements the auto-merge repository using go-github for REST
// reads and GraphQL rebase requests. Its reusable client loads credentials at
// startup. GitHub performs rebases guarded by the expected PR head SHA.
package github
