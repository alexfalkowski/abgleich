# Accepted design for the runtime

Part of the [accepted-design index](../accepted-design.md).

## Comment command overlap

Scope webhook overlap protection to each registered serial command name.
Different serial commands can run at the same time, including on the same PR.
Concurrent review commands share a separate pair of delivery slots.

[Webhook delivery](../../commands.md#-webhook-delivery)

## Job registration and scheduling

Each feature module registers its own named job before startup through
`Scheduler.Register` or `Scheduler.RegisterTriggered`. Names are unique.
Keep the scheduler independent of the feature packages.

Jobs start immediately and then run on independent schedules with per-job
deadlines and overlap protection. Skip scheduled starts missed during active
work. Review queue notifications wake the existing job and share its deadline
and overlap protection. Thread maintenance runs on the schedule.

[Scheduler](../../../internal/scheduler/scheduler.go) ·
[Job loop](../../../internal/scheduler/job.go) ·
[Scheduling scenarios](../../../test/features/automerge/schedule.feature)

## Shutdown and health

Handlers observe cancellation and return the complete outcome. Use
`go-sync` for synchronization and error aggregation. Shutdown cancels all
jobs before waiting and preserves shutdown failures with their job names.
Keep lifecycle ordering so jobs stop before outbound client transports close.

Readiness and liveness describe the process independently of job failures,
conflicts, or timeouts. Keep structured job-outcome and overrun logs for
monitoring.

[Health observers](../../../internal/health/health.go) ·
[Health scenarios](../../../test/features/health/observability.feature) ·
[Operations guide](../../operations.md)
