package golang

import "github.com/alexfalkowski/go-signal/v2"

// Module contributes the Go strategy to the review source DI group.
var Module = signal.Module(signal.Constructor(NewProvider))
