// Package commands parses /ab PR comment commands and dispatches
// exact names to feature-owned handlers. It owns installation validation,
// overlap protection, reaction feedback, shared human-command consent, and
// optional background queues.
// Feature modules register unique names before startup and own PR eligibility,
// current-state checks, and actions. Authorizer refreshes human comment identity
// and permissions wherever the caller requires fresh consent. Long-running work uses a Queue
// processed by a non-overlapping job, optionally woken by queue notifications. HTTP verification, deadlines,
// and responses belong to transport/http; GitHub access belongs to github.
package commands
