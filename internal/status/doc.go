// Package status reports observed automation policy in response to authorized
// /ab status PR comments. It owns command authorization, snapshot rechecks,
// rendering, and publication. Automerge owns policy assessment; commands owns
// dispatch and reactions, and github owns remote access and response validation.
package status
