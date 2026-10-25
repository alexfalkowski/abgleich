@github @manual
Feature: Preserve work during branch cleanup
  Branch cleanup respects new work and maintainer decisions.
  If GitHub cannot confirm that deletion is safe, the branch is preserved.

  Background:
    Given branch cleanup is configured
    And GitHub has cleanup branches:
      | branch | commit age in days | pull request |
      | old    | 31                 | closed       |

  Scenario Outline: Preserve branches when their safety cannot be established
    Given GitHub cannot confirm "<information>" during cleanup
    When the worker runs branch cleanup
    Then cleanup should preserve exactly "old"
    And cleanup should make no deletion request
    And the worker should remain healthy

    Examples:
      | information                       |
      | whether branches are protected    |
      | a branch's latest commit          |
      | a branch's pull request history   |
      | the repository's identity         |
      | the complete branch listing       |
      | the current branch's protection   |
      | the current branch's identity     |
      | current pull requests             |
      | new pull requests before deletion |
      | a new contribution's revision     |
      | a contribution's history revision |
      | a contribution's history branch   |

  Scenario Outline: Preserve work when an active contribution cannot be identified
    Given GitHub has cleanup branches:
      | branch | commit age in days | pull request |
      | active | 90                 | open         |
    And GitHub cannot confirm "<information>" during cleanup
    When the worker runs branch cleanup
    Then cleanup should preserve exactly "old, active"
    And cleanup should make no deletion request

    Examples:
      | information                         |
      | an active contribution's revision   |
      | an active contribution's repository |

  Scenario Outline: Respect maintainer changes during cleanup
    Given a maintainer "<change>" while cleanup is running
    When the worker runs branch cleanup
    Then cleanup should preserve exactly "old"
    And cleanup should make no deletion request

    Examples:
      | change                       |
      | protects the branch          |
      | makes the branch the default |
      | opts the repository out      |
      | withdraws repository access  |

  Scenario: Preserve an abandoned branch when its contribution is reopened
    Given an author reopens their contribution while cleanup is running
    When the worker runs branch cleanup
    Then cleanup should preserve exactly "old"
    And cleanup should make no deletion request
    And the cleanup check should succeed

  Scenario Outline: Preserve branches when their recorded merge cannot be verified
    Given branch "old" was merged in a later pull request
    And GitHub cannot confirm "<information>" during cleanup
    When the worker runs branch cleanup
    Then cleanup should preserve exactly "old"
    And cleanup should make no deletion request

    Examples:
      | information             |
      | the recorded merge      |
      | the merged contribution |

  Scenario: Preserve a branch when a pull request opens during cleanup
    Given a new pull request opens while cleanup is running
    When the worker runs branch cleanup
    Then cleanup should preserve exactly "old"
    And cleanup should make no deletion request

  Scenario: Preserve new work pushed during cleanup
    Given an author pushes new work while cleanup is running
    When the worker runs branch cleanup
    Then cleanup should preserve exactly "old"
    And the worker should remain healthy

  Scenario: Preserve commits pushed before cleanup rechecks the branch
    Given an author pushes new work before the branch is rechecked
    When the worker runs branch cleanup
    Then cleanup should preserve exactly "old"
    And cleanup should make no deletion request

  Scenario: Preserve work that changes while a busy repository is being listed
    Given GitHub has cleanup branches:
      | branch | commit age in days | pull request |
      | z-old  | 31                 | none         |
    And the repository has many branches and past pull requests
    And an author pushes while GitHub is listing the repository
    When the worker runs branch cleanup
    Then cleanup should preserve exactly "old, z-old"
    And cleanup should make no deletion request

  Scenario: Leave remaining branches alone when the cleanup deadline expires
    Given GitHub has cleanup branches:
      | branch | commit age in days | pull request |
      | z-old  | 31                 | none         |
    And the operator's cleanup deadline expires while GitHub is slow
    When the worker runs branch cleanup
    Then cleanup should preserve exactly "old, z-old"
    And cleanup should make no deletion request
    And the worker should remain healthy

  Scenario: Preserve a branch when its recheck exceeds the request timeout
    Given GitHub takes longer than the request timeout to recheck a branch
    When the worker runs branch cleanup
    Then cleanup should preserve exactly "old"
    And cleanup should make no deletion request
    And the cleanup check should report a branch request timeout
    And the worker should remain healthy

  Scenario Outline: Clean other branches when one branch cannot be deleted
    Given GitHub has cleanup branches:
      | branch | commit age in days | pull request |
      | z-old  | 31                 | none         |
    And GitHub cannot delete branch "old" because "<reason>"
    When the worker runs branch cleanup
    Then cleanup should delete exactly "z-old"
    And cleanup should preserve exactly "old"
    And the worker should remain healthy

    Examples:
      | reason                  |
      | the deletion is refused |
      | GitHub is unavailable   |

  Scenario: Clean other branches when one contribution's history is unavailable
    Given GitHub has cleanup branches:
      | branch | commit age in days | pull request |
      | z-old  | 31                 | none         |
    And GitHub cannot retrieve the contribution history for branch "old"
    When the worker runs branch cleanup
    Then cleanup should delete exactly "z-old"
    And cleanup should preserve exactly "old"
    And the worker should remain healthy

  Scenario: Keep a completed deletion after its confirmation is lost
    Given GitHub deleted branch "old" without confirming the deletion
    When the worker restarts after cleanup recovers
    Then branch "old" should stay deleted without another deletion request

  Scenario: Resume cleanup when GitHub recovers
    Given cleanup has failed to delete branch "old" repeatedly
    When GitHub accepts branch deletion again
    Then cleanup should delete exactly "old"
    And the cleanup check should succeed
    And the worker should remain healthy
