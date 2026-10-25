@github @manual
Feature: Clean up repositories independently
  A repository's unavailable history must not put its branches at risk or stop
  cleanup of another opted-in repository.

  Background:
    Given branch cleanup is configured
    And GitHub has cleanup branches:
      | repository            | branch    | commit age in days | pull request |
      | alexfalkowski/blocked | uncertain | 31                 | none         |
      | alexfalkowski/healthy | abandoned | 31                 | none         |

  Scenario: Clean up a healthy repository when another cannot be inspected
    Given GitHub cannot list branches in repository "alexfalkowski/blocked"
    When the worker runs branch cleanup
    Then cleanup should delete exactly "abandoned"
    And cleanup should preserve exactly "uncertain"
    And the worker should remain healthy

  Scenario: Preserve all branches when repository discovery is incomplete
    Given GitHub cannot finish discovering the cleanup repositories
    When the worker runs branch cleanup
    Then cleanup should preserve exactly "uncertain, abandoned"
    And cleanup should make no deletion request

