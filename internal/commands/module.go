package commands

import "github.com/alexfalkowski/go-service/v2/di"

// Module provides shared comment dispatch independently of feature packages.
var Module = di.Module(di.Constructor(NewDispatcher))
