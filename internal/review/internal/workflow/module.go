package workflow

import "github.com/alexfalkowski/go-service/v2/di"

// Module supplies the factory shared by review features.
var Module = di.Module(di.Constructor(NewFactory))
