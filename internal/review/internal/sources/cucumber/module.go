package cucumber

import "github.com/alexfalkowski/go-signal/v2"

// Module contributes framework guidance to the review source DI group.
var Module = signal.Module(signal.Constructor(NewProvider))
