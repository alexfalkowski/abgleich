@github @manual @reconciliation
Feature: Scheduled auto-merge checks
  Slow or failed checks must not prevent later checks.
  Restarting the worker cancels the active check and resumes checking.

  Scenario Outline: Resume rebasing after a GitHub timeout
    Given a check for branch "feature" timed out while "<operation>"
    When GitHub becomes available again
    Then branch "feature" should be rebased
    And the branch check should succeed
    And the worker should remain healthy

    Examples:
      | operation             |
      | listing pull requests |
      | rebasing the branch   |

  Scenario: Warn when a successful check takes too long
    Given GitHub has a stale auto-merge branch "feature"
    And GitHub responds after the schedule interval
    When the worker checks auto-merge branches
    Then branch "feature" should be rebased
    And the branch check should succeed
    And the worker should warn about the slow check
    And branch checks should not overlap
    And no branch checks should have failed

  Scenario: Cancel an active check during a restart
    Given the worker is waiting for GitHub to check branch "feature"
    When the worker restarts
    Then the interrupted check should be cancelled
    And branch "feature" should be rebased
    And the branch check should succeed

  Scenario: Recover after repeated failures
    Given GitHub failures have interrupted successful branch checks
    When GitHub becomes available again
    Then the branch check should succeed
    And the worker should remain healthy
