package github

import "github.com/alexfalkowski/go-service/v2/di"

// Module provides the API client and repository composed with rebase logging.
var Module = di.Module(di.Constructor(NewClient, NewRepository))
