package status

import (
	"embed"
	"text/template"

	"github.com/alexfalkowski/abgleich/internal/automerge"
	"github.com/alexfalkowski/abgleich/internal/github"
	"github.com/alexfalkowski/go-service/v2/strings"
	"github.com/alexfalkowski/go-service/v2/time"
)

//go:embed status.md.tmpl
var statusTemplates embed.FS

var statusTemplate = template.Must(template.ParseFS(statusTemplates, "status.md.tmpl"))

// Reply holds the observed assessment and display values for a status comment.
// CheckedAt is formatted in UTC, and RequestID identifies the request comment.
type Reply struct {
	// Assessment supplies the observed automation policy and its explanation.
	Assessment automerge.Assessment
	// Emoji distinguishes unavailable assessments, possible actions, and paused outcomes.
	Emoji string
	// AutoMerge displays whether GitHub auto-merge is enabled at the observed snapshot.
	AutoMerge string
	// SHA identifies the validated head commit associated with the assessment.
	SHA string
	// CheckedAt records the assessment time formatted in UTC.
	CheckedAt string
	// RequestID links the reply to the authorizing status command comment.
	RequestID int64
}

func reply(assessment automerge.Assessment, pull *github.PullRequest, requestID int64, checkedAt time.Time) Reply {
	var autoMerge string
	if pull.GetAutoMerge() != nil {
		autoMerge = "Enabled"
	} else {
		autoMerge = "Disabled"
	}

	var emoji string
	switch {
	case assessment.Decision == "Assessment unavailable":
		emoji = "❓"
	case !strings.IsEmpty(assessment.Action):
		emoji = "▶️"
	default:
		emoji = "⏸️"
	}

	return Reply{
		Assessment: assessment,
		Emoji:      emoji,
		AutoMerge:  autoMerge,
		SHA:        pull.GetHead().GetSHA(),
		CheckedAt:  checkedAt.Format("2006-01-02 15:04:05 UTC"),
		RequestID:  requestID,
	}
}
