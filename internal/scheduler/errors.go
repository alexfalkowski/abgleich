package scheduler

import "github.com/alexfalkowski/go-service/v2/errors"

// ErrJobRegistered is returned when a job name is already registered.
var ErrJobRegistered = errors.New("scheduler: job already registered")
