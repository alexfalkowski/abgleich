@github @manual @commands @review
Feature: Discuss findings beside the changed code
  Maintainers receive one review overview and a separate conversation for each located finding.

  Scenario Outline: Place findings on both sides of a contribution
    Given AI <kind> reviews are configured for this repository
    And a pull request from "<source>" for branch "contribution" is "waiting for checks"
    And the reviewer reports "findings on added and removed lines"
    When a maintainer with "write" access comments "/ab <command>"
    Then the <kind> review should contain one overview
    And the inline findings should be:
      | path      | line | side  |
      | main.go   | 2    | RIGHT |
      | worker.go | 2    | LEFT  |
    And no successful review should be posted as an ordinary comment

    @codereview
    Examples:
      | kind     | command    | source         |
      | code     | codereview | the repository |

    @secreview
    Examples:
      | kind     | command    | source         |
      | security | secreview  | the repository |

  @codereview
  Scenario: Preserve findings whose locations cannot be placed
    Given AI code reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the contribution renames a reviewed file
    And the reviewer reports "a finding outside the changed lines"
    When a maintainer with "write" access comments "/ab codereview"
    Then the code review should contain one overview
    And the inline findings should be:
      | path       | line | side  |
      | renamed.go | 2    | RIGHT |
      | worker.go  | 2    | LEFT  |
    And its overview should retain the finding at the unavailable location

  @codereview
  Scenario: Publish an overview when no supported findings remain
    Given AI code reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the reviewer reports "no supported findings"
    When a maintainer with "write" access comments "/ab codereview"
    Then the code review should describe no findings within the inspected scope
    And the native review should have no inline findings

  @codereview
  Scenario: Report malformed model output without submitting a review
    Given AI code reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the reviewer reports "an unreadable answer"
    When a maintainer with "write" access comments "/ab codereview"
    Then the code review should fail without publishing findings
    And its failure comment should explain "unreadable or invalid review result" with recovery advice
    And exactly 1 request should reach the AI service
    And no native review should be submitted

  @codereview
  Scenario: Keep review examples readable and model markup inert in every body
    Given AI code reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the reviewer reports "examples with untrusted markup"
    When a maintainer with "write" access comments "/ab codereview"
    Then the code review should escape model markup and preserve examples
    And every inline finding should escape model markup and preserve examples

  @codereview
  Scenario: Reuse a legacy review after restart
    Given AI code reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And a legacy App code review exists for the current revision
    And the worker has restarted
    When a maintainer with "write" access comments "/ab codereview"
    Then the original code review should remain without another model call
    And no native review should be submitted

  @codereview
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

  @codereview
  Scenario: Find a completed native review beyond the first history page
    Given AI code reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the pull request has 105 earlier native reviews
    And the pull request has already received a code review
    When a maintainer with "write" access comments "/ab codereview"
    Then the existing native review should prevent another model call

  @secreview
  Scenario: Keep code and security conversations independent
    Given AI security reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And security reviews and summaries are independently enabled
    And the reviewer reports "findings on added and removed lines"
    And the pull request has already received a code review
    When a maintainer with "write" access comments "/ab secreview"
    Then both review commands should have their own conversations

  @codereview
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
      | kind     | command    | author      |
      | code     | codereview | a human     |
      | code     | codereview | another App |

    @secreview
    Examples:
      | kind     | command    | author      |
      | security | secreview  | a human     |
      | security | secreview  | another App |

  @codereview
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

  @codereview
  Scenario: Find the App result after a full batch of forged markers
    Given AI code reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And 105 native code review lookalikes were posted by "another App"
    And the pull request has already received a code review
    When a maintainer with "write" access comments "/ab codereview"
    Then the existing native review should prevent another model call
