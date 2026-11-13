package workflow

import "github.com/alexfalkowski/go-signal/v2"

// Module supplies the factory shared by review features.
var Module = signal.Module(signal.Constructor(NewFactory))
