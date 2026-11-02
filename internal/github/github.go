package github

import (
	"github.com/alexfalkowski/go-service/v2/errors"
	"github.com/alexfalkowski/go-service/v2/strings"
	"github.com/google/go-github/v93/github"
)

// Repository aliases [github.Repository] to keep SDK imports in this package.
type Repository = github.Repository

// Branch aliases [github.Branch] for branch cleanup snapshots.
type Branch = github.Branch

// RepositoryCommit aliases [github.RepositoryCommit] for commit age checks.
type RepositoryCommit = github.RepositoryCommit

// RefUpdate carries the expected and replacement OIDs for an atomic ref update.
type RefUpdate struct {
	// Name is the fully qualified reference name, such as refs/heads/feature.
	Name string `json:"name"`
	// BeforeOID requires the reference to still point at this commit before it can change.
	BeforeOID string `json:"beforeOid"`
	// AfterOID sets the replacement commit; the all-zero OID deletes the reference.
	AfterOID string `json:"afterOid"`
}

// UpdateRefsRequest deletes one branch with an atomic expected-commit guard.
// The zero AfterOID deletes the reference; ClientMutationID must be echoed.
type UpdateRefsRequest struct {
	// Query contains the GraphQL operation submitted with these variables.
	Query string `json:"query"`
	// Variables binds the operation's GraphQL arguments.
	Variables struct {
		// Input supplies the mutation arguments, including the target identity.
		Input struct {
			// RepositoryID identifies the repository by its GraphQL node ID.
			RepositoryID string `json:"repositoryId"`
			// ClientMutationID correlates the mutation request with its acceptance response.
			ClientMutationID string `json:"clientMutationId"`
			// RefUpdates contains the guarded reference changes; cleanup submits one deletion at a time.
			RefUpdates []RefUpdate `json:"refUpdates"`
		} `json:"input"`
	} `json:"variables"`
}

// UpdateRefsResponse requires a matching mutation ID and no GraphQL errors.
type UpdateRefsResponse struct {
	// Data contains returned operation data, which still requires error and identity validation.
	Data struct {
		// Update confirms an accepted updateRefs mutation; nil cannot confirm success.
		Update *struct {
			// ClientMutationID correlates the mutation request with its acceptance response.
			ClientMutationID string `json:"clientMutationId"`
		} `json:"updateRefs"`
	} `json:"data"`
	// Errors retains GraphQL failures even when the HTTP response succeeds.
	Errors Errors `json:"errors"`
}

// Validate rejects absent, unconfirmed, or failed reference updates.
func (r *UpdateRefsResponse) Validate(expectedID string) error {
	if err := r.Errors.Validate(); err != nil {
		return errors.Prefix("github: delete branch", err)
	}

	if r.Data.Update == nil || r.Data.Update.ClientMutationID != expectedID {
		return errors.New("github: delete branch returned no matching mutation")
	}

	return nil
}

// PullRequest aliases [github.PullRequest] so consumers use the project-owned package.
type PullRequest = github.PullRequest

// PullRequestReview aliases [github.PullRequestReview] for native review history.
type PullRequestReview = github.PullRequestReview

// PullRequestReviewRequest aliases [github.PullRequestReviewRequest] for review publication.
type PullRequestReviewRequest = github.PullRequestReviewRequest

// DraftReviewComment aliases [github.DraftReviewComment] for inline findings.
type DraftReviewComment = github.DraftReviewComment

// Label aliases [github.Label] for shared label membership checks.
type Label = github.Label

// IssueEvent aliases [github.IssueEvent] for pull request automation history.
type IssueEvent = github.IssueEvent

// IssueComment aliases [github.IssueComment] for signed command requests.
type IssueComment = github.IssueComment

// IssueCommentEvent aliases [github.IssueCommentEvent] for verified deliveries.
type IssueCommentEvent = github.IssueCommentEvent

// User aliases [github.User] for current command-author permission checks.
type User = github.User

// CommitsComparison aliases [github.CommitsComparison] so consumers use the project-owned package.
type CommitsComparison = github.CommitsComparison

// CommitFile aliases [github.CommitFile] for pull request file snapshots.
type CommitFile = github.CommitFile

// CheckRun aliases [github.CheckRun] for current-head CI observations.
type CheckRun = github.CheckRun

// CommitStatus aliases [github.RepoStatus] for commit-status contexts.
type CommitStatus = github.RepoStatus

