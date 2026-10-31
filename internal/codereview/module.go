package codereview

import (
	"github.com/alexfalkowski/abgleich/internal/commands"
	"github.com/alexfalkowski/abgleich/internal/scheduler"
	"github.com/alexfalkowski/go-service/v2/di"
)

// Module registers the exact command and its optional independent queue job.
var Module = di.Module(di.Constructor(NewCommand), di.Register(registerCommand, registerSchedule))

func registerCommand(dispatcher *commands.Dispatcher, command *Command) error {
	return dispatcher.Register(codeReviewCommand, command)
}

func registerSchedule(schedules *scheduler.Scheduler, command *Command) error {
	if command.config == nil {
		return nil
	}

	return schedules.Register(codeReviewCommand, command.config, command.queue.Process)
}
