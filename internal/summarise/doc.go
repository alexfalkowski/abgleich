// Package summarise authorizes pull-request summaries, sends changed-file
// patches returned by GitHub to the provider, and publishes one advisory text
// summary per pull request. Both branches must belong to the target repository;
// fork pull requests are ignored.
package summarise
