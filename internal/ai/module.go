package ai

import "github.com/alexfalkowski/go-service/v2/di"

// Module resolves optional provider credentials before handlers start.
var Module = di.Module(di.Constructor(NewProvider))