// PullRequestReviewAuthorsRequest queries authenticated authorship for a batch
// of review node IDs.
type PullRequestReviewAuthorsRequest struct {
	// Query contains the GraphQL operation submitted with these variables.
	Query string `json:"query"`
	// Variables binds the operation's GraphQL arguments.
	Variables struct {
		// IDs lists review node IDs whose authenticated authorship must be checked.
		IDs []string `json:"ids"`
	} `json:"variables"`
}

// PullRequestReviewAuthorsResponse requires each requested review identity and
// explicit authorship. Call Validate even when the HTTP request succeeds.
type PullRequestReviewAuthorsResponse struct {
	// Data contains returned operation data, which still requires error and identity validation.
	Data struct {
		// Nodes returns one authorship result for each requested review identity.
		Nodes []*struct {
			// ViewerDidAuthor reports authenticated viewer authorship; nil cannot establish ownership.
			ViewerDidAuthor *bool `json:"viewer_did_author"`
			// ID identifies the review whose authorship was returned.
			ID string `json:"id"`
		} `json:"nodes"`
	} `json:"data"`
	// Errors retains GraphQL failures even when the HTTP response succeeds.
	Errors Errors `json:"errors"`
}

// Validate requires exactly one explicit authorship result for every requested
// ID, regardless of response order. False is a valid authorship result.
func (r *PullRequestReviewAuthorsResponse) Validate(expectedIDs []string) error {
	if err := r.Errors.Validate(); err != nil {
		return errors.Prefix("github: review authorship", err)
	}
	if len(r.Data.Nodes) != len(expectedIDs) {
		return errors.New("github: review authorship returned an incomplete batch")
	}

	expected := make(map[string]bool, len(expectedIDs))
	for _, id := range expectedIDs {
		expected[id] = true
	}
	for _, node := range r.Data.Nodes {
		if node == nil || !expected[node.ID] || node.ViewerDidAuthor == nil {
			return errors.New("github: review authorship returned no matching review")
		}
		delete(expected, node.ID)
	}

	return nil
}

// UpdatePullRequestBranchInput supplies GitHub's branch-update arguments.
// ExpectedHeadOID rejects concurrent head changes; UpdateMethod selects REBASE
// or MERGE.
type UpdatePullRequestBranchInput struct {
	// PullRequestID identifies the target PR by its GraphQL node ID, rather than its numeric ID.
	PullRequestID string `json:"pullRequestId"`
	// ExpectedHeadOID guards the mutation against concurrent changes to the PR head commit.
	ExpectedHeadOID string `json:"expectedHeadOid"`
	// UpdateMethod selects the GitHub update strategy; the worker requests REBASE.
	UpdateMethod string `json:"updateMethod"`
}

// UpdatePullRequestBranchRequest encodes a branch-update mutation. Query must
// declare $input with the GraphQL type UpdatePullRequestBranchInput.
type UpdatePullRequestBranchRequest struct {
	// Query contains the GraphQL operation submitted with these variables.
	Query string `json:"query"`
	// Variables binds the operation's GraphQL arguments.
	Variables struct {
		// Input supplies the mutation arguments, including the target identity.
		Input UpdatePullRequestBranchInput `json:"input"`
	} `json:"variables"`
}

// UpdatePullRequestBranchResponse decodes GitHub's branch-update mutation.
// HTTP success can include GraphQL errors or incomplete data; call Validate
// before treating an update as accepted.
type UpdatePullRequestBranchResponse struct {
	// Data contains returned operation data, which still requires error and identity validation.
	Data struct {
		// Update contains the branch-update mutation's acceptance payload.
		Update struct {
			// PullRequest identifies the PR accepted for a branch update.
			PullRequest struct {
				// ID must match the requested PR node ID to confirm acceptance.
				ID string `json:"id"`
			} `json:"pullRequest"`
		} `json:"updatePullRequestBranch"`
	} `json:"data"`
	// Errors retains GraphQL failures even when the HTTP response succeeds.
	Errors Errors `json:"errors"`
}

// Validate returns joined GraphQL errors or an error when the payload does not
// identify expectedID. A matching payload confirms acceptance, not completion.
func (r *UpdatePullRequestBranchResponse) Validate(expectedID string) error {
	if err := r.Errors.Validate(); err != nil {
		return errors.Prefix("github: rebase", err)
	}

	if r.Data.Update.PullRequest.ID != expectedID {
		return errors.New("github: rebase returned no matching pull request")
	}

	return nil
}

// EnablePullRequestAutoMergeInput selects a PR and merge method for auto-merge,
// guarded by ExpectedHeadOID against concurrent pushes.
type EnablePullRequestAutoMergeInput struct {
	// PullRequestID identifies the target PR by its GraphQL node ID, rather than its numeric ID.
	PullRequestID string `json:"pullRequestId"`
	// ExpectedHeadOID guards the mutation against concurrent changes to the PR head commit.
	ExpectedHeadOID string `json:"expectedHeadOid"`
	// MergeMethod selects the eventual merge strategy; the worker requests SQUASH.
	MergeMethod string `json:"mergeMethod"`
}

