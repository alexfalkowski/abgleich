@manual
Feature: Start with a usable worker configuration
  A readable RSA private key is required at startup.
  Restored or replaced credentials take effect after restarting the worker.

  Scenario: Reject a CI completion check configured as a list
    Given the CI completion check is configured as a list
    When the operator starts the worker
    Then the worker should refuse to start

  Scenario: Reject an empty Dependabot repository exclusion
    Given Dependabot automation excludes repositories:
      | repository |
      |            |
    When the operator starts the worker
    Then the worker should refuse to start

  Scenario Outline: Reject a <state> App private key at startup
    Given the App private key is "<state>"
    When the operator starts the worker
    Then the worker should refuse to start

    Examples:
      | state       |
      | unavailable |
      | empty       |
      | invalid     |

  Scenario Outline: Start after restoring a <state> App private key
    Given the worker could not start with a "<state>" App private key
    When the worker restarts with valid App credentials
    Then the branch check should succeed

    Examples:
      | state       |
      | unavailable |
      | empty       |
      | invalid     |

  Scenario: Resume rebasing after replacing an unrecognized App key
    Given the worker cannot authenticate when checking branch "feature"
    When the worker restarts with valid App credentials
    Then branch "feature" should be rebased
    And the branch check should succeed
    And the worker should remain healthy

  Scenario: Require an App installation in configuration
    Given the App installation is not configured
    When the operator starts the worker
    Then the worker should refuse to start

  Scenario Outline: Reject an empty resolved webhook secret at startup
    Given the webhook secret comes from "<source>" with "<content>" content
    When the operator starts the worker
    Then the worker should refuse to start

    Examples:
      | source      | content    |
      | file        | empty      |
      | file        | whitespace |
      | environment | empty      |
