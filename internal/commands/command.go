package commands

import (
	"github.com/alexfalkowski/abgleich/internal/github"
	"github.com/alexfalkowski/go-service/v2/context"
	"github.com/alexfalkowski/go-service/v2/strings"
)

// Handler owns a named command's authorization and action. Handle must re-read
// the current comment and target state before acting, observe cancellation, and
// return whether the request was accepted. The dispatcher supplies feedback.
type Handler interface {
	// Handle revalidates the comment, current permissions, and target state before
	// acting. It observes cancellation and returns whether the request was accepted
	// along with any processing error.
	Handle(ctx context.Context, event *github.IssueCommentEvent) (bool, error)
}

// Parse returns the text after /ab and a single space.
// Surrounding whitespace is ignored; the remaining text is kept
// intact for exact command-name matching. Unrelated comments return an empty string.
func Parse(body string) string {
	prefix, name, _ := strings.Cut(strings.TrimSpace(body), " ")
	if prefix == "/ab" {
		return name
	}

	return strings.Empty
}