// EnablePullRequestAutoMergeRequest encodes GitHub's auto-merge mutation.
type EnablePullRequestAutoMergeRequest struct {
	// Query contains the GraphQL operation submitted with these variables.
	Query string `json:"query"`
	// Variables binds the operation's GraphQL arguments.
	Variables struct {
		// Input supplies the mutation arguments, including the target identity.
		Input EnablePullRequestAutoMergeInput `json:"input"`
	} `json:"variables"`
}

// EnablePullRequestAutoMergeResponse requires validation even on HTTP success.
type EnablePullRequestAutoMergeResponse struct {
	// Data contains returned operation data, which still requires error and identity validation.
	Data struct {
		// Enable contains the enablePullRequestAutoMerge acceptance payload.
		Enable struct {
			// PullRequest identifies the PR accepted for auto-merge enablement.
			PullRequest struct {
				// ID must match the requested PR node ID to confirm acceptance.
				ID string `json:"id"`
			} `json:"pullRequest"`
		} `json:"enablePullRequestAutoMerge"`
	} `json:"data"`
	// Errors retains GraphQL failures even when the HTTP response succeeds.
	Errors Errors `json:"errors"`
}

// Validate rejects GraphQL errors and responses identifying a different PR.
func (r *EnablePullRequestAutoMergeResponse) Validate(expectedID string) error {
	if err := r.Errors.Validate(); err != nil {
		return errors.Prefix("github: auto-merge", err)
	}

	if r.Data.Enable.PullRequest.ID != expectedID {
		return errors.New("github: auto-merge returned no matching pull request")
	}

	return nil
}

// PullRequestAutomationEventsRequest identifies a PR by node ID and supplies the cursor
// for the next page of its automation history.
type PullRequestAutomationEventsRequest struct {
	// Variables binds the operation's GraphQL arguments.
	Variables struct {
		// ID identifies the PR node whose complete automation history is requested.
		ID string `json:"id"`
		// After selects the next page; nil starts pagination.
		After *string `json:"after"`
	} `json:"variables"`
	// Query contains the GraphQL operation submitted with these variables.
	Query string `json:"query"`
}

// PullRequestAutomationEventsResponse decodes timeline history into SDK issue-event snapshots.
// Call Validate before using the history, even when the HTTP request succeeds.
type PullRequestAutomationEventsResponse struct {
	// Data contains returned operation data, which still requires error and identity validation.
	Data struct {
		// Node contains the PR identity and requested automation timeline page.
		Node struct {
			// ID must match the requested PR node ID before its history is used.
			ID string `json:"id"`
			// TimelineItems contains label and auto-merge-disable history used to establish consent.
			TimelineItems struct {
				// Nodes contains the page's automation events; nil indicates an incomplete page.
				Nodes []*IssueEvent `json:"nodes"`
				// PageInfo reports whether discovery must continue before the results are complete.
				PageInfo struct {
					// EndCursor selects the following page when HasNextPage is true.
					EndCursor *string `json:"endCursor"`
					// HasNextPage reports whether more results remain; nil indicates an incomplete response.
					HasNextPage *bool `json:"hasNextPage"`
				} `json:"pageInfo"`
			} `json:"timelineItems"`
		} `json:"node"`
	} `json:"data"`
	// Errors retains GraphQL failures even when the HTTP response succeeds.
	Errors Errors `json:"errors"`
}

// Validate rejects GraphQL errors, a PR node ID different from expectedID,
// incomplete pages, and events that cannot establish valid automation history.
func (r *PullRequestAutomationEventsResponse) Validate(expectedID string) error {
	if err := r.Errors.Validate(); err != nil {
		return errors.Prefix("github: automation events", err)
	}

	node := r.Data.Node
	if node.ID != expectedID {
		return errors.New("github: automation events returned no matching pull request")
	}

	if node.TimelineItems.Nodes == nil || node.TimelineItems.PageInfo.HasNextPage == nil {
		return errors.New("github: automation events returned an incomplete page")
	}

	return validateAutomationEvents(node.TimelineItems.Nodes)
}

// Errors contains GraphQL failures that may accompany data even on HTTP success.
type Errors []struct {
	// Message contains the failure explanation included in the joined service error.
	Message string `json:"message"`
	// Type classifies the GraphQL failure, including RATE_LIMITED responses.
	Type string `json:"type"`
}

