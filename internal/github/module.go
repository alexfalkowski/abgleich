package github

import "github.com/alexfalkowski/go-service/v2/di"

// Module provides the API client and injects it into the repository adapter.
var Module = di.Module(di.Constructor(NewClient, NewRepository))
