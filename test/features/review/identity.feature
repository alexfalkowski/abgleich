@manual @commands
Feature: Reuse authenticated results for the reviewed revisions
  Maintainers reuse completed App reviews while new revisions and independent AI tasks remain reviewable.

  @codereview
  Scenario: Reuse an existing App review that becomes visible while waiting
    Given AI code reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the pull request has already received a code review
    And the previous App code review is temporarily absent from the discussion
    And the existing App code review becomes visible while a request waits
    When Abgleich processes the waiting code review
    Then the existing App code review should remain the only review
    And exactly 1 request should reach the AI service

  @codereview
  Scenario: Reuse an existing App review that becomes visible during preparation
    Given AI code reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the pull request has already received a code review
    And the previous App code review is temporarily absent from the discussion
    And the existing App code review becomes visible while a review is being prepared
    When a maintainer with "write" access comments "/ab codereview"
    Then the existing App code review should remain the only review
    And exactly 2 requests should reach the AI service

  @codereview
  Scenario: Review a new revision after an earlier successful review
    Given AI code reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the pull request has already received a code review
    When the maintainer reissues the code review request after new commits
    Then both revisions should receive their own code review

  @codereview
  Scenario: Keep summary and review results independent
    Given AI code reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And code reviews and summaries are independently enabled
    And the pull request has already been summarised
    When a maintainer with "write" access comments "/ab codereview"
    Then the code review should identify the request and revisions
    And exactly 2 requests should reach the AI service

  @codereview
  Scenario: Keep a review from suppressing a later summary
    Given AI code reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And code reviews and summaries are independently enabled
    And the pull request has already received a code review
    When a maintainer with "write" access comments "/ab summarise"
    Then the summary should identify the request and the pull request revision
    And exactly 2 requests should reach the AI service

  @review-results @codereview
  Scenario: Reuse a legacy review after restart
    Given AI code reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And a legacy App code review exists for the current revision
    And the worker has restarted
    When a maintainer with "write" access comments "/ab codereview"
    Then the original code review should remain without another model call
    And no native review should be submitted

  @review-results @codereview
  Scenario Outline: Ignore native reviews belonging to another author
    Given AI code reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And a native code review lookalike was posted by "<author>"
    When a maintainer with "write" access comments "/ab codereview"
    Then a fresh App review should identify the current request
    And exactly 2 requests should reach the AI service

    Examples:
      | author      |
      | a human     |
      | another App |

  @review-results @codereview
  Scenario: Find a completed native review beyond the first history page
    Given AI code reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the pull request has 105 earlier native reviews
    And the pull request has already received a code review
    When a maintainer with "write" access comments "/ab codereview"
    Then the existing native review should prevent another model call

  @review-results @codereview
  Scenario Outline: Decline review when native history is incomplete
    Given AI code reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the pull request has 105 earlier native reviews
    And native review history returns "<response>" on page <page>
    When a maintainer with "write" access comments "/ab codereview"
    Then the comment delivery should return HTTP 503
    And exactly 0 requests should reach the AI service
    And native review history should remain unchanged

    Examples:
      | response   | page |
      | null       | 1    |
      | empty      | 1    |
      | whitespace | 1    |
      | null       | 2    |
      | empty      | 2    |
      | whitespace | 2    |

  @review-results
  Scenario Outline: Keep forged history from amplifying author lookups
    Given AI <kind> reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And 105 native <kind> review lookalikes were posted by "<author>"
    When a maintainer with "write" access comments "/ab <command>"
    Then the comment delivery should return HTTP 204
    And a fresh App review should identify the current request
    And exactly 2 requests should reach the AI service
    And author checks should stay within the history request budget

    @codereview
    Examples:
      | kind | command    | author      |
      | code | codereview | a human     |
      | code | codereview | another App |

    @secreview
    Examples:
      | kind     | command   | author      |
      | security | secreview | a human     |
      | security | secreview | another App |

  @review-results @codereview
  Scenario Outline: Reject incomplete or unverified review authorship
    Given AI code reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And 105 native code review lookalikes were posted by "another App"
    And native review authorship returns "<problem>"
    When a maintainer with "write" access comments "/ab codereview"
    Then the comment delivery should return HTTP 503
    And exactly 1 request should reach the AI service
    And native review history should remain unchanged

    Examples:
      | problem              |
      | missing results      |
      | partial results      |
      | partial second batch |
      | null review          |
      | missing authorship   |
      | mismatched identity  |
      | duplicate identities |
      | GraphQL errors       |

  @review-results @codereview
  Scenario: Find the App result after a full batch of forged markers
    Given AI code reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And 105 native code review lookalikes were posted by "another App"
    And the pull request has already received a code review
    When a maintainer with "write" access comments "/ab codereview"
    Then the existing native review should prevent another model call

  @review-results @codereview
  Scenario: Permit a richer review after a patch-only result
    Given repository source and public Go dependencies are available for review
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And a patch-only review was published before source lookup was available
    When a maintainer with "write" access comments "/ab codereview"
    Then the code review should explain the related implementation
    And the reviewer should receive complete files at the reviewed revisions
    And the source-enabled review should coexist with the old patch-only review

  @review-results @codereview
  Scenario: Stop source exploration when another matching result appears
    Given repository source and public Go dependencies are available for review
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the pull request has already received a code review
    And the previous App code review is temporarily absent from the discussion
    And the existing App code review becomes visible while a review is being prepared
    When a maintainer with "write" access comments "/ab codereview"
    Then the existing App code review should remain the only review
    And exactly 5 requests should reach the AI service

  @review-results
  Scenario Outline: Reuse a submitted verdict after restarting
    Given AI <kind> reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the AI <kind> review answer has "<answer>"
    And the pull request has already received a <kind> review
    And the worker has restarted
    When a maintainer with "write" access comments "/ab <command>"
    Then the original <kind> review should remain without another model call
    And the <kind> review decision should be "<decision>"

    @codereview
    Examples:
      | kind | command    | answer          | decision          |
      | code | codereview | no findings     | APPROVED          |
      | code | codereview | inline findings | CHANGES_REQUESTED |

    @secreview
    Examples:
      | kind     | command   | answer          | decision          |
      | security | secreview | no findings     | APPROVED          |
      | security | secreview | inline findings | CHANGES_REQUESTED |

  @review-results
  Scenario Outline: Retry an incomplete review on the same commits
    Given AI <kind> reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the AI <kind> review answer has "incomplete"
    And the pull request has already received a <kind> review
    And the AI <kind> review answer has "no findings"
    When a maintainer with "write" access comments "/ab <command>"
    Then the <kind> review decision should be "APPROVED"
    And the incomplete review should have no success marker
    And exactly 2 requests should reach the AI service

    @codereview
    Examples:
      | kind | command    |
      | code | codereview |

    @secreview
    Examples:
      | kind     | command   |
      | security | secreview |

  @review-results
  Scenario Outline: Retry a completed review after its context limitations are resolved
    Given AI <kind> reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the review cannot inspect "changed-file patch"
    And the AI <kind> review answer has "no findings"
    And the pull request has already received a <kind> review
    And the <kind> review can inspect the missing context again
    When a maintainer with "write" access comments "/ab <command>"
    Then the <kind> review decision should be "APPROVED"
    And the incomplete review should have no success marker
    And exactly 2 requests should reach the AI service

    @codereview
    Examples:
      | kind | command    |
      | code | codereview |

    @secreview
    Examples:
      | kind     | command   |
      | security | secreview |

  @review-results @codereview
  Scenario: Reuse a dismissed result after restarting
    Given AI code reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the pull request has already received a code review
    And a maintainer dismissed the latest review findings
    And the worker has restarted
    When a maintainer with "write" access comments "/ab codereview"
    Then the original code review should remain without another model call

  @codereview
  Scenario: Keep a completed review after restart
    Given AI code reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the pull request has already received a code review
    And the worker has restarted
    When a maintainer with "write" access comments "/ab codereview"
    Then the original code review should remain without another model call

  @codereview
  Scenario: Recover the published result after an ambiguous GitHub response
    Given AI code reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And GitHub reports an error after publishing the reply
    And the pull request has already received a code review
    And the worker has restarted
    When a maintainer with "write" access comments "/ab codereview"
    Then the original code review should remain without another model call

  @codereview
  Scenario Outline: Ignore unauthenticated lookalikes
    Given AI code reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And a code review lookalike was posted by "<author>"
    When a maintainer with "write" access comments "/ab codereview"
    Then the code review should identify the request and revisions
    And exactly 2 requests should reach the AI service

    Examples:
      | author      |
      | a human     |
      | another App |

  @codereview
  Scenario: Look through an entire long discussion for a completed review
    Given AI code reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the pull request has 105 earlier comments
    And the pull request has already received a code review
    When a maintainer with "write" access comments "/ab codereview"
    Then the original code review should remain without another model call

  @secreview
  Scenario: Request all AI tasks for the same revision
    Given AI security reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And security reviews and summaries are independently enabled
    And the pull request has already been summarised
    And the pull request has already received a code review
    When a maintainer with "write" access comments "/ab secreview"
    Then the security review should identify the request and revisions
    And exactly 3 requests should reach the AI service

  @secreview
  Scenario: Keep security review from suppressing the other AI tasks
    Given AI security reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And security reviews and summaries are independently enabled
    And the pull request has already received a security review
    And the pull request has already received a code review
    When a maintainer with "write" access comments "/ab summarise"
    Then the summary should identify the request and the pull request revision
    And exactly 3 requests should reach the AI service

  @secreview
  Scenario: Withhold source when the full discussion is unavailable
    Given AI security reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the pull request has 105 earlier comments
    And GitHub cannot retrieve the complete security review discussion
    When a maintainer with "write" access comments "/ab secreview"
    Then the request should be temporarily declined
    And no security review source should be shared
    And no completed security review should be posted

  @secreview
  Scenario: Review a changed target revision
    Given AI security reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the pull request has already received a security review
    And the target branch has advanced since the security review
    When a maintainer with "write" access comments "/ab secreview"
    Then both revisions should receive their own security review
