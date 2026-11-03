@github @manual @commands @codereview @secreview @review
Feature: Start manual reviews promptly
  Maintainers receive requested reviews without waiting for thread maintenance.

  Scenario Outline: Process a manual request between scheduled maintenance runs
    Given AI <kind> reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And <kind> reviews use infrequent thread maintenance
    When a maintainer with "write" access comments "/ab <command>"
    Then a <kind> review should arrive within 2 seconds
    And the <kind> review should identify the request and revisions
    And manual <kind> review should leave scheduled maintenance pending
    And exactly 1 request should reach the AI service

    Examples:
      | kind     | command    |
      | code     | codereview |
      | security | secreview  |

  Scenario Outline: Drain requests queued behind active inference
    Given AI <kind> reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And four maintainers are waiting for <kind> reviews
    When the AI service completes the active review
    Then all 4 accepted <kind> reviews should finish promptly
    And exactly 4 requests should reach the AI service
    And the worker should remain healthy

    Examples:
      | kind     |
      | code     |
      | security |
