@github @manual @reconciliation
Feature: Auto-merge failure recovery
  A GitHub failure must not stop the worker from keeping branches current.

  Scenario Outline: Resume rebasing when GitHub recovers
    Given the worker could not rebase branch "feature" because "<reason>"
    When GitHub becomes available again
    Then branch "feature" should be rebased
    And the branch check should succeed
    And the worker should remain healthy

    Examples:
      | reason                                   |
      | pull requests could not be listed        |
      | the pull request could not be read       |
      | the pull request could not be rechecked  |
      | the token was rejected                   |
      | the rate limit was reached               |
      | the pull request response was malformed  |
      | the branches could not be compared       |
      | the comparison was incomplete            |
      | the pull request identity was missing    |
      | the branch update request failed         |
      | GitHub rejected the rebase               |
      | GitHub reported a partial rebase failure |
      | the rebase response was empty            |
      | the rebase response was malformed        |

  Scenario: Resume a failed rebase after restarting
    Given the worker could not rebase branch "feature" because "GitHub rejected the rebase"
    When the worker restarts after GitHub recovers
    Then branch "feature" should be rebased
    And the branch check should succeed

  Scenario: Avoid repeating a rebase whose response was lost
    Given GitHub rebased branch "feature" without confirming it to the worker
    When GitHub becomes available again
    Then branch "feature" should be rebased only once
    And only one rebase request should have been sent
    And the branch check should succeed

  Scenario: Retry a temporary GitHub failure within the same check
    Given GitHub has a stale auto-merge branch "feature"
    And GitHub temporarily cannot list pull requests
    When the worker checks auto-merge branches
    Then branch "feature" should be rebased
    And the branch check should succeed
    And no branch checks should have failed