// Validate joins reported failures and includes [ErrRateLimited] for rate limits.
// It returns nil when Errors is empty or nil.
func (e Errors) Validate() error {
	var failures error
	for _, failure := range e {
		failures = errors.Join(failures, errors.New("github: "+failure.Message))

		if failure.Type == "RATE_LIMITED" || rateLimitMessage(failure.Message) {
			failures = errors.Join(failures, ErrRateLimited)
		}
	}

	return failures
}

// validateAutomationEvents requires timestamps, supported event types, and label
// names for label changes so incomplete history cannot authorize automation.
func validateAutomationEvents(events []*IssueEvent) error {
	for _, event := range events {
		if event.GetCreatedAt().IsZero() {
			return errors.New("github: automation event is missing its timestamp")
		}

		switch event.GetEvent() {
		case "LabeledEvent", "UnlabeledEvent":
			if strings.IsEmpty(event.GetLabel().GetName()) {
				return errors.New("github: automation event is missing its label")
			}
		case "AutoMergeDisabledEvent":
		default:
			return errors.New("github: automation event has an unexpected type")
		}
	}

	return nil
}

type threadPullRequest struct {
	ID         string `json:"id"`
	State      string `json:"state"`
	Repository struct {
		NameWithOwner string `json:"nameWithOwner"`
	} `json:"repository"`
	Number int `json:"number"`
}

func (p *threadPullRequest) matches(repo *Repository, pull *PullRequest) bool {
	return p.ID == pull.GetNodeID() && p.Number == pull.GetNumber() &&
		strings.ToLower(p.Repository.NameWithOwner) == strings.ToLower(repo.GetFullName()) &&
		(p.State == "OPEN" || p.State == "CLOSED" || p.State == "MERGED")
}

type reviewThreadNode struct {
	ID               string            `json:"id"`
	IsOutdated       *bool             `json:"isOutdated"`
	IsResolved       *bool             `json:"isResolved"`
	ViewerCanResolve *bool             `json:"viewerCanResolve"`
	PullRequest      threadPullRequest `json:"pullRequest"`
	Comments         struct {
		Nodes []*struct {
			ID              string `json:"id"`
			ViewerDidAuthor *bool  `json:"viewerDidAuthor"`
			Author          struct {
				Type string `json:"__typename"`
			} `json:"author"`
			Review *struct {
				ID              string  `json:"id"`
				Body            *string `json:"body"`
				ViewerDidAuthor *bool   `json:"viewerDidAuthor"`
				Author          struct {
					Type string `json:"__typename"`
				} `json:"author"`
				PullRequest threadPullRequest `json:"pullRequest"`
			} `json:"pullRequestReview"`
		} `json:"nodes"`
	} `json:"comments"`
}

func (n *reviewThreadNode) validate(repo *Repository, pull *PullRequest) (*ReviewThread, error) {
	if n == nil || strings.IsEmpty(n.ID) || n.IsOutdated == nil || n.IsResolved == nil || n.ViewerCanResolve == nil || !n.PullRequest.matches(repo, pull) {
		return nil, errors.New("github: review thread returned incomplete or mismatched state")
	}
	if err := n.validateOwnership(repo, pull); err != nil {
		return nil, err
	}
	return n.snapshot(), nil
}

func (n *reviewThreadNode) snapshot() *ReviewThread {
	root := n.Comments.Nodes[0]
	review := root.Review
	return &ReviewThread{
		ID: n.ID, ReviewBody: *review.Body, IsOutdated: *n.IsOutdated, IsResolved: *n.IsResolved, ViewerCanResolve: *n.ViewerCanResolve,
		IsAppAuthor: *root.ViewerDidAuthor && *review.ViewerDidAuthor && root.Author.Type == "Bot" && review.Author.Type == "Bot",
	}
}

func (n *reviewThreadNode) validateOwnership(repo *Repository, pull *PullRequest) error {
	if len(n.Comments.Nodes) != 1 || n.Comments.Nodes[0] == nil {
		return errors.New("github: review thread returned no root comment")
	}
	root := n.Comments.Nodes[0]
	review := root.Review
	if strings.IsEmpty(root.ID) || root.ViewerDidAuthor == nil || review == nil {
		return errors.New("github: review thread returned incomplete ownership")
	}
	if strings.IsEmpty(review.ID) || review.Body == nil || review.ViewerDidAuthor == nil || !review.PullRequest.matches(repo, pull) {
		return errors.New("github: review thread returned incomplete or mismatched review ownership")
	}
	return nil
}

