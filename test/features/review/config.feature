@manual @commands @review-results
Feature: Keep source configuration independent of review consent
  Configuring source access does not enable optional review commands.

  Background:
    Given repository source and public Go dependencies are available for review
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"

  Scenario Outline: Configuring dependency lookup does not enable reviews
    Given dependency lookup is configured but reviews are disabled
    When a maintainer with "write" access comments "/ab <command>"
    Then the command comment should show "receipt and rejection" feedback
    And the AI service should remain unused

    @codereview
    Examples:
      | command    |
      | codereview |

    @secreview
    Examples:
      | command    |
      | secreview  |
