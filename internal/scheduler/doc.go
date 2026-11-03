// Package scheduler runs independently scheduled jobs and logs their outcomes
// and durations. Feature packages register named handlers with Scheduler before
// startup. Each job runs immediately and at its configured interval, skipping
// starts while its previous run is active. Successful overruns log at warning
// level. Optional queue notifications coalesce and run a separate handler as soon
// as the job is free, using the same timeout and overlap protection. Failures log
// at error level and scheduled recovery runs at the next interval. Shutdown
// cancels all jobs before waiting for active work.
package scheduler
