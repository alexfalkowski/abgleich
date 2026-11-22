package provider

import (
	"github.com/alexfalkowski/go-service/v3/net/url"
	"github.com/alexfalkowski/go-service/v3/strings"
	"github.com/alexfalkowski/go-service/v3/time"
)

// RequestTimeout bounds each external source HTTP request independently of inference.
const RequestTimeout = 10 * time.Second

// ValidEndpoint requires a configured secure base URL without credentials, queries, or fragments.
func ValidEndpoint(base *url.SecureURL) bool {
	if base == nil {
		return false
	}

	endpoint := base.URL()
	return endpoint != nil && endpoint.User == nil && !endpoint.ForceQuery &&
		strings.IsEmpty(endpoint.RawQuery) && strings.IsEmpty(endpoint.Fragment)
}
