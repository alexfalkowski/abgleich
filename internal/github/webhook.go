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

// ErrIgnoredDelivery identifies a verified event or action with no worker handler.
var ErrIgnoredDelivery = errors.New("github: delivery does not require processing")

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

// Delivery routes a verified comment command or repository notification.
// Repository is only a discovery hint; current settings and PR state must be read
// from GitHub before automation. Comment is nil for repository notifications.
type Delivery struct {
	Comment    *IssueCommentEvent
	Repository string
}

type repositoryIdentity interface {
	GetOwner() *github.User
	GetName() string
	GetFullName() string
}

// Read verifies SHA-256 signatures and installation/repository identity before
// routing supported events. Unknown event types and actions are ignored.
func (w *Webhook) Read(req *http.Request) (*Delivery, error) {
	signature := req.Header.Get("X-Hub-Signature-256")
	if !strings.HasPrefix(signature, "sha256=") {
		return nil, status.Error(http.StatusUnauthorized, "github: missing GitHub SHA-256 signature")
	}

	payload, err := github.ValidatePayload(req, w.secret)
	if err != nil {
		return nil, status.SafeError(http.StatusUnauthorized, err)
	}

	kind := github.WebHookType(req)
	switch kind {
	case "issue_comment", "pull_request", "push", "status", "check_run":
	default:
		return nil, ErrIgnoredDelivery
	}

	event, err := github.ParseWebHook(kind, payload)
	if err != nil {
		return nil, status.SafeError(http.StatusBadRequest, err)
	}

	return w.readEvent(event)
}

func (w *Webhook) readComment(event *IssueCommentEvent) (*Delivery, error) {
	issue := event.GetIssue()
	if issue == nil {
		return nil, status.Error(http.StatusBadRequest, "github: missing GitHub issue")
	}

	isNewComment := event.GetAction() == "created"
	if !isNewComment || !issue.IsPullRequest() {
		return nil, ErrIgnoredDelivery
	}

	if err := w.validateComment(event); err != nil {
		return nil, err
	}

	return &Delivery{Comment: event}, nil
}

func (w *Webhook) validateComment(event *IssueCommentEvent) error {
	if err := w.validateRepository(event.GetInstallation(), event.GetRepo()); err != nil {
		return err
	}

	comment := event.GetComment()
	hasIssueNumber := event.GetIssue().GetNumber() > 0
	hasCommentID := comment.GetID() > 0
	hasAuthorID := comment.GetUser().GetID() > 0
	hasCreatedAt := !comment.GetCreatedAt().IsZero()
	if !hasIssueNumber || !hasCommentID || !hasAuthorID || !hasCreatedAt {
		return status.Error(http.StatusBadRequest, "github: incomplete GitHub comment")
	}

	return nil
}

func (w *Webhook) validateRepository(installation *github.Installation, repo repositoryIdentity) error {
	sameInstallation := installation.GetID() == w.config.App.InstallationID
	if !sameInstallation {
		return status.Error(http.StatusForbidden, "github: unexpected GitHub installation")
	}

	expectedFullName := w.config.Owner + "/" + repo.GetName()
	sameOwner := strings.ToLower(repo.GetOwner().GetLogin()) == strings.ToLower(w.config.Owner)
	sameFullName := strings.ToLower(repo.GetFullName()) == strings.ToLower(expectedFullName)
	hasName := !strings.IsEmpty(repo.GetName())
	if !sameOwner || !sameFullName || !hasName {
		return status.Error(http.StatusForbidden, "github: unexpected GitHub repository")
	}

	return nil
}
