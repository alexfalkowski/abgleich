package ai

import "github.com/alexfalkowski/go-signal/v2"

// Module resolves optional provider credentials before handlers start.
var Module = signal.Module(signal.Constructor(NewProvider))
