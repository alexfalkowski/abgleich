// Package commands parses /ab PR comment commands and dispatches
// exact names to feature-owned handlers. It owns installation validation,
// overlap protection, reaction feedback, and optional background queues.
// Feature modules register unique names before startup and own each command's
// authorization, current-state checks, and action. Long-running work uses a Queue
// processed by a non-overlapping scheduled job. HTTP verification, deadlines,
// and responses belong to transport/http; GitHub access belongs to github.
package commands
