package status

import (
	"github.com/alexfalkowski/abgleich/internal/commands"
	"github.com/alexfalkowski/go-service/v2/di"
)

// Module registers the status comment command before startup.
var Module = di.Module(
	di.Constructor(NewCommand),
	di.Register(registerCommand),
)

func registerCommand(dispatcher *commands.Dispatcher, command *Command) error {
	return dispatcher.Register(statusCommand, command)
}
