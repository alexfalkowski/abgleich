@manual @commands @codereview
Feature: Choose whether maintainers can request code reviews
  Code reviews are separately enabled and require a configured model and provider.

  Scenario Outline: Refuse invalid review configuration
    Given AI code reviews are configured for this repository
    And the code review configuration has "<problem>"
    When the operator starts the worker
    Then the worker should refuse to start

    Examples:
      | problem                     |
      | no AI provider              |
      | no model                    |
      | an empty model              |
      | an invalid interval         |
      | an invalid timeout          |
      | an invalid reasoning effort |

  Scenario: Leave disabled review commands unused
    Given AI code reviews are configured for this repository
    And the code review configuration has "disabled reviews"
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    When a maintainer with "write" access comments "/ab codereview"
    Then the command comment should show "receipt and rejection" feedback
    And the AI service should remain unused

  Scenario: Request a code review with the default schedule
    Given AI code reviews are configured for this repository
    And the code review configuration has "the default schedule"
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    When a maintainer with "write" access comments "/ab codereview"
    Then a code review should arrive within 40 seconds
    And the code review should identify the request and revisions
