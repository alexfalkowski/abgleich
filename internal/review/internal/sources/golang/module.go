package golang

import "github.com/alexfalkowski/go-service/v2/di"

// Module contributes the Go strategy to the review source DI group.
var Module = di.Module(di.Constructor(NewProvider))
