package commands

import "github.com/alexfalkowski/go-service/v3/errors"

// ErrCommandRegistered rejects duplicate command names during registration.
var ErrCommandRegistered = errors.New("commands: command already registered")

// ErrCommandInProgress reports exhausted delivery slots or a full command queue.
var ErrCommandInProgress = errors.New("commands: another GitHub command is being processed")

// ErrWriteAccessRequired rejects a command whose author lacks repository write access.
var ErrWriteAccessRequired = errors.New("commands: the comment author needs repository write access")

// ErrPullRequestChanged rejects a command whose target pull request identity or
// head changes during processing.
var ErrPullRequestChanged = errors.New("commands: pull request changed while processing the command")
