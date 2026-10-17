@startup
Feature: Observability

  Observability is a measure of how well internal states of a system can be inferred by knowledge of its external outputs.

  Scenario Outline: <endpoint> with HTTP
    When the system requests the "<endpoint>" with HTTP
    Then the system should respond with a healthy status with HTTP

    Examples:
      | endpoint  |
      | health    |
      | liveness  |
      | readiness |

  Scenario: Metrics
    When the system requests the "metrics" with HTTP
    Then the system should respond with metrics
