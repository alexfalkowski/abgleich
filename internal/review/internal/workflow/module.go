package workflow

import "github.com/alexfalkowski/go-service/v2/di"

// Module supplies public dependency access shared by independent review commands.
var Module = di.Module(di.Constructor(NewSources))
