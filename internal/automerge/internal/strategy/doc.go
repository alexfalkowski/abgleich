// Package strategy owns pull request eligibility and fresh pre-mutation checks
// for Dependabot, labeled contributions, and auto-merge branch rebasing. Callers
// choose the order and select one strategy per pull request. Decisions expose
// the same policy without report wording or mutations; github owns remote
// state, identity, and mutations.
package strategy
