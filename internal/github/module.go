package github

import "github.com/alexfalkowski/go-service/v2/di"

// Module provides shared API access and owner-wide discovery.
var Module = di.Module(di.Constructor(NewConnection, NewClient, NewService, NewWebhook))
