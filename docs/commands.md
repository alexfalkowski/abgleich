# 💬 Comment commands

[← Back to README](../README.md)

## 💬 Comment requests

With [webhooks configured](configuration.md#-comment-webhooks), post a new PR comment containing
exactly `/ab automerge`, `/ab status`, `/ab summarise`, `/ab review`,
`/ab codereview`, or `/ab secreview`. Commands are case-sensitive and allow surrounding whitespace. Extra text and unknown commands
are ignored without reactions. Edited comments and bot authors do not authorize
requests. Mention-based commands are unsupported. The worker re-reads the comment
and PR and checks the author's current write or admin permission. The App reacts
on the original comment:

| Reaction | Meaning |
| --- | --- |
| 👀 | The signed command was received and processing started. |
| 👍 | Accepted: automerge labeled or already enabled, status published, or summary/review queued or already present. |
| 😕 | The request was ineligible, rejected, or failed. |

Use [`/ab review`](reviews.md#-request-all-configured-reviews) to request every
configured review with one comment.

`/ab automerge` adds the durable `automerge` label for the next automation check
when both branches belong to the repository. Fork requests are rejected, including
when already labeled or enabled for auto-merge. With PR event subscriptions
enabled, GitHub's label event prompts that check. Existing labels or enabled
auto-merge need no write, but
authorization is still checked. Acceptance does not mean merging has finished.

A command at or before a label removal or auto-merge disablement cannot restore
the label; missing history blocks action. After revocation, remove any remaining
label and post a later command. Checks survive restarts. Label writes lack an
expected-state guard, so concurrent manual changes can still race them.

Reactions are best effort; failures are logged. Receipt feedback gets at most
one second and a quarter of the remaining delivery budget; final feedback uses
the remainder. Redelivery reuses reactions and can leave both rejection and
acceptance feedback. Consult current PR state for the outcome.

## 🔎 Explain a pull request's automation status

`/ab status` replies with the observed policy decision, reason, next step, head
commit, UTC assessment time, auto-merge state, and request link. It explains
opt-out, Dependabot exclusions, PR eligibility (including ignored forks), `hold`,
squash settings, label consent, and the applicable strategy, explicitly reporting
missing history or comparison evidence.

Status is advisory and performs no merge automation. Consult GitHub's merge box
for individual failing checks or missing reviews. A detected revision or policy
change rejects the request without a reply (HTTP 409); post a new command to retry.
Reads are not atomic, and state can change after the final check. Publication
is never retried automatically; an ambiguous response or webhook redelivery can
leave duplicate replies.

## 💬 Webhook delivery

Webhooks require SHA-256 signatures and respect the configured body limit.
Ignored, queued, or completed deliveries return 204, invalid signatures 401,
unauthorized commands 403, and API failures, conflicts, or timeouts non-success responses.
Repository notifications queue without waiting for GitHub; 204 confirms queue
acceptance, not a completed check. Monitor event-check logs for the later outcome.

Each process handles one command at a time; concurrent commands receive 503.
Automerge and status finish within the request; summaries, code reviews, and
security reviews queue in memory and release the command slot before inference.
Authenticated API requests run independently, so a slow background request does
not block command processing. Token acquisition is serialized, and the delivery
deadline also bounds waiting for token renewal. Polling continues without webhooks;
App API failures can block both. See [webhook recovery](operations.md#-webhook-recovery).
