package client

import "github.com/alexfalkowski/go-service/v2/di"

// Module provides a factory without sharing client settings or pools between callers.
var Module = di.Module(di.Constructor(NewFactory))
