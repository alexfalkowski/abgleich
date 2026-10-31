@startup
Feature: Monitor worker health
  Operators can check whether the worker is healthy and collect monitoring metrics.

  Scenario Outline: Check the worker's <endpoint>
    When an operator checks worker "<endpoint>"
    Then the worker should report healthy

    Examples:
      | endpoint  |
      | health    |
      | liveness  |
      | readiness |

  Scenario: Collect monitoring metrics
    When an operator checks worker "metrics"
    Then the operator should receive monitoring metrics
