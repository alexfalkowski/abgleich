@manual @commands @codereview @review-results
Feature: Carry previous blockers into a follow-up review
  Maintainers can verify findings after rewriting a PR and restarting the worker.

  Scenario Outline: Account for previous blockers after a rebase
    Given AI code reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the pull request received inline findings that are "<outcome>" after the rebase
    And the worker has restarted
    When the maintainer requests a code review after rebasing and force pushing
    Then the code review decision should be "<decision>"
    And the follow-up should report the previous findings as "<outcome>"

    Examples:
      | outcome    | decision          |
      | still open | CHANGES_REQUESTED |
      | cleared    | APPROVED          |
      | unverified | COMMENTED         |

  Scenario: Recheck a related file whose contents changed outside the PR patch
    Given AI code reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the reviewer needs "a related source file"
    And the pull request has already received a code review
    And related source is "helper changed at new head"
    When the maintainer reissues the code review request after new commits
    Then the code review decision should be "APPROVED"
    And the follow-up should call out the changed related file

  Scenario: Verify earlier findings when a base advances and the contribution rebases
    Given AI code reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And an earlier code review found blockers on the contribution
    And the worker has restarted
    When the maintainer requests follow-ups as the base advances and the contribution is rebased and fixed
    Then the follow-up should confirm the blockers without opening another thread
    And the reviewer should read the new file at the base while the contribution is behind
    And the reviewer should see the inherited file after rebasing without any change to the pull request patch
    And the fix verification should approve the corrected revision

  Scenario: Verify a head-only fix against the same base before rebasing
    Given AI code reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And an earlier code review found blockers on the contribution
    And a follow-up confirmed the blockers using a new file on the base
    When the maintainer requests a review of a fix pushed without rebasing
    Then the reviewer should recheck the file on the unchanged base
    And the fix verification should approve the corrected revision

  Scenario: Verify blockers when GitHub no longer has the superseded head tree
    Given AI code reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And an earlier code review found blockers on the contribution
    And GitHub has dropped the previously reviewed head tree
    When the maintainer reissues the code review request after new commits
    Then the code review decision should be "CHANGES_REQUESTED"
    And the follow-up should carry current source and blockers without needing the superseded tree

  Scenario: Retry a partial review on the same revisions
    Given AI code reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the pull request received inline findings that are "partially verified" after the rebase
    And an earlier follow-up on the rebased contribution found a new blocker and left one previous blocker unverified
    And GitHub redelivered the same request and the reviewer reported a different new blocker
    When the maintainer requests another review of the same revisions
    Then the code review decision should be "APPROVED"
    And the follow-up should account for every earlier blocker once

  Scenario: Stop a follow-up when GitHub cannot verify the earlier review's authorship
    Given AI code reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the pull request has already received a code review
    And native review authorship returns "missing results"
    When the maintainer reissues the code review request after new commits
    Then the follow-up should report a source collection failure before sending anything to the model
    And native review history should remain unchanged

  Scenario: Preserve earlier blockers when GitHub cannot compare previously inspected base source
    Given AI code reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And an earlier review with blockers inspected a removed file
    And the file no longer appears as removed and GitHub cannot return the base tree
    When the maintainer reissues the code review request after new commits
    Then the code review decision should be "COMMENTED"
    And the follow-up should identify the unavailable comparison and the remaining blockers

  Scenario: Use a full assessment when the latest decisive review has no follow-up record
    Given AI code reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the pull request received two requests for changes but only the earlier review saved follow-up context
    When the maintainer reissues the code review request after new commits
    Then the code review decision should be "APPROVED"
    And the reviewer should receive the full contribution without earlier follow-up context

  Scenario: Keep a partial code review's findings blocking for a security review
    Given AI code reviews are configured for this repository
    And both review commands are enabled
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the AI code review answer has "incomplete with findings"
    And the pull request has already received a code review
    And the AI security review answer has "no findings"
    When a maintainer with "write" access comments "/ab secreview"
    Then the security review decision should be "COMMENTED"
    And its decision should explain that the other review still requests changes

  Scenario: Preserve blockers when fingerprints exceed the review body budget
    Given AI code reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the pull request received a partial review with a compact blocker record
    When the maintainer requests another review of the same revisions
    Then the code review decision should be "APPROVED"
    And the follow-up should carry the original blockers and disclose the unavailable fingerprints

  Scenario Outline: Preserve an active request for changes when its blocker record is too large
    Given AI code reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the pull request received a "<assessment>" review whose blocker record could not fit
    When the maintainer requests another review of the same revisions
    Then the worker should explain that the previous blockers are unavailable
    And native review history should remain unchanged

    Examples:
      | assessment |
      | partial    |
      | completed  |

  Scenario: Reassess newer blockers when the pull request returns to a reviewed revision
    Given AI code reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the pull request received inline findings that are "partially verified" after the rebase
    And an earlier follow-up on the rebased contribution found a new blocker and left one previous blocker unverified
    When the maintainer restores the originally reviewed revisions and requests a review
    Then the code review decision should be "APPROVED"
    And the follow-up should account for every earlier blocker once

  Scenario: Keep genuine blockers when a newer lookalike appears in history
    Given AI code reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the pull request received inline findings that are "unverified" after the rebase
    And a newer same-feature review belongs to another App
    When the maintainer requests a code review after rebasing and force pushing
    Then the code review decision should be "COMMENTED"
    And only the original blocker IDs should reach the model

  Scenario: Keep genuine blockers when GitHub omits the earlier review author
    Given AI code reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the pull request received inline findings that are "unverified" after the rebase
    And GitHub omits the author of the earlier review
    When the maintainer requests a code review after rebasing and force pushing
    Then the code review decision should be "COMMENTED"
    And only the original blocker IDs should reach the model

  Scenario: Keep a redacted finding unresolved
    Given AI code reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the pull request received inline findings that are "cleared" after the rebase
    And a maintainer redacted an earlier finding
    When the maintainer requests a code review after rebasing and force pushing
    Then the code review decision should be "COMMENTED"
    And the redacted finding should remain unverified

  Scenario: Stop sending earlier findings after a maintainer redacts one
    Given AI code reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the pull request received inline findings that are "cleared" after the rebase
    And the reviewer needs "a related source file"
    When a maintainer redacts an earlier finding while the follow-up runs
    Then the worker should stop the follow-up before sending the finding again
    And the pull request should only have its earlier request for changes

  Scenario: Respect a dismissal received during a follow-up
    Given AI code reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the pull request received inline findings that are "still open" after the rebase
    When a maintainer dismisses the earlier review while the follow-up runs
    Then the worker should discard the result because the App review history changed
    And the pull request should only have its dismissed review

  Scenario: Respect a partial review that becomes visible during a retry
    Given AI code reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And an earlier partial request for changes is temporarily absent from review history
    When the partial review becomes visible while the maintainer's retry runs
    Then the worker should discard the result because the App review history changed
    And the pull request should only have its earlier request for changes
