package github

import (
	"github.com/alexfalkowski/go-service/v2/errors"
	"github.com/alexfalkowski/go-service/v2/net/http"
	"github.com/alexfalkowski/go-service/v2/net/http/status"
	"github.com/alexfalkowski/go-service/v2/os"
	"github.com/alexfalkowski/go-service/v2/strings"
	"github.com/alexfalkowski/go-service/v2/time"
	"github.com/google/go-github/v93/github"
)

// ErrIgnoredDelivery identifies a verified delivery that needs no command action.
var ErrIgnoredDelivery = errors.New("github: delivery does not contain a new PR comment")

// NewWebhook resolves optional delivery credentials at startup, returning nil
// when webhook configuration is disabled. A configured webhook requires App
// credentials, a nonempty secret, and a timeout below ten seconds.
func NewWebhook(cfg *Config, filesystem *os.FS) (*Webhook, error) {
	if !cfg.Webhook.IsEnabled() {
		return nil, nil
	}

	if !cfg.Webhook.IsValid() {
		return nil, errors.New("github: webhook timeout must be positive and below ten seconds")
	}

	secret, err := filesystem.ReadSource(cfg.Webhook.Secret)
	if err != nil {
		return nil, errors.Prefix("github: webhook secret", err)
	}

	if len(secret) == 0 {
		return nil, errors.New("github: webhook secret is empty")
	}

	return &Webhook{config: cfg, secret: secret}, nil
}

// Webhook verifies raw GitHub deliveries before exposing SDK events to policies.
// It holds no delivery history; policy must re-read current remote state.
type Webhook struct {
	config *Config
	secret []byte
}

// GetTimeout returns the independent synchronous processing budget.
func (w *Webhook) GetTimeout() time.Duration { return w.config.Webhook.GetTimeout() }

// Read verifies SHA-256 signatures and installation/repository identity. Signed
// events other than newly created PR comments are acknowledged without action.
func (w *Webhook) Read(req *http.Request) (*IssueCommentEvent, error) {
	if !strings.HasPrefix(req.Header.Get("X-Hub-Signature-256"), "sha256=") {
		return nil, status.Error(http.StatusUnauthorized, "github: missing GitHub SHA-256 signature")
	}

	payload, err := github.ValidatePayload(req, w.secret)
	if err != nil {
		return nil, status.SafeError(http.StatusUnauthorized, err)
	}

	if github.WebHookType(req) != "issue_comment" {
		return nil, ErrIgnoredDelivery
	}

	event, err := github.ParseWebHook("issue_comment", payload)
	if err != nil {
		return nil, status.SafeError(http.StatusBadRequest, err)
	}

	comment, ok := event.(*github.IssueCommentEvent)
	if !ok {
		return nil, status.Error(http.StatusBadRequest, "github: unexpected GitHub event type")
	}

	if comment.GetIssue() == nil {
		return nil, status.Error(http.StatusBadRequest, "github: missing GitHub issue")
	}

	if comment.GetAction() != "created" || !comment.GetIssue().IsPullRequest() {
		return nil, ErrIgnoredDelivery
	}

	if err := w.validateComment(comment); err != nil {
		return nil, err
	}

	return comment, nil
}

func (w *Webhook) validateComment(comment *IssueCommentEvent) error {
	if comment.GetInstallation().GetID() != w.config.App.InstallationID {
		return status.Error(http.StatusForbidden, "github: unexpected GitHub installation")
	}

	repo := comment.GetRepo()

	if strings.ToLower(repo.GetOwner().GetLogin()) != strings.ToLower(w.config.Owner) ||
		strings.ToLower(repo.GetFullName()) != strings.ToLower(w.config.Owner+"/"+repo.GetName()) || strings.IsEmpty(repo.GetName()) {
		return status.Error(http.StatusForbidden, "github: unexpected GitHub repository")
	}

	if comment.GetIssue().GetNumber() <= 0 || comment.GetComment().GetID() <= 0 ||
		comment.GetComment().GetUser().GetID() <= 0 || comment.GetComment().GetCreatedAt().IsZero() {
		return status.Error(http.StatusBadRequest, "github: incomplete GitHub comment")
	}

	return nil
}
