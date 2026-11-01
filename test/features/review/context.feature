@github @manual @commands @codereview @review
Feature: Review large contributions with incomplete information
  The advisory review uses pull request metadata and every available changed-file patch.
  Missing patches remain explicit and incomplete pagination prevents inference.

  Background:
    Given AI code reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"

  Scenario Outline: Review every change in a large contribution
    Given the contribution contains "<changes>"
    When a maintainer with "write" access comments "/ab codereview"
    Then the code review should identify the request and revisions
    And the code reviewer should receive "<evidence>"

    Examples:
      | changes                        | evidence                        |
      | more than twenty changed files | the final changed file          |
      | large combined changes         | the end of the combined changes |
      | an oversized individual change | the end of the large change     |

  Scenario: Disclose a missing patch
    Given the contribution contains "a change without a text diff"
    And the AI code review answer has "no findings"
    When a maintainer with "write" access comments "/ab codereview"
    Then the code review should describe no findings within the inspected scope
    And the code reviewer should receive "the missing patch notice"

  Scenario: Withhold a review when some changes cannot be read
    Given GitHub can retrieve only the first part of the changes
    When a maintainer with "write" access comments "/ab codereview"
    Then code review collection should fail before sharing source

  Scenario Outline: Explain when the reviewer cannot complete an answer
    Given the AI code review answer has "<answer>"
    When a maintainer with "write" access comments "/ab codereview"
    Then the code review should fail without publishing findings
    And exactly 1 request should reach the AI service

    Examples:
      | answer                                      |
      | an empty answer                             |
      | a refusal alongside an answer               |
      | a request outside the approved capabilities |