// ReviewThreadsRequest queries one page of threads by pull request node ID.
// A nil After cursor starts pagination.
type ReviewThreadsRequest struct {
	// Variables binds the operation's GraphQL arguments.
	Variables struct {
		// ID identifies the PR node whose inline conversations are requested.
		ID string `json:"id"`
		// After selects the next page; nil starts pagination.
		After *string `json:"after"`
	} `json:"variables"`
	// Query contains the GraphQL operation submitted with these variables.
	Query string `json:"query"`
}

// ReviewThreadsResponse carries thread ownership, state, and pagination.
// HTTP success still requires GraphQL, identity, and completeness validation.
type ReviewThreadsResponse struct {
	// Data contains returned operation data, which still requires error and identity validation.
	Data struct {
		// Node contains the requested PR and thread page; nil cannot establish identity.
		Node *struct {
			threadPullRequest
			// ReviewThreads contains one page of thread state and ownership evidence.
			ReviewThreads struct {
				// Nodes contains thread candidates; nil indicates an incomplete discovery page.
				Nodes []*reviewThreadNode `json:"nodes"`
				// PageInfo reports whether discovery must continue before the results are complete.
				PageInfo struct {
					// EndCursor selects the following page when HasNextPage is true.
					EndCursor *string `json:"endCursor"`
					// HasNextPage reports whether more results remain; nil indicates an incomplete response.
					HasNextPage *bool `json:"hasNextPage"`
				} `json:"pageInfo"`
			} `json:"reviewThreads"`
		} `json:"node"`
	} `json:"data"`
	// Errors retains GraphQL failures even when the HTTP response succeeds.
	Errors Errors `json:"errors"`
}

// ReviewThreadRequest queries one review thread by its node ID.
type ReviewThreadRequest struct {
	// Query contains the GraphQL operation submitted with these variables.
	Query string `json:"query"`
	// Variables binds the operation's GraphQL arguments.
	Variables struct {
		// ID identifies the thread node to refresh immediately before maintenance.
		ID string `json:"id"`
	} `json:"variables"`
}

// ReviewThreadResponse carries one thread's state and authenticated ownership.
// The service validates its identity and parent PR before returning a snapshot.
type ReviewThreadResponse struct {
	// Data contains returned operation data, which still requires error and identity validation.
	Data struct {
		// Node contains the thread's current state and ownership evidence; nil fails validation.
		Node *reviewThreadNode `json:"node"`
	} `json:"data"`
	// Errors retains GraphQL failures even when the HTTP response succeeds.
	Errors Errors `json:"errors"`
}

// ResolveReviewThreadRequest resolves a thread by node ID without an
// expected-state guard; callers must refresh eligibility before submission.
type ResolveReviewThreadRequest struct {
	// Query contains the GraphQL operation submitted with these variables.
	Query string `json:"query"`
	// Variables binds the operation's GraphQL arguments.
	Variables struct {
		// Input supplies the mutation arguments, including the target identity.
		Input struct {
			// ThreadID identifies the thread to resolve after final eligibility rechecks.
			ThreadID string `json:"threadId"`
		} `json:"input"`
	} `json:"variables"`
}

// ResolveReviewThreadResponse confirms a matching thread's resolved state.
// GraphQL errors or missing confirmation must not be treated as acceptance.
type ResolveReviewThreadResponse struct {
	// Data contains returned operation data, which still requires error and identity validation.
	Data struct {
		// Resolve contains the resolveReviewThread acceptance payload.
		Resolve struct {
			// Thread confirms the target thread and its resolved state; nil cannot confirm acceptance.
			Thread *struct {
				// ID must match the requested thread node ID to confirm acceptance.
				ID string `json:"id"`
				// IsResolved must be present and true to confirm that the mutation resolved the thread.
				IsResolved *bool `json:"isResolved"`
			} `json:"thread"`
		} `json:"resolveReviewThread"`
	} `json:"data"`
	// Errors retains GraphQL failures even when the HTTP response succeeds.
	Errors Errors `json:"errors"`
}

func (r *ReviewThreadsResponse) validate(repo *Repository, pull *PullRequest) error {
	if err := r.Errors.Validate(); err != nil {
		return err
	}
	node := r.Data.Node
	if node == nil || !node.matches(repo, pull) {
		return errors.New("github: review threads returned no matching pull request")
	}
	page := node.ReviewThreads
	if page.Nodes == nil || page.PageInfo.HasNextPage == nil {
		return errors.New("github: review threads returned an incomplete page")
	}
	if *page.PageInfo.HasNextPage {
		cursor := page.PageInfo.EndCursor
		if cursor == nil || strings.IsEmpty(*cursor) {
			return errors.New("github: review threads returned no next cursor")
		}
	}
	return nil
}
