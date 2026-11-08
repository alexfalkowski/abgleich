@manual @commands @review-results
Feature: Inspect selected dependency source lines
  Reviewers receive the requested lines while inspection records reflect only the source actually supplied.

  Background:
    Given repository source and public Go dependencies are available for review
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"

  Scenario Outline: Read a relevant dependency excerpt without sending unrelated source
    Given the reviewer needs "related source excerpts"
    When a maintainer with "write" access comments "/ab <command>"
    Then the <kind> reviewer should receive only the requested source lines

    @codereview
    Examples:
      | command    | kind |
      | codereview | code |

    @secreview
    Examples:
      | command   | kind     |
      | secreview | security |

  @codereview
  Scenario: Reject an invalid excerpt before fetching source
    Given the reviewer needs "an invalid excerpt"
    When a maintainer with "write" access comments "/ab codereview"
    Then the code review should disclose external source note "invalid source line range"
    And the code review decision should be "APPROVED"
    And the excerpt should not claim inspection of the related file
    And the public dependency proxy should receive 0 requests without credentials

  @codereview
  Scenario: Preserve an empty dependency excerpt without inventing inspected lines
    Given the reviewer needs "an empty excerpt"
    When a maintainer with "write" access comments "/ab codereview"
    Then the code review should have no context limitations
    And the dependency excerpt should contain no source lines
    And the excerpt should not claim inspection of the related file
