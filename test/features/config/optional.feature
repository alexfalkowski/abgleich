@github @manual @commands @summaries @codereview @secreview
Feature: Leave optional commands and jobs disabled unless configured
  Maintainers receive rejection feedback for disabled AI commands.
  Merge automation remains available without optional background jobs.

  Background:
    Given GitHub App comment commands are configured
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"

  Scenario Outline: Decline disabled commands regardless of provider configuration
    Given optional AI commands are disabled with the provider "<provider>"
    When a maintainer with "write" access comments "/ab <command>"
    Then the comment delivery should return HTTP 204
    And the command comment should show "receipt and rejection" feedback
    And the AI service should remain unused
    And only merge automation should run in the background
    And the worker should remain healthy

    Examples:
      | command    | provider   |
      | codereview | configured |
      | codereview | omitted    |
      | secreview  | configured |
      | secreview  | omitted    |
      | summarise  | configured |
      | summarise  | omitted    |
