// Package client constructs outbound HTTP clients with independent settings and
// connection pools. TLS resolution, circuit breaking, redirect protection, and
// pool cleanup are shared; callers own credentials, SDKs, and retry policies.
package client
