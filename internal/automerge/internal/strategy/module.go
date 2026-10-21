package strategy

import "github.com/alexfalkowski/go-service/v2/di"

// Module provides strategies in Dependabot, label, and rebase precedence order.
var Module = di.Module(di.Constructor(strategies))
