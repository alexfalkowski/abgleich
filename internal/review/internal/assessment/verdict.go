package assessment

import (
	"github.com/alexfalkowski/abgleich/internal/github"
	"github.com/alexfalkowski/abgleich/internal/review/internal/history"
	"github.com/alexfalkowski/go-service/v3/slices"
)

func (r *reviewReply) assess(result *reviewResult) {
	requiresChanges := slices.ContainsFunc(result.Findings, func(finding reviewFinding) bool {
		return finding.Severity != severityP3
	})
	r.Event = github.ReviewComment
	switch {
	case requiresChanges:
		r.Event = github.ReviewRequestChanges
		r.Assessment = "Findings."
		r.Decision = "Request changes."
	case len(r.Unverified) > 0:
		r.Assessment = "Review unfinished."
		r.Decision = "Comment. Approval withheld because earlier blockers remain unverified."
	case !*result.Complete:
		r.Assessment = "Review unfinished."
		r.Decision = "Comment. Approval withheld because the reviewer did not finish. See the overview for remaining work."
	case len(r.Missing) > 0:
		r.Assessment = "Review completed with limitations."
		r.Decision = "Comment. Approval withheld because the worker recorded the limitations below."
	case len(result.Findings) > 0:
		r.Event = github.ReviewApprove
		r.Assessment = history.LowSeverityFindings
		r.Decision = "Approve. Only low-severity findings."
	default:
		r.Event = github.ReviewApprove
		r.Assessment = history.NoFindings
		r.Decision = "Approve."
	}
}
