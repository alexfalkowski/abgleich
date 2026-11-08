package ai

import (
	"fmt"
	"net"
	"regexp"

	"github.com/alexfalkowski/go-service/v2/context"
	"github.com/alexfalkowski/go-service/v2/errors"
	"github.com/alexfalkowski/go-service/v2/net/http"
	"github.com/alexfalkowski/go-service/v2/strings"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/responses"
)

// ProviderErrorCode covers HTTP errors, failed responses, and incomplete-response reasons.
// Generation diagnostics retain only codes accepted by the local allowlist.
type ProviderErrorCode string

const (
	codeContextLimit      ProviderErrorCode = "context_length_exceeded"
	codeInvalidReasoning  ProviderErrorCode = "invalid_encrypted_content"
	codeInvalidKey        ProviderErrorCode = "invalid_api_key"
	codeModelNotFound     ProviderErrorCode = "model_not_found"
	codeRateLimit         ProviderErrorCode = ProviderErrorCode(responses.ResponseErrorCodeRateLimitExceeded)
	codeInsufficientQuota ProviderErrorCode = "insufficient_quota"
	codeBillingLimit      ProviderErrorCode = "billing_hard_limit_reached"
	codeServerError       ProviderErrorCode = ProviderErrorCode(responses.ResponseErrorCodeServerError)
	codeServerOverloaded  ProviderErrorCode = "server_is_overloaded"
	codeSlowDown          ProviderErrorCode = "slow_down"
	codeOrgSpendLimit     ProviderErrorCode = "organization_spend_limit_exceeded"
	codeProjectSpendLimit ProviderErrorCode = "project_spend_limit_exceeded"
	codeOrgUsageLimit     ProviderErrorCode = "organization_usage_limit_exceeded"
	codeInvalidPrompt     ProviderErrorCode = ProviderErrorCode(responses.ResponseErrorCodeInvalidPrompt)
	codeContentFilter     ProviderErrorCode = "content_filter"
	codeMaxOutputTokens   ProviderErrorCode = "max_output_tokens"
	codeMaxMessages       ProviderErrorCode = "max_messages"
)

// String returns the provider diagnostic code without changing its protocol value.
func (c ProviderErrorCode) String() string {
	return string(c)
}

// IsDefined reports whether a diagnostic code is present, without validating it.
func (c ProviderErrorCode) IsDefined() bool {
	return !strings.IsEmpty(c.String())
}

// ProviderErrorType identifies HTTP error categories retained in generation diagnostics.
type ProviderErrorType string

const (
	typeInvalidRequest    ProviderErrorType = "invalid_request_error"
	typeAuthentication    ProviderErrorType = "authentication_error"
	typePermission        ProviderErrorType = "permission_error"
	typeRateLimit         ProviderErrorType = "rate_limit_error"
	typeServer            ProviderErrorType = "server_error"
	typeInsufficientQuota ProviderErrorType = "insufficient_quota"
)

// String returns the provider's HTTP error category.
func (t ProviderErrorType) String() string {
	return string(t)
}

// IsDefined reports whether an error category is present, without validating it.
func (t ProviderErrorType) IsDefined() bool {
	return !strings.IsEmpty(t.String())
}

var providerRequestID = regexp.MustCompile(`^req_[a-fA-F0-9]{32}$`)

func requestFailure(err error) *GenerationError {
	failure := &GenerationError{
		Reason:   "The AI request failed before a complete response was received.",
		Recovery: "Inspect the worker's AI generation diagnostics for the failed request.",
	}

	if response, ok := errors.AsType[*openai.Error](err); ok {
		failure.StatusCode = response.StatusCode
		failure.Code = knownProviderCode(response.Code)

		switch kind := ProviderErrorType(response.Type); kind {
		case typeInvalidRequest, typeAuthentication, typePermission,
			typeRateLimit, typeServer, typeInsufficientQuota:
			failure.Type = kind
		}
		if response.Response != nil {
			id := response.Response.Header.Get("X-Request-ID")
			if providerRequestID.MatchString(id) {
				failure.RequestID = id
			}
		}

		failure.classifyHTTP()
		failure.classifyCode()

		return failure
	}

	switch {
	case errors.Is(err, context.DeadlineExceeded):
		failure.Reason = "The AI request exceeded its deadline."
		failure.Recovery = "Check the AI HTTP timeout and the inference and job budgets."
	case errors.Is(err, context.Canceled):
		failure.Reason = "The AI request was canceled."
		failure.Recovery = "Check whether the worker was shutting down or the job was canceled."
	default:
		if transport, ok := errors.AsType[net.Error](err); ok {
			failure.Reason = "The connection to the AI provider failed."
			failure.Recovery = "Check network connectivity, DNS, TLS, and the configured AI endpoint."
			if transport.Timeout() {
				failure.Reason = "The AI HTTP request timed out."
				failure.Recovery = "Check the AI HTTP timeout and provider availability."
			}
		}
	}

	return failure
}

