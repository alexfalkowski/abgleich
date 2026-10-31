@github @manual @commands @codereview
Feature: Keep code review recoverable and independent
  Background:
    Given AI code reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"

  Scenario Outline: Fail visibly when the provider cannot answer
    Given the AI service supplies "<answer>"
    When a maintainer with "write" access comments "/ab codereview"
    Then the code review should fail without publishing findings
    And exactly 1 request should reach the AI service

    Examples:
      | answer                  |
      | a refusal               |
      | an unfinished answer    |
      | no answer               |
      | an answer arriving late |

  Scenario: Report provider context limits without truncating or retrying
    Given the AI provider rejects the review context as too large
    When a maintainer with "write" access comments "/ab codereview"
    Then the code review should fail without publishing findings
    And the review context should include "File: test.go"
    And exactly 1 request should reach the AI service

  Scenario: Coalesce concurrent maintainer requests
    When two maintainers request code review while inference is running
    Then the code review should identify the request and revisions
    And exactly 1 request should reach the AI service

  Scenario: Deduplicate an actual repeated webhook delivery
    When GitHub delivers the same code review request twice
    Then the code review should identify the request and revisions
    And exactly 1 request should reach the AI service

  Scenario: Use an App result restored while the request is waiting
    Given the pull request has already received a code review
    And a previous App review was temporarily removed
    And that App review is restored while a request waits
    When Abgleich processes the waiting code review
    Then only the restored App review should remain
    And exactly 1 request should reach the AI service

  Scenario: Use an App result restored while inference is running
    Given the pull request has already received a code review
    And a previous App review was temporarily removed
    And that App review is restored while inference is running
    When a maintainer with "write" access comments "/ab codereview"
    Then only the restored App review should remain
    And exactly 2 requests should reach the AI service

  Scenario: Cancel inference during worker shutdown
    Given the worker is waiting for the code review provider
    When the worker restarts
    Then the interrupted code review should be cancelled without publication

  Scenario: Keep status available while inference waits
    When the maintainer asks for status while code review is running
    Then exactly 1 request should reach the AI service
    And the worker should remain healthy

  Scenario: Review a new revision after an earlier successful review
    Given the pull request has already received a code review
    When the maintainer reissues the review request after new commits
    Then both revisions should receive their own code review

  Scenario: Bound pending work without blocking automation
    Given four maintainers are waiting for code reviews
    When a maintainer with "write" access comments "/ab codereview"
    Then the request should be temporarily declined
    And ordinary automation should continue while those requests wait

  Scenario: Reissue code review after pending work is lost on restart
    Given a code review request was interrupted by restart
    When a maintainer with "write" access comments "/ab codereview"
    Then the code review should identify the request and revisions
    And exactly 1 request should reach the AI service

  Scenario: Ignore a request without an identified target revision
    Given the pull request has no target branch revision
    When a maintainer with "write" access comments "/ab codereview"
    Then no code review source should be shared
    And no completed code review should be posted

  Scenario Outline: Reject eligibility changes before inference
    Given the review request loses eligibility through "<change>" before inference
    When a maintainer with "write" access comments "/ab codereview"
    Then no code review source should be shared
    And no completed code review should be posted

    Examples:
      | change                |
      | request edited        |
      | request deleted       |
      | closed                |
      | draft                 |
      | retargeted            |
      | base changed          |
      | missing base revision |

  Scenario Outline: Reject eligibility changes during inference
    Given the review request loses eligibility through "<change>" during inference
    When a maintainer with "write" access comments "/ab codereview"
    Then no completed code review should be posted
    And exactly 1 request should reach the AI service

    Examples:
      | change                |
      | request deleted       |
      | closed                |
      | draft                 |
      | retargeted            |
      | missing base revision |

  Scenario: Keep summary and review results independent
    Given code reviews and summaries are independently enabled
    And the pull request has already been summarised
    When a maintainer with "write" access comments "/ab codereview"
    Then the code review should identify the request and revisions
    And exactly 2 requests should reach the AI service

  Scenario: Keep a review from suppressing a later summary
    Given code reviews and summaries are independently enabled
    And the pull request has already received a code review
    When a maintainer with "write" access comments "/ab summarise"
    Then the summary should identify the request and the pull request revision
    And exactly 2 requests should reach the AI service

  Scenario Outline: Show actual usage without estimating money
    Given the AI service reports "<information>" token usage
    When a maintainer with "write" access comments "/ab codereview"
    Then the review should show "<expected>" in its usage details

    Examples:
      | information    | expected    |
      | ordinary input | test-review |
      | cached input   | 20          |
      | cache writes   | 30          |
      | no usage       | Unavailable |

