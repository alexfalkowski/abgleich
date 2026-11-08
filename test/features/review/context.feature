@manual @commands @codereview
Feature: Collect complete changed-file review context
  Reviews receive all available patches and changed-file text at the reviewed head or base revision.

  Scenario Outline: Review every change in a large contribution
    Given AI code reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the contribution contains "<changes>"
    When a maintainer with "write" access comments "/ab codereview"
    Then the code review should identify the request and revisions
    And the code reviewer should receive "<evidence>"

    Examples:
      | changes                        | evidence                        |
      | more than twenty changed files | the final changed file          |
      | large combined changes         | the end of the combined changes |
      | an oversized individual change | the end of the large change     |

  Scenario: Disclose a missing patch
    Given AI code reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the contribution contains "a change without a text diff"
    And the AI code review answer has "no findings"
    When a maintainer with "write" access comments "/ab codereview"
    Then the code review should describe no findings within the inspected scope
    And the code reviewer should receive "the missing patch notice"

  Scenario: Withhold a review when some changes cannot be read
    Given AI code reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And GitHub can retrieve only the first part of the changes
    When a maintainer with "write" access comments "/ab codereview"
    Then code review collection should fail before sharing source

  @review-results
  Scenario: Inspect deleted source at the reviewed base
    Given repository source and public Go dependencies are available for review
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the contribution contains "a deleted source file"
    When a maintainer with "write" access comments "/ab codereview"
    Then the code review should explain the related implementation
    And the reviewer should receive complete files at the reviewed revisions
    And deleted source should be read from the reviewed base revision

  @review-results
  Scenario: Identify an excluded deleted binary at the base revision
    Given AI code reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the contribution has a "deleted" binary file
    And the AI code review answer has "no findings"
    When a maintainer with "write" access comments "/ab codereview"
    Then the code review decision should be "APPROVED"
    And the deleted binary file should be identified at the reviewed base revision