// GenerationError exposes locally classified diagnostics for logs and advisory replies.
// Provider messages, request bodies, URLs, and credentials are never retained.
// Unrecognized provider codes/types and invalid request IDs are omitted.
type GenerationError struct {
	Reason     string
	Recovery   string
	Code       ProviderErrorCode
	Type       ProviderErrorType
	RequestID  string
	StatusCode int
}

// Error includes only the safe diagnostics retained by the provider adapter.
func (f *GenerationError) Error() string {
	return fmt.Sprintf("ai: %s (http_status=%d provider_code=%q provider_type=%q provider_request_id=%q)",
		f.Reason, f.StatusCode, f.Code, f.Type, f.RequestID)
}

func (f *GenerationError) classifyHTTP() {
	switch f.StatusCode {
	case http.StatusBadRequest, http.StatusUnprocessableEntity:
		f.Reason = "The AI provider rejected the request."
		f.Recovery = "Check the provider code, model settings, and conversation input."
	case http.StatusUnauthorized:
		f.Reason = "The AI provider rejected its credentials."
		f.Recovery = "Check the configured AI credentials and restart the worker after replacing them."
	case http.StatusForbidden:
		f.Reason = "The AI provider denied access."
		f.Recovery = "Check the project's model permissions and provider access policies."
	case http.StatusNotFound:
		f.Reason = "The AI model or endpoint was not found."
		f.Recovery = "Check the configured model, AI endpoint, and project access."
	case http.StatusRequestTimeout, http.StatusGatewayTimeout:
		f.Reason = "The AI provider timed out while handling the request."
		f.Recovery = "Check provider availability and the AI request timeout."
	case http.StatusRequestEntityTooLarge:
		f.Reason = "The AI provider rejected the request as too large."
		f.Recovery = "Use a smaller contribution or a model with sufficient context capacity."
	case http.StatusTooManyRequests:
		f.Reason = "The AI provider rate limited the request."
		f.Recovery = "Check the project's rate limits and allow capacity to recover."
	default:
		if f.StatusCode >= http.StatusInternalServerError {
			f.Reason = "The AI provider encountered a server error."
			f.Recovery = "Check provider availability before sending another request."
		}
	}
}

func (f *GenerationError) classifyCode() {
	switch f.Code {
	case codeContextLimit:
		f.Reason = "The supplied source exceeded the model's context window."
		f.Recovery = "Use a smaller contribution or a model with sufficient context capacity."
	case codeInsufficientQuota, codeBillingLimit, codeOrgSpendLimit,
		codeProjectSpendLimit, codeOrgUsageLimit:
		f.Reason = "The AI project's quota or spending limit was reached."
		f.Recovery = "Check the provider project's available quota and spending limits."
	case codeInvalidReasoning:
		f.Reason = "The AI provider rejected the conversation's encrypted reasoning state."
		f.Recovery = "Check the provider's stateless conversation handling and the worker's conversation diagnostics."
	case codeInvalidKey, codeModelNotFound, codeRateLimit, codeServerError,
		codeServerOverloaded, codeSlowDown, codeInvalidPrompt, codeContentFilter,
		codeMaxOutputTokens, codeMaxMessages:
		// These codes retain the HTTP or generic failure classification.
		return
	}
}

func knownProviderCode(code string) ProviderErrorCode {
	// These are protocol identifiers, never arbitrary provider text echoed from input.
	providerCode := ProviderErrorCode(code)
	switch providerCode {
	case codeContextLimit, codeInvalidReasoning, codeInvalidKey,
		codeModelNotFound, codeRateLimit, codeInsufficientQuota,
		codeBillingLimit, codeServerError, codeServerOverloaded,
		codeSlowDown, codeOrgSpendLimit, codeProjectSpendLimit,
		codeOrgUsageLimit, codeInvalidPrompt, codeContentFilter,
		codeMaxOutputTokens, codeMaxMessages:
		return providerCode
	default:
		return ""
	}
}
