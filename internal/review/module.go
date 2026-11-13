package review

import (
	"github.com/alexfalkowski/abgleich/internal/commands"
	"github.com/alexfalkowski/abgleich/internal/review/code"
	"github.com/alexfalkowski/abgleich/internal/review/internal/sources"
	"github.com/alexfalkowski/abgleich/internal/review/internal/workflow"
	"github.com/alexfalkowski/abgleich/internal/review/sec"
	"github.com/alexfalkowski/go-signal/v2"
)

// Module composes optional review commands, their aggregate, and shared source access.
// Each feature registers its own handler and independently scheduled queue.
var Module = signal.Module(
	signal.Constructor(codeConfig, secConfig, sourceConfig),
	sources.Module,
	workflow.Module,
	code.Module,
	sec.Module,
	signal.Register(registerCommand),
)

func registerCommand(dispatcher *commands.Dispatcher, factory *workflow.Factory) error {
	return dispatcher.Register(workflow.ReviewCommand, factory)
}
