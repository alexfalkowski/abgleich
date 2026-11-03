package github

import "github.com/alexfalkowski/go-service/v2/errors"

// Errors contains GraphQL failures that may accompany data even on HTTP success.
type Errors []struct {
	// Message contains the failure explanation included in the joined service error.
	Message string `json:"message"`
	// Type classifies the GraphQL failure, including RATE_LIMITED responses.
	Type string `json:"type"`
}

// Validate joins reported failures and includes [ErrRateLimited] for rate limits.
// It returns nil when Errors is empty or nil.
func (e Errors) Validate() error {
	var failures error
	for _, failure := range e {
		failures = errors.Join(failures, errors.New("github: "+failure.Message))

		if failure.Type == "RATE_LIMITED" || rateLimitMessage(failure.Message) {
			failures = errors.Join(failures, ErrRateLimited)
		}
	}

	return failures
}
