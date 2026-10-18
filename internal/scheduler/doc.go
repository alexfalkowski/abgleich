// Package scheduler runs independently scheduled jobs and logs their outcomes
// and durations. Feature packages register named handlers with Scheduler before
// startup. Each job runs immediately and at its configured interval, skipping
// starts while its previous run is active. Successful overruns log at warning
// level; failures log at error level and retry at the next interval. Shutdown
// cancels all jobs before waiting for active work.
package scheduler
