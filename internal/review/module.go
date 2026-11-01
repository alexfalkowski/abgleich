package review

import (
	"github.com/alexfalkowski/abgleich/internal/review/code"
	"github.com/alexfalkowski/abgleich/internal/review/internal/workflow"
	"github.com/alexfalkowski/abgleich/internal/review/sec"
	"github.com/alexfalkowski/go-service/v2/di"
)

// Module composes both optional review commands and shared source access.
// Each feature registers its own handler and independently scheduled queue.
var Module = di.Module(
	di.Constructor(codeConfig, secConfig, workflowConfig),
	workflow.Module,
	code.Module,
	sec.Module,
)
