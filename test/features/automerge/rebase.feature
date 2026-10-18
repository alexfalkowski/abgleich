@github @manual
Feature: Auto-merge branch rebasing
  Stale auto-merge branches are rebased onto their base branches.
  Ineligible branches and concurrent author changes are respected.

  Scenario: Rebase a stale auto-merge branch
    Given GitHub has a stale auto-merge branch "team/feature"
    When the worker checks auto-merge branches
    Then branch "team/feature" should be rebased
    And the branch check should succeed
    And the worker should report these accepted rebases:
      | pull request | branch       |
      | 1            | team/feature |

  Scenario Outline: Leave ineligible branches alone
    Given the pull request for branch "feature" is "<condition>"
    When the worker checks auto-merge branches
    Then branch "feature" should remain unchanged
    And the branch check should succeed

    Examples:
      | condition             |
      | not set to auto-merge |
      | a draft               |
      | from a fork           |
      | closed                |
      | up to date            |

  Scenario Outline: Respect auto-merge being disabled before <operation>
    Given GitHub has a stale auto-merge branch "feature"
    And auto-merge is disabled before "<operation>"
    When the worker checks auto-merge branches
    Then branch "feature" should remain unchanged
    And the branch check should succeed

    Examples:
      | operation  |
      | comparison |
      | rebase     |

  Scenario: Rebase branches from every page
    Given GitHub has stale auto-merge branches:
      | branch |
      | first  |
      | second |
    And the pull requests span multiple pages
    When the worker checks auto-merge branches
    Then the branches should have these outcomes:
      | branch | outcome |
      | first  | rebased |
      | second | rebased |
    And the branch check should succeed

  Scenario: Continue rebasing when another branch has a conflict
    Given GitHub has stale auto-merge branches:
      | branch  |
      | blocked |
      | ready   |
    And GitHub cannot rebase branch "blocked"
    When the worker checks auto-merge branches
    Then the branch check should fail
    And the branches should have these outcomes:
      | branch  | outcome     |
      | ready   | rebased     |
      | blocked | not rebased |
    And the worker should report these accepted rebases:
      | pull request | branch |
      | 2            | ready  |

  Scenario: Preserve concurrent author changes
    Given an author's commit interrupted the rebase of branch "feature"
    When the worker retries the changed branch
    Then branch "feature" should be rebased
    And the rebase should use the author's latest commit
    And the branch check should succeed

  Scenario: Leave a successfully rebased branch alone
    Given GitHub has a stale auto-merge branch "feature"
    When the worker checks auto-merge branches
    Then branch "feature" should be rebased only once

  Scenario Outline: Defer rebasing when a pull request changes
    Given GitHub has a stale auto-merge branch "feature"
    And the pull request's "<field>" changes before rebase
    When the worker checks auto-merge branches
    Then the worker should defer rebasing branch "feature"

    Examples:
      | field       |
      | identity    |
      | head branch |
      | head commit |
      | base branch |
