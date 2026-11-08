@manual @commands @secreview
Feature: Choose whether maintainers can request security reviews
  Security reviews are separately enabled and require a configured model and provider.

  Scenario Outline: Refuse invalid security review configuration
    Given AI security reviews are configured for this repository
    And the security review configuration has "<problem>"
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

  Scenario: Leave disabled security review commands unused
    Given AI security reviews are configured for this repository
    And the security review configuration has "disabled reviews"
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    When a maintainer with "write" access comments "/ab secreview"
    Then the command comment should show "receipt and rejection" feedback
    And the AI service should remain unused

  Scenario: Request a security review with the default schedule
    Given AI security reviews are configured for this repository
    And the security review configuration has "the default schedule"
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    When a maintainer with "write" access comments "/ab secreview"
    Then a security review should arrive within 40 seconds
    And the security review should identify the request and revisions
