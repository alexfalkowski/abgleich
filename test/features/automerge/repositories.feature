@github @manual @reconciliation
Feature: Owner-wide auto-merge reconciliation
  The worker discovers the owner's repositories on every sweep.
  Repository failures are isolated and GitHub throttling stops a sweep.

  Scenario: Rebase eligible branches across every repository page
    Given GitHub has repositories with stale auto-merge branches:
      | repository              | state               | branch  |
      | alexfalkowski/infraops  | public              | feature |
      | alexfalkowski/private   | private             | feature |
      | alexfalkowski/archived  | archived            | feature |
      | alexfalkowski/disabled  | disabled            | feature |
      | alexfalkowski/opted-out | auto-merge disabled | feature |
      | someone/foreign         | public              | feature |
    And the repositories span multiple pages
    When the worker checks auto-merge branches
    Then the repositories should have these branch outcomes:
      | repository              | branch  | outcome     |
      | alexfalkowski/infraops  | feature | rebased     |
      | alexfalkowski/private   | feature | rebased     |
      | alexfalkowski/archived  | feature | not rebased |
      | alexfalkowski/disabled  | feature | not rebased |
      | alexfalkowski/opted-out | feature | not rebased |
      | someone/foreign         | feature | not rebased |
    And the branch check should succeed
    And accepted rebases should identify their repositories

  Scenario: Discover a repository created after a successful sweep
    Given the worker has successfully checked the existing repositories
    When GitHub adds repository "alexfalkowski/new" with stale auto-merge branch "feature"
    Then branch "feature" in repository "alexfalkowski/new" should be rebased
    And the branch check should succeed

  Scenario: Reconcile an account without repositories
    Given GitHub has no repositories
    When the worker checks auto-merge branches
    Then the branch check should succeed
    And no repositories should have been checked

  Scenario: Reject an installation for another account
    Given GitHub has a stale auto-merge branch "feature"
    And the App installation belongs to account "someone"
    When the worker checks auto-merge branches
    Then the branch check should fail without affecting worker health
    And no repositories should have been checked

  Scenario: Finish discovery before rebasing any repository
    Given GitHub has repositories with stale auto-merge branches:
      | repository           | state  | branch  |
      | alexfalkowski/first  | public | feature |
      | alexfalkowski/second | public | feature |
    And the repositories span multiple pages
    And a later repository page is unavailable
    When the worker checks auto-merge branches
    Then the branch check should fail without affecting worker health
    And no repositories should have been checked

  Scenario: Continue after a repository permission failure
    Given GitHub has repositories with stale auto-merge branches:
      | repository            | state  | branch  |
      | alexfalkowski/blocked | public | feature |
      | alexfalkowski/ready   | public | feature |
    And repository "alexfalkowski/blocked" fails with "permission"
    When the worker checks auto-merge branches
    Then the repositories should have these branch outcomes:
      | repository            | branch  | outcome     |
      | alexfalkowski/blocked | feature | not rebased |
      | alexfalkowski/ready   | feature | rebased     |
    And the branch check should fail without affecting worker health
    And the failure should identify repository "alexfalkowski/blocked"

  Scenario Outline: Isolate repository settings failures
    Given GitHub has repositories with stale auto-merge branches:
      | repository            | state  | branch  |
      | alexfalkowski/blocked | public | feature |
      | alexfalkowski/ready   | public | feature |
    And repository "alexfalkowski/blocked" fails with "<failure>"
    When the worker checks auto-merge branches
    Then the repositories should have these branch outcomes:
      | repository            | branch  | outcome     |
      | alexfalkowski/blocked | feature | not rebased |
      | alexfalkowski/ready   | feature | rebased     |
    And the branch check should fail without affecting worker health
    And the failure should identify repository "alexfalkowski/blocked"

    Examples:
      | failure               |
      | repository settings   |
      | mismatched repository |

  Scenario Outline: Stop a sweep when GitHub reports throttling
    Given GitHub has repositories with stale auto-merge branches:
      | repository              | state  | branch  |
      | alexfalkowski/a-limited | public | feature |
      | alexfalkowski/a-limited | public | sibling |
      | alexfalkowski/z-later   | public | feature |
    And repository "alexfalkowski/z-later" fails with "permission"
    And repository "alexfalkowski/a-limited" fails with "<limit>"
    When the worker checks auto-merge branches
    Then the branch check should fail without affecting worker health
    And the first check should stop at repository "alexfalkowski/a-limited" before repository "alexfalkowski/z-later"
    And the first check should stop before pull request 2 in repository "alexfalkowski/a-limited"

    Examples:
      | limit                                 |
      | rate limit                            |
      | primary rate limit                    |
      | secondary rate limit                  |
      | secondary rate limit without metadata |
      | graphql rate limit                    |
      | graphql secondary limit               |

  Scenario: Resume checking all repositories after a timeout
    Given GitHub has repositories with stale auto-merge branches:
      | repository            | state  | branch  |
      | alexfalkowski/blocked | public | feature |
      | alexfalkowski/ready   | public | feature |
    And repository "alexfalkowski/blocked" fails with "timeout"
    And a repository timeout has interrupted a sweep
    When repository "alexfalkowski/blocked" recovers
    Then the repositories should have these branch outcomes:
      | repository            | branch  | outcome |
      | alexfalkowski/blocked | feature | rebased |
      | alexfalkowski/ready   | feature | rebased |
    And the branch check should succeed
    And the worker should remain healthy
