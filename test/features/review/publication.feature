@manual @commands @review-results
Feature: Publish independent review results safely
  Native reviews preserve separate feature conversations and reject ambiguous or mismatched publication outcomes.

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
  Scenario: Preserve findings when both reviews complete together
    Given AI code reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And both review commands are enabled
    And the AI code review answer has "conflicting verdicts"
    When maintainers request both reviews together
    Then the code review decision should be "CHANGES_REQUESTED"
    And the clean security review should preserve the code review's decision

  Scenario Outline: Reject mismatched verdicts without retrying publication
    Given AI <kind> reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the AI <kind> review answer has "<answer>"
    And GitHub returns a different review verdict
    When a maintainer with "write" access comments "/ab <command>"
    Then the <kind> review publication should be reported as failed
    And exactly 1 request should reach the AI service

    @codereview
    Examples:
      | kind | command    | answer      |
      | code | codereview | no findings |

    @secreview
    Examples:
      | kind     | command   | answer          |
      | security | secreview | inline findings |
