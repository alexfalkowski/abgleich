// Package automerge keeps eligible pull request branches current with their
// bases. Rebaser owns eligibility and rebase decisions, using the github package
// for remote pull request state and guarded rebase requests. Module registers
// Rebase with the shared scheduler as the automerge job.
package automerge
