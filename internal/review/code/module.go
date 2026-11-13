package code

import (
	"github.com/alexfalkowski/abgleich/internal/commands"
	"github.com/alexfalkowski/abgleich/internal/scheduler"
	"github.com/alexfalkowski/go-signal/v2"
)

// Module registers the exact command and its optional independent queue job.
var Module = signal.Module(signal.Constructor(NewCommand), signal.Register(registerCommand, registerSchedule))

func registerCommand(dispatcher *commands.Dispatcher, command *Command) error {
	return dispatcher.Register(codeReviewCommand, command)
}

func registerSchedule(schedules *scheduler.Scheduler, command *Command) error {
	if command == nil {
		return nil
	}

	return schedules.RegisterTriggered(codeReviewCommand, command.config, command)
}
