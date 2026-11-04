package workflow

import (
	"github.com/alexfalkowski/abgleich/internal/ai"
	"github.com/alexfalkowski/go-service/v2/errors"
)

type reviewFailure struct {
	Provider  *ai.GenerationError
	Title     string
	Command   string
	Model     string
	HeadSHA   string
	BaseSHA   string
	Stage     string
	Reason    string
	Recovery  string
	RequestID int64
}

func (c *Command) failureReply(task *reviewTask, stage string, err error) (string, error) {
	failure := reviewFailure{
		Title: c.definition.Title, Command: c.definition.Name, Model: c.definition.Model,
		RequestID: task.commandComment.GetID(), HeadSHA: task.pullRequest.GetHead().GetSHA(), BaseSHA: task.pullRequest.GetBase().GetSHA(),
		Stage: stage, Reason: "The review conversation could not finish.",
		Recovery: "Check the worker logs for source lookup or authorization failures.",
	}
	switch stage {
	case "Source collection":
		failure.Reason = "The pull request source could not be collected or validated."
		failure.Recovery = "Check GitHub availability, App source permissions, and source validation errors in the worker logs."
	case "Result validation":
		failure.Reason = "The model returned an unreadable or invalid review result."
		failure.Recovery = "Check the model's review output format before requesting another review."
	}
	if provider, ok := errors.AsType[*ai.GenerationError](err); ok {
		failure.Provider = provider
		failure.Reason, failure.Recovery = provider.Reason, provider.Recovery
	}
	return render("failure.md.tmpl", failure)
}
