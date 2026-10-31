@github @manual @commands @codereview
Feature: Review the complete patch context supplied by GitHub
  The advisory review uses pull request metadata and every available changed-file patch.
  Missing patches remain explicit and incomplete pagination prevents inference.

  Background:
    Given AI code reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"

  Scenario Outline: Share every changed-file patch without local caps
    Given the contribution contains "<changes>"
    When a maintainer with "write" access comments "/ab codereview"
    Then the code review should identify the request and revisions
    And the review context should include "<evidence>"

    Examples:
      | changes                        | evidence          |
      | more than twenty changed files | File: file21.go   |
      | large combined changes         | test.go-patch-end |
      | an oversized individual change | main.go-patch-end |

  Scenario: Disclose a missing patch
    Given the contribution contains "a change without a text diff"
    And the AI review answer has "no findings"
    When a maintainer with "write" access comments "/ab codereview"
    Then the review should describe no findings within the inspected scope
    And the review context should include "Patch unavailable from GitHub."

  Scenario: Reject incomplete changed-file pagination
    Given GitHub can retrieve only the first part of the changes
    When a maintainer with "write" access comments "/ab codereview"
    Then review collection should fail before sharing source

  Scenario Outline: Report unusable provider responses
    Given the AI review answer has "<answer>"
    When a maintainer with "write" access comments "/ab codereview"
    Then the code review should fail without publishing findings
    And exactly 1 request should reach the AI service

    Examples:
      | answer                 |
      | empty answer           |
      | mixed refusal          |
      | unexpected tool output |
