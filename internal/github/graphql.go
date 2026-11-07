package github

import (
	"github.com/alexfalkowski/go-service/v2/context"
	"github.com/alexfalkowski/go-service/v2/errors"
	"github.com/alexfalkowski/go-service/v2/net/http"
	"github.com/alexfalkowski/go-service/v2/net/http/media"
)

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
		failures = errors.Join(failures, errors.Prefix("github", errors.New(failure.Message)))

		if failure.Type == "RATE_LIMITED" || rateLimitMessage(failure.Message) {
			failures = errors.Join(failures, ErrRateLimited)
		}
	}

	return failures
}

// graphql sends one operation through the installation client. Callers validate
// GraphQL errors and response identity before accepting the result.
func (s *service) graphql(ctx context.Context, body, response any) error {
	request, err := s.client.installation.NewRequest(ctx, http.MethodPost, "graphql", body)
	if err != nil {
		return err
	}

	request.Header.Set(http.AcceptKey, media.JSON)
	_, err = s.client.installation.Do(request, response)
	return err
}
