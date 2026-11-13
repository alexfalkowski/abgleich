package status

import (
	"github.com/alexfalkowski/abgleich/internal/commands"
	"github.com/alexfalkowski/go-signal/v2"
)

// Module registers the status comment command before startup.
var Module = signal.Module(
	signal.Constructor(NewCommand),
	signal.Register(registerCommand),
)

func registerCommand(dispatcher *commands.Dispatcher, command *Command) error {
	return dispatcher.Register(statusCommand, command)
}
