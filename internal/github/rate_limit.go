package github

import (
	"github.com/alexfalkowski/go-service/v2/errors"
	"github.com/alexfalkowski/go-service/v2/net/http"
	"github.com/alexfalkowski/go-service/v2/strings"
	"github.com/google/go-github/v93/github"
)

// ErrRateLimited identifies a rate-limit error in a GraphQL HTTP-success payload.
var ErrRateLimited = errors.New("github rate limit reached")

// IsRateLimited recognizes throttling so reconciliation stops the current sweep
// instead of making more requests for other repositories against the same limit.
func IsRateLimited(err error) bool {
	if errors.Is(err, ErrRateLimited) {
		return true
	}

	if _, ok := errors.AsType[*github.RateLimitError](err); ok {
		return true
	}

	if _, ok := errors.AsType[*github.AbuseRateLimitError](err); ok {
		return true
	}

	response, ok := errors.AsType[*github.ErrorResponse](err)
	if !ok || response.Response == nil {
		return false
	}

	return response.Response.StatusCode == http.StatusTooManyRequests ||
		(response.Response.StatusCode == http.StatusForbidden && rateLimitMessage(response.Message))
}

func rateLimitMessage(message string) bool {
	message = strings.ToLower(message)
	return strings.Contains(message, "rate limit") || strings.Contains(message, "rate_limit")
}
