package automerge

import (
	"fmt"

	"github.com/alexfalkowski/go-service/v2/context"
	"github.com/alexfalkowski/go-service/v2/errors"
	"github.com/alexfalkowski/go-service/v2/strings"
	"github.com/alexfalkowski/go-service/v2/telemetry/logger"
)

// NewRebaser constructs a rebaser for the full repository name. The repository
// supplies fresh snapshots on each invocation, allowing recovery after failures
// without keeping local checkpoints.
func NewRebaser(name string, repository GitRepository, log *logger.Logger) *Rebaser {
	return &Rebaser{name: strings.ToLower(name), repository: repository, logger: log}
}

// Rebaser keeps non-draft, same-repository auto-merge branches current with
// their bases. It continues processing other pull requests after a failure.
type Rebaser struct {
	repository GitRepository
	logger     *logger.Logger
	name       string
}

// Rebase discovers current pull requests and requests rebases for stale eligible
// branches. It returns joined failures and respects cancellation. Mutations are
// attempted once per pull request; a later invocation discovers remote state
// after partial success or an ambiguous response.
func (r *Rebaser) Rebase(ctx context.Context) error {
	pulls, err := r.repository.ListOpenPullRequests(ctx)
	if err != nil {
		return err
	}

	var failures error
	for _, pull := range pulls {
		if ctx.Err() != nil {
			return errors.Join(failures, context.Cause(ctx))
		}

		if !r.eligible(pull) {
			continue
		}

		if err := r.rebase(ctx, pull.Number); err != nil {
			failures = errors.Join(failures, fmt.Errorf("pull request #%d: %w", pull.Number, err))
		}
	}

	return failures
}

func (r *Rebaser) rebase(ctx context.Context, number int) error {
	pull, err := r.repository.GetPullRequest(ctx, number)
	if err != nil {
		return err
	}

	if !r.eligible(pull) {
		return nil
	}

	if strings.IsAnyEmpty(pull.ID, pull.Head.SHA, pull.Base.Name) {
		return errors.New("pull request is missing branch metadata")
	}

	comparison, err := r.repository.Compare(ctx, pull.Base.Name, pull.Head.SHA)
	if err != nil {
		return err
	}

	if comparison.BehindBy <= 0 {
		return nil
	}

	return r.requestRebase(ctx, pull)
}

func (r *Rebaser) requestRebase(ctx context.Context, expected PullRequest) error {
	current, err := r.repository.GetPullRequest(ctx, expected.Number)
	if err != nil {
		return err
	}

	if !r.eligible(current) {
		return nil
	}

	sameID := current.ID == expected.ID
	sameHead := current.Head.Name == expected.Head.Name && current.Head.SHA == expected.Head.SHA
	sameBase := current.Base.Name == expected.Base.Name
	if !sameID || !sameHead || !sameBase {
		return errors.New("pull request changed before rebase; retry on the next run")
	}

	if err := r.repository.Rebase(ctx, current); err != nil {
		return err
	}

	r.logger.Info("requested pull request rebase", "pull_request", current.Number, "branch", current.Head.Name)

	return nil
}

func (r *Rebaser) eligible(pull PullRequest) bool {
	headRepository := strings.ToLower(pull.Head.Repository)
	baseRepository := strings.ToLower(pull.Base.Repository)
	sameRepository := headRepository == r.name && baseRepository == r.name

	return pull.State == "open" &&
		!pull.Draft &&
		pull.AutoMerge &&
		sameRepository
}
