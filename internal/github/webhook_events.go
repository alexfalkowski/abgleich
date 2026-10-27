package github

import (
	"github.com/alexfalkowski/go-service/v2/strings"
	"github.com/google/go-github/v93/github"
)

// Event payloads select work only. Every selected repository is rediscovered
// before policy is evaluated, including after duplicate or reordered deliveries.
func (w *Webhook) readEvent(event any) (*Delivery, error) {
	switch event := event.(type) {
	case *github.IssueCommentEvent:
		return w.readComment(event)

	case *github.PullRequestEvent:
		switch event.GetAction() {
		case "opened", "reopened", "synchronize", "ready_for_review", "converted_to_draft", "edited",
			"labeled", "unlabeled", "auto_merge_enabled", "auto_merge_disabled", "closed":
			return w.repositoryDelivery(event.GetInstallation(), event.GetRepo())
		}

	case *github.PushEvent:
		if strings.HasPrefix(event.GetRef(), "refs/heads/") {
			return w.repositoryDelivery(event.GetInstallation(), event.GetRepo())
		}

	case *github.StatusEvent:
		switch event.GetState() {
		case "success", "failure", "error":
			return w.repositoryDelivery(event.GetInstallation(), event.GetRepo())
		}

	case *github.CheckRunEvent:
		if event.GetAction() == "completed" {
			return w.repositoryDelivery(event.GetInstallation(), event.GetRepo())
		}
	}

	return nil, ErrIgnoredDelivery
}

func (w *Webhook) repositoryDelivery(installation *github.Installation, repo repositoryIdentity) (*Delivery, error) {
	if err := w.validateRepository(installation, repo); err != nil {
		return nil, err
	}

	return &Delivery{Repository: strings.ToLower(repo.GetFullName())}, nil
}
