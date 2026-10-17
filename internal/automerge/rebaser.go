package automerge

import (
	"fmt"

	"github.com/alexfalkowski/abgleich/internal/github"
	"github.com/alexfalkowski/go-service/v2/context"
	"github.com/alexfalkowski/go-service/v2/errors"
	"github.com/alexfalkowski/go-service/v2/strings"
	"github.com/alexfalkowski/go-service/v2/telemetry/logger"
)

// NewRebaser constructs a rebaser for the configured GitHub repository. The repository
// supplies fresh snapshots on each invocation, allowing recovery after failures
// without keeping local checkpoints.
func NewRebaser(repository *github.Repository, log *logger.Logger) *Rebaser {
	return &Rebaser{repository: repository, logger: log}
}

// Rebaser keeps non-draft, same-repository auto-merge branches current with
// their bases. It continues processing other pull requests after a failure.
type Rebaser struct {
	repository *github.Repository
	logger     *logger.Logger
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

		if err := r.rebase(ctx, pull.GetNumber()); err != nil {
			failures = errors.Join(failures, fmt.Errorf("pull request #%d: %w", pull.GetNumber(), err))
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

	head, base := pull.GetHead(), pull.GetBase()
	if strings.IsAnyEmpty(pull.GetNodeID(), head.GetSHA(), base.GetRef()) {
		return errors.New("pull request is missing branch metadata")
	}

	comparison, err := r.repository.Compare(ctx, base.GetRef(), head.GetSHA())
	if err != nil {
		return err
	}

	if comparison.GetBehindBy() <= 0 {
		return nil
	}

	return r.requestRebase(ctx, pull)
}

func (r *Rebaser) requestRebase(ctx context.Context, expected *github.PullRequest) error {
	current, err := r.repository.GetPullRequest(ctx, expected.GetNumber())
	if err != nil {
		return err
	}

	if !r.eligible(current) {
		return nil
	}

	currentHead, expectedHead := current.GetHead(), expected.GetHead()
	sameID := current.GetNodeID() == expected.GetNodeID()
	sameHead := currentHead.GetRef() == expectedHead.GetRef() && currentHead.GetSHA() == expectedHead.GetSHA()
	sameBase := current.GetBase().GetRef() == expected.GetBase().GetRef()
	if !sameID || !sameHead || !sameBase {
		return errors.New("pull request changed before rebase; retry on the next run")
	}

	if err := r.repository.Rebase(ctx, current); err != nil {
		return err
	}

	r.logger.Info("requested pull request rebase", "pull_request", current.GetNumber(), "branch", currentHead.GetRef())

	return nil
}

func (r *Rebaser) eligible(pull *github.PullRequest) bool {
	return pull.GetState() == "open" &&
		!pull.GetDraft() &&
		pull.GetAutoMerge() != nil &&
		r.repository.IsSameRepository(pull)
}
