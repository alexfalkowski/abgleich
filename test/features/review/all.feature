@manual @commands @codereview
Feature: Request every configured review
  Maintainers can use one comment to request every enabled review.
  Each review retains its own consent checks, queue, and revision-specific result.

  Background:
    Given AI code reviews are configured for this repository
    And both review commands are enabled

  Scenario: Request every configured review on hold without repository opt-in
    Given a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the reviewed repository has hold and automatic merging disabled
    When a maintainer with "write" access comments "/ab review"
    Then the command comment should show "receipt and acceptance" feedback
    And the code review should identify the request and revisions
    And the security review should identify the request and revisions
    And exactly 2 requests should reach the AI service

  Scenario: Reuse an individual review while requesting the remaining reviews
    Given a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the pull request has already received a code review
    When a maintainer with "write" access comments "/ab review"
    Then the security review should identify the request and revisions
    And each configured review should have exactly one result
    And exactly 2 requests should reach the AI service

  Scenario: Reuse aggregate results after restart
    Given a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And every configured review has already completed
    And the worker has restarted
    When a maintainer with "write" access comments "/ab review"
    Then the command comment should show "receipt and acceptance" feedback
    And each configured review should have exactly one result
    And exactly 2 requests should reach the AI service

  Scenario: Keep aggregate results reusable by individual commands
    Given a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And every configured review has already completed
    When a maintainer with "write" access comments "/ab codereview"
    Then the command comment should show "receipt and acceptance" feedback
    And each configured review should have exactly one result
    And exactly 2 requests should reach the AI service

  Scenario: Reuse pending reviews when the aggregate request is delivered twice
    Given a pull request from "the repository" for branch "contribution" is "waiting for checks"
    When GitHub delivers the same aggregate review request twice while reviews are running
    Then the code review should identify the request and revisions
    And the security review should identify the request and revisions
    And each configured review should have exactly one result
    And exactly 2 requests should reach the AI service

  Scenario Outline: Decline when no review is configured
    Given a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And optional AI commands are disabled with the provider "<provider>"
    When a maintainer with "write" access comments "/ab review"
    Then the command comment should show "receipt and rejection" feedback
    And only merge automation should run in the background
    And the AI service should remain unused

    Examples:
      | provider   |
      | configured |
      | omitted    |

  Scenario: Require current write access for every review
    Given a pull request from "the repository" for branch "contribution" is "waiting for checks"
    When a maintainer with "read" access comments "/ab review"
    Then the request should be refused without write access
    And no code review source should be shared
    And no security review source should be shared

  Scenario: Decline fork contributions
    Given a pull request from "a fork" for branch "contribution" is "waiting for checks"
    When a maintainer with "write" access comments "/ab review"
    Then the command comment should show "receipt and rejection" feedback
    And no code review source should be shared
    And no security review source should be shared

  Scenario Outline: Ignore unsupported aggregate command spellings and arguments
    Given a pull request from "the repository" for branch "contribution" is "waiting for checks"
    When a maintainer with "write" access comments "<command>"
    Then no code review source should be shared
    And no security review source should be shared

    Examples:
      | command           |
      | /ab Review        |
      | /ab reviews       |
      | /ab review please |

  Scenario Outline: Preserve consent checks before sharing source
    Given a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the code review request loses eligibility through "<change>" before review preparation
    When a maintainer with "write" access comments "/ab review"
    Then no code review source should be shared
    And no security review source should be shared

    Examples:
      | change          |
      | request edited  |
      | request deleted |

  Scenario Outline: Recheck aggregate consent before publication
    Given a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And permission for the review is withdrawn through "<change>" while the answer is prepared
    When a maintainer with "write" access comments "/ab review"
    Then no completed code review should be posted
    And no completed security review should be posted

    Examples:
      | change               |
      | loss of write access |
      | an edited request    |

  Scenario: Continue requesting other reviews when one queue is full
    Given a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the code review queue is full while both reviews are enabled
    When a maintainer with "write" access comments "/ab review"
    Then the request should be temporarily declined
    And the command comment should show "receipt and rejection" feedback
    And the security review should complete after the active reviews finish

  Scenario: Keep cancellation feedback authorized by the aggregate request
    Given a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the aggregate review is waiting behind active reviews
    When the pull request merges before its queued review starts
    Then the code review cancellation should explain "merged"
    And the security review cancellation should explain "merged"
    And the canceled review should not publish findings
