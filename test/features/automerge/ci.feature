@github @manual @reconciliation
Feature: Let CI synchronize branches before event-driven rebasing
  CI reaches its own sync stage before Abgleich repairs a branch on notification.
  Missing completion evidence leaves repair to the scheduled recovery sweep.

  Background:
    Given GitHub event checks are configured between hourly sweeps
    And the worker has successfully checked the existing repositories

  Scenario Outline: Leave an unfinished CI workflow alone
    Given GitHub has a stale auto-merge branch "feature"
    And CI for branch "feature" is "<phase>"
    When GitHub delivers a "push" notification with action ""
    Then the event delivery should return HTTP 204
    And exactly 1 event checks should finish
    And no event mutations should reach GitHub

    Examples:
      | phase                       |
      | not started                 |
      | building                    |
      | between validation and sync |
      | syncing                     |
      | failed before sync          |
      | final status pending        |
      | check queued                |
      | check running               |
      | other check running         |
      | check missing conclusion    |
      | unknown status              |

  Scenario Outline: Repair a stale branch after explicit workflow completion
    Given GitHub has a stale auto-merge branch "feature"
    And CI for branch "feature" is "<phase>"
    When GitHub reports the final CI result for branch "feature"
    Then the event delivery should return HTTP 204
    And branch "feature" should be rebased
    And exactly 1 event checks should finish
    And only one mutation request should have been sent

    Examples:
      | phase                 |
      | finished              |
      | finished with failure |
      | finished with error   |
      | check finished        |
      | check failed          |
      | check cancelled       |

  Scenario: An individual job finishing does not authorize rebasing
    Given GitHub has a stale auto-merge branch "feature"
    And CI for branch "feature" is "between validation and sync"
    When GitHub reports a successful validation job for branch "feature"
    Then the event delivery should return HTTP 204
    And exactly 1 event checks should finish
    And no event mutations should reach GitHub

  Scenario: Reconsider a branch when its final CI result arrives
    Given an event check left branch "feature" waiting for CI
    When CI finishes its sync stage for branch "feature" and reports completion
    Then branch "feature" should be rebased
    And exactly 2 event checks should finish
    And only one mutation request should have been sent
    And no further scheduled sweep should have completed

  Scenario: Leave CI's successful branch update alone
    Given an event check left branch "feature" waiting for CI
    When CI synchronizes and pushes branch "feature"
    Then exactly 2 event checks should finish
    And CI's pushed head for branch "feature" should remain untouched
    And no event mutations should reach GitHub

  Scenario: A base update after CI finished prompts repair
    Given CI finished syncing branch "feature" before its base advanced
    When GitHub delivers a "push" notification with action ""
    Then branch "feature" should be rebased
    And only one mutation request should have been sent

  Scenario: An old commit's completed checks do not authorize a new head
    Given GitHub has a stale auto-merge branch "feature"
    And CI pushed a new stale head for branch "feature" without starting checks
    When GitHub reports the final CI result for branch "feature"
    Then exactly 1 event checks should finish
    And no event mutations should reach GitHub

  Scenario: Respect CI pushing while completion evidence is being read
    Given GitHub has a stale auto-merge branch "feature"
    And the labeled operation fails with "CI push during observation"
    When GitHub reports the final CI result for branch "feature"
    Then the event check should fail
    And CI's pushed head for branch "feature" should remain untouched
    And no event mutations should reach GitHub

  Scenario: Respect a hold applied while completion evidence is being read
    Given GitHub has a stale auto-merge branch "feature"
    And the labeled operation fails with "hold during CI observation"
    When GitHub reports the final CI result for branch "feature"
    Then exactly 1 event checks should finish
    And no event mutations should reach GitHub
    And branch "feature" should retain the "hold" label

  Scenario: Collect every page before considering CI complete
    Given GitHub has a stale auto-merge branch "feature"
    And CI for branch "feature" is "multiple checks finished"
    And the CI observations span multiple pages
    When GitHub reports the final CI result for branch "feature"
    Then branch "feature" should be rebased
    And only one mutation request should have been sent

  Scenario Outline: Do not infer completion from failed or incomplete reads
    Given GitHub has a stale auto-merge branch "feature"
    And CI for branch "feature" is "multiple checks finished"
    And the CI observations span multiple pages
    And the labeled operation fails with "<failure>"
    When GitHub reports the final CI result for branch "feature"
    Then the event check should fail
    And no event mutations should reach GitHub

    Examples:
      | failure               |
      | statuses access       |
      | checks access         |
      | statuses page         |
      | checks page           |
      | wrong status revision |
      | wrong check revision  |
      | missing status total  |
      | missing check total   |
      | missing statuses      |
      | missing checks        |
      | incomplete statuses   |
      | incomplete checks     |

  Scenario Outline: Require explicit event rebase configuration
    Given the event completion check is "<configuration>"
    And GitHub has a stale auto-merge branch "feature"
    When GitHub reports the final CI result for branch "feature"
    Then exactly 1 event checks should finish
    And no event mutations should reach GitHub

    Examples:
      | configuration     |
      | omitted           |
      | empty             |
      | unreported        |
      | differently cased |

  Scenario: Scheduled recovery repairs a branch whose failed CI never reached sync
    Given an event check left branch "feature" waiting after CI failed before sync
    When the worker restarts
    Then branch "feature" should be rebased
    And the branch check should succeed
