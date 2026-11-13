package github

import "github.com/alexfalkowski/go-signal/v2"

// Module provides shared API access and owner-wide discovery.
var Module = signal.Module(signal.Constructor(NewConnection, NewClient, NewService, NewWebhook))
