package commands

import "github.com/alexfalkowski/go-service/v2/errors"

// ErrCommandRegistered rejects duplicate command names during registration.
var ErrCommandRegistered = errors.New("command already registered")

// ErrCommandInProgress rejects concurrent commands while another is running.
var ErrCommandInProgress = errors.New("another GitHub command is being processed")

// ErrWriteAccessRequired rejects a command whose author lacks repository write access.
var ErrWriteAccessRequired = errors.New("the comment author needs repository write access")

// ErrPullRequestChanged rejects a command whose target pull request identity or
// head changes during processing.
var ErrPullRequestChanged = errors.New("pull request changed while processing the command")
