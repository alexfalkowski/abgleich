package ai

import (
	"fmt"
	"net"
	"regexp"

	"github.com/alexfalkowski/go-service/v2/context"
	"github.com/alexfalkowski/go-service/v2/errors"
	"github.com/alexfalkowski/go-service/v2/net/http"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/responses"
)

const (
	codeContextLimit      = "context_length_exceeded"
	codeInvalidReasoning  = "invalid_encrypted_content"
	codeInvalidKey        = "invalid_api_key"
	codeModelNotFound     = "model_not_found"
	codeRateLimit         = string(responses.ResponseErrorCodeRateLimitExceeded)
	codeInsufficientQuota = "insufficient_quota"
	codeBillingLimit      = "billing_hard_limit_reached"
	codeServerError       = string(responses.ResponseErrorCodeServerError)
	codeServerOverloaded  = "server_is_overloaded"
	codeSlowDown          = "slow_down"
	codeOrgSpendLimit     = "organization_spend_limit_exceeded"
	codeProjectSpendLimit = "project_spend_limit_exceeded"
	codeOrgUsageLimit     = "organization_usage_limit_exceeded"
	codeInvalidPrompt     = string(responses.ResponseErrorCodeInvalidPrompt)
	codeContentFilter     = "content_filter"
	codeMaxOutputTokens   = "max_output_tokens"
	codeMaxMessages       = "max_messages"
)

const (
	typeInvalidRequest    = "invalid_request_error"
	typeAuthentication    = "authentication_error"
	typePermission        = "permission_error"
	typeRateLimit         = "rate_limit_error"
	typeServer            = "server_error"
	typeInsufficientQuota = "insufficient_quota"
)

var providerRequestID = regexp.MustCompile(`^req_[a-fA-F0-9]{32}$`)

func requestFailure(err error) *GenerationError {
	failure := &GenerationError{
		Reason:   "The AI request failed before a complete response was received.",
		Recovery: "Inspect the worker's AI generation diagnostics for the failed request.",
	}

	if response, ok := errors.AsType[*openai.Error](err); ok {
		failure.StatusCode = response.StatusCode
		failure.Code = knownProviderCode(response.Code)

		switch response.Type {
		case typeInvalidRequest, typeAuthentication, typePermission,
			typeRateLimit, typeServer, typeInsufficientQuota:
			failure.Type = response.Type
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
	Code       string
	Type       string
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
	}
}

func knownProviderCode(code string) string {
	// These are protocol identifiers, never arbitrary provider text echoed from input.
	switch code {
	case codeContextLimit, codeInvalidReasoning, codeInvalidKey,
		codeModelNotFound, codeRateLimit, codeInsufficientQuota,
		codeBillingLimit, codeServerError, codeServerOverloaded,
		codeSlowDown, codeOrgSpendLimit, codeProjectSpendLimit,
		codeOrgUsageLimit, codeInvalidPrompt, codeContentFilter,
		codeMaxOutputTokens, codeMaxMessages:
		return code
	default:
		return ""
	}
}
