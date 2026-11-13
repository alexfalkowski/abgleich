package commands

import "github.com/alexfalkowski/go-signal/v2"

// Module provides shared comment dispatch and authorization independently of feature packages.
var Module = signal.Module(signal.Constructor(NewDispatcher, NewAuthorizer))
