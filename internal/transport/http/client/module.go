package client

import "github.com/alexfalkowski/go-signal/v2"

// Module provides a factory without sharing client settings or pools between callers.
var Module = signal.Module(signal.Constructor(NewFactory))
