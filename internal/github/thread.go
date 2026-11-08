package github

import (
	"github.com/alexfalkowski/go-service/v2/errors"
	"github.com/alexfalkowski/go-service/v2/strings"
)

const (
	graphQLOpen   = "OPEN"
	graphQLClosed = "CLOSED"
	graphQLMerged = "MERGED"
)

// ReviewThread contains the state and authenticated ownership needed to maintain
// an existing conversation. IsAppAuthor requires both its root and parent review.
type ReviewThread struct {
	// ID is the GraphQL node identity used for final rechecks and resolution.
	ID string

	// ReviewBody contains the parent review's result marker for feature ownership checks.
	ReviewBody string

	// IsOutdated reports stale placement from GitHub, without establishing that a finding was fixed.
	IsOutdated bool

	// IsResolved reports whether the conversation is already closed.
	IsResolved bool

	// ViewerCanResolve reports whether the authenticated installation may resolve the thread.
	ViewerCanResolve bool

	// IsAppAuthor requires API-authenticated Bot authorship of both the root comment and parent review.
	IsAppAuthor bool
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
	sameNodeID := p.ID == pull.GetNodeID()
	sameNumber := p.Number == pull.GetNumber()
	sameRepository := strings.ToLower(p.Repository.NameWithOwner) == strings.ToLower(repo.GetFullName())
	validState := p.State == graphQLOpen || p.State == graphQLClosed || p.State == graphQLMerged

	return sameNodeID && sameNumber && sameRepository && validState
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
		IsAppAuthor: *root.ViewerDidAuthor && *review.ViewerDidAuthor && root.Author.Type == UserBot && review.Author.Type == UserBot,
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
