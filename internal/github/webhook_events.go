package github

import (
	"github.com/alexfalkowski/go-service/v3/strings"
	"github.com/google/go-github/v93/github"
)

const (
	actionOpened            = "opened"
	actionReopened          = "reopened"
	actionSynchronize       = "synchronize"
	actionReadyForReview    = "ready_for_review"
	actionConvertedToDraft  = "converted_to_draft"
	actionEdited            = "edited"
	actionLabeled           = "labeled"
	actionUnlabeled         = "unlabeled"
	actionAutoMergeEnabled  = "auto_merge_enabled"
	actionAutoMergeDisabled = "auto_merge_disabled"
	actionClosed            = "closed"
	checkRunCompleted       = "completed"
)

// Event payloads select work only. Every selected repository is rediscovered
// before policy is evaluated, including after duplicate or reordered deliveries.
func (w *Webhook) readEvent(event any) (*Delivery, error) {
	switch event := event.(type) {
	case *github.IssueCommentEvent:
		return w.readComment(event)

	case *github.PullRequestEvent:
		switch event.GetAction() {
		case actionOpened, actionReopened, actionSynchronize,
			actionReadyForReview, actionConvertedToDraft, actionEdited,
			actionLabeled, actionUnlabeled, actionAutoMergeEnabled,
			actionAutoMergeDisabled, actionClosed:
			return w.repositoryDelivery(event.GetInstallation(), event.GetRepo())
		}

	case *github.PushEvent:
		if strings.HasPrefix(event.GetRef(), branchPrefix) {
			return w.repositoryDelivery(event.GetInstallation(), event.GetRepo())
		}

	case *github.StatusEvent:
		switch event.GetState() {
		case StatusSuccess, StatusFailure, StatusError:
			return w.repositoryDelivery(event.GetInstallation(), event.GetRepo())
		}

	case *github.CheckRunEvent:
		if event.GetAction() == checkRunCompleted {
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
