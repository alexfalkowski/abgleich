package status

import (
	"embed"
	"text/template"

	"github.com/alexfalkowski/abgleich/internal/automerge"
	"github.com/alexfalkowski/abgleich/internal/github"
	"github.com/alexfalkowski/go-service/v2/time"
)

//go:embed status.md.tmpl
var statusTemplates embed.FS

var statusTemplate = template.Must(template.ParseFS(statusTemplates, "status.md.tmpl"))

// Reply holds the observed assessment and display values for a status comment.
// CheckedAt is formatted in UTC, and RequestID identifies the request comment.
type Reply struct {
	Assessment automerge.Assessment
	Emoji      string
	AutoMerge  string
	SHA        string
	CheckedAt  string
	RequestID  int64
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
	case assessment.Action != "":
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
