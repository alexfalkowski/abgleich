package cucumber

import "github.com/alexfalkowski/go-service/v2/di"

// Module contributes framework guidance to the review source DI group.
var Module = di.Module(di.Constructor(NewProvider))
