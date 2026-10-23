package automerge

import "github.com/alexfalkowski/go-service/v2/errors"

// ErrCommandInProgress rejects concurrent commands while another is running.
var ErrCommandInProgress = errors.New("another GitHub command is being processed")

// ErrWriteAccessRequired rejects a command whose author lacks repository write access.
var ErrWriteAccessRequired = errors.New("the comment author needs repository write access")

// ErrPullRequestChanged rejects labeling when the pull request identity or head
// changes during command processing.
var ErrPullRequestChanged = errors.New("pull request changed while processing the command")
