package summarise

import (
	"github.com/alexfalkowski/abgleich/internal/commands"
	"github.com/alexfalkowski/abgleich/internal/scheduler"
	"github.com/alexfalkowski/go-signal/v2"
)

// Module registers the command and, when configured, its independent job.
var Module = signal.Module(signal.Constructor(NewCommand), signal.Register(registerCommand, registerSchedule))

func registerCommand(dispatcher *commands.Dispatcher, command *Command) error {
	return dispatcher.Register(summaryCommand, command)
}

func registerSchedule(schedules *scheduler.Scheduler, command *Command) error {
	if command == nil {
		return nil
	}

	return schedules.Register(summaryCommand, command.config, command.queue.Process)
}
