package provider

import (
	"net/url"

	"github.com/alexfalkowski/go-service/v3/strings"
	"github.com/alexfalkowski/go-service/v3/time"
)

// RequestTimeout bounds each external source HTTP request independently of inference.
const RequestTimeout = 10 * time.Second

// ValidEndpoint permits trusted HTTP(S) bases without credentials, queries, or fragments.
func ValidEndpoint(raw string) bool {
	endpoint, err := url.Parse(raw)
	if err != nil {
		return false
	}
	if strings.IsEmpty(endpoint.Host) || endpoint.User != nil {
		return false
	}
	if !strings.IsEmpty(endpoint.RawQuery) || !strings.IsEmpty(endpoint.Fragment) {
		return false
	}

	return endpoint.Scheme == "https" || endpoint.Scheme == "http"
}
