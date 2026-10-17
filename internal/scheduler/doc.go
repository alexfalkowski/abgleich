// Package scheduler runs independently scheduled jobs and logs their outcomes
// and durations. Jobs start at startup and at their configured interval,
// ignoring attempts while the previous run is active. Successful overruns log at
// warning level; failures log at error level. Failed jobs retry at the next interval.
package scheduler
