// Package github provides owner-wide discovery and repository access through
// Service, using go-github for REST reads and merges, and GraphQL rebase and
// auto-merge requests and automation history. Client loads the App private key at
// startup. Rebases, auto-merge enablement, and immediate merges are guarded by
// the expected PR head SHA.
// Signing and installation token renewal are delegated to ghinstallation.
// Webhook verifies raw GitHub deliveries; worker and comment requests share Service.
// Snapshot completeness, label membership, and timeline timestamp extraction
// belong here; callers decide whether those facts authorize automation.
// Service composes accepted-action logging and caching decorators. Only
// validated source trees and blobs at pinned revisions use go-service's in-memory
// ttlcache backend. Other backends are bypassed for source reads; mutable GitHub
// state is always read fresh and cache errors fall back to GitHub.
package github
