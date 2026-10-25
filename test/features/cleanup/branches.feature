@github @manual
Feature: Delete merged and inactive branches
  Configured cleanup removes leftover merged branches and branches whose latest
  commit is older than thirty days. Maintainers can preserve long-lived branches.

  Background:
    Given branch cleanup is configured

  Scenario: Remove merged leftovers and thirty-day inactive branches
    Given GitHub has cleanup branches:
      | branch         | commit age in days | pull request |
      | merged         | 1                  | merged       |
      | team/abandoned | 31                 | none         |
      | closed         | 31                 | closed       |
      | recent         | 29                 | none         |
    When the worker runs branch cleanup
    Then cleanup should delete exactly "merged, team/abandoned, closed"
    And cleanup should preserve exactly "recent"
    And the cleanup check should succeed
    And the worker should remain healthy

  Scenario: Preserve default, protected, excluded, and open pull request branches
    Given GitHub has cleanup branches:
      | branch      | commit age in days | pull request |
      | master      | 90                 | none         |
      | protected   | 90                 | none         |
      | develop     | 90                 | none         |
      | open        | 90                 | open         |
      | integration | 90                 | none         |
    And branch "protected" is protected from cleanup
    And cleanup excludes branch "develop"
    And an open pull request targets branch "integration"
    When the worker runs branch cleanup
    Then cleanup should preserve exactly "master, protected, develop, open, integration"
    And cleanup should make no deletion request
    And the cleanup check should succeed

  Scenario: Preserve new commits made after a pull request merged
    Given GitHub has cleanup branches:
      | branch | commit age in days | pull request |
      | reused | 90                 | merged       |
    And branch "reused" received new commits after merging
    When the worker runs branch cleanup
    Then cleanup should preserve exactly "reused"
    And cleanup should make no deletion request
    And the cleanup check should succeed

  Scenario: Leave cleanup disabled unless configured
    Given GitHub has cleanup branches:
      | branch | commit age in days | pull request |
      | old    | 90                 | merged       |
    And branch cleanup is not configured
    When the worker runs branch cleanup
    Then no cleanup job should run
    And cleanup should preserve exactly "old"
    And cleanup should make no deletion request

  Scenario: Configure a longer inactivity period
    Given GitHub has cleanup branches:
      | branch | commit age in days | pull request |
      | old    | 31                 | none         |
    And cleanup waits for 60 inactive days
    When the worker runs branch cleanup
    Then cleanup should preserve exactly "old"
    And cleanup should make no deletion request
    And the cleanup check should succeed

  Scenario: Enable cleanup without custom schedules or age limits
    Given GitHub has cleanup branches:
      | branch    | commit age in days | pull request |
      | abandoned | 31                 | none         |
      | recent    | 29                 | none         |
    And the operator uses the default cleanup settings
    When the worker runs branch cleanup
    Then cleanup should delete exactly "abandoned"
    And cleanup should preserve exactly "recent"
    And the cleanup check should succeed

  Scenario Outline: Respect repository eligibility
    Given GitHub has cleanup branches:
      | branch | commit age in days | pull request |
      | old    | 90                 | none         |
    And the cleanup repository is "<state>"
    When the worker runs branch cleanup
    Then cleanup should preserve exactly "old"
    And cleanup should make no deletion request
    And the cleanup check should succeed

    Examples:
      | state     |
      | opted out |
      | archived  |
      | disabled  |

  Scenario: Clean up all eligible branches in a busy repository
    Given GitHub has cleanup branches:
      | branch | commit age in days | pull request |
      | a-old  | 1                  | closed       |
      | b-old  | 31                 | none         |
    And the repository has many branches and past pull requests
    And branch "a-old" was merged in a later pull request
    When the worker runs branch cleanup
    Then cleanup should delete exactly "a-old, b-old"
    And the cleanup check should succeed

  Scenario: Clean up abandoned branches when a busy listing repeats a branch
    Given GitHub has cleanup branches:
      | branch | commit age in days | pull request |
      | a-old  | 31                 | none         |
      | b-old  | 31                 | none         |
    And the repository has many branches and past pull requests
    And GitHub repeats a branch while listing the repository
    When the worker runs branch cleanup
    Then cleanup should delete exactly "a-old, b-old"
    And the cleanup check should succeed

  Scenario: Cleanup runs independently of merging and rebasing
    Given GitHub has cleanup branches:
      | branch | commit age in days | pull request |
      | old    | 31                 | none         |
    And GitHub has a stale auto-merge branch "feature"
    When the worker runs branch cleanup
    Then cleanup should delete exactly "old"
    And branch "feature" should be rebased
    And the cleanup check should succeed
