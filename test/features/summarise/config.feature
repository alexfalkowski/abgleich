@github @manual @summaries
Feature: Enable optional pull request summaries
  Operators can enable summaries with a supported AI service and usable credentials.
  Ordinary automation continues when summaries are disabled.

  Scenario Outline: Refuse to start with unusable AI credentials
    Given AI summaries are configured for this repository
    And the AI service credential is "<state>"
    When I attempt to start the system
    Then starting the system should raise an error

    Examples:
      | state       |
      | unavailable |
      | blank       |

  Scenario: Refuse an unsupported AI service
    Given AI summaries are configured for this repository
    And the operator selects an unsupported AI service
    When I attempt to start the system
    Then starting the system should raise an error

  Scenario: Require an AI service when summaries are enabled
    Given AI summaries are configured for this repository
    And the operator enables summaries without an AI service
    When I attempt to start the system
    Then starting the system should raise an error

  Scenario: Refuse an invalid trusted certificate for the AI service
    Given AI summaries are configured for this repository
    And the operator configures "an invalid trusted certificate" for the AI service
    When I attempt to start the system
    Then starting the system should raise an error

  Scenario: Continue ordinary automation with summaries disabled
    Given AI summaries are disabled
    And GitHub has a stale auto-merge branch "feature"
    When the worker checks auto-merge branches
    Then branch "feature" should be rebased
    And the branch check should succeed
    And the AI service should remain unused

  Scenario: Request a summary without custom scheduling
    Given AI summaries are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the operator uses the default summary schedule
    When a maintainer with "write" access comments "/ab summarise"
    Then the request should be accepted
    And the maintainer should receive a summary within 40 seconds
    And the summary should identify the request and the pull request revision
