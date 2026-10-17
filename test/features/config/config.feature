@github @manual
Feature: GitHub credentials
  A readable, non-empty GitHub token is required at startup.
  Restored or replaced credentials take effect after restarting the worker.

  Scenario Outline: Reject a <state> token at startup
    Given the GitHub token is "<state>"
    When I attempt to start the system
    Then starting the system should raise an error

    Examples:
      | state       |
      | unavailable |
      | empty       |

  Scenario Outline: Start after restoring a <state> token
    Given the worker could not start with a "<state>" GitHub token
    When the worker restarts with valid credentials
    Then the branch check should succeed

    Examples:
      | state       |
      | unavailable |
      | empty       |

  Scenario: Resume rebasing after replacing an invalid token
    Given the worker cannot authenticate when checking branch "feature"
    When the worker restarts with valid credentials
    Then branch "feature" should be rebased
    And the branch check should succeed
    And the worker should remain healthy
