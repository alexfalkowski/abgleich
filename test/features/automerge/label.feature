@github @manual @merges
Feature: Labeled pull request automation
  Maintainers opt pull requests into squash merging with the automerge label.
  Fork branches remain under their authors' control.

  Scenario Outline: Enable labeled auto-merge while requirements are pending
    Given a pull request from "<source>" for branch "contribution" is "waiting for checks"
    And branch "contribution" has the "bug" label
    And branch "contribution" has the "automerge" label
    When the worker checks auto-merge branches
    Then branch "contribution" should have squash auto-merge enabled
    And the branch check should succeed
    And the worker should report "enabled pull request auto-merge" for branch "contribution"
    And only one merge action should occur for branch "contribution"

    Examples:
      | source         |
      | a fork         |
      | the repository |

  Scenario Outline: Merge a ready labeled pull request
    Given a pull request from "<source>" for branch "contribution" is "ready to merge"
    And branch "contribution" has the "automerge" label
    When the worker checks auto-merge branches
    Then branch "contribution" should be squash-merged
    And the branch check should succeed
    And the worker should report "merged pull request" for branch "contribution"
    And only one merge action should occur for branch "contribution"

    Examples:
      | source         |
      | a fork         |
      | the repository |

  Scenario Outline: Require the exact opt-in label
    Given a pull request from "a fork" for branch "contribution" is "ready to merge"
    And branch "contribution" has the "<label>" label
    When the worker checks auto-merge branches
    Then no pull request mutations should be requested
    And the branch check should succeed

    Examples:
      | label           |
      | bug             |
      | automerge:ready |
      | AUTOMERGE       |

  Scenario: Leave an unlabeled external contribution alone
    Given a pull request from "a fork" for branch "contribution" is "ready to merge"
    When the worker checks auto-merge branches
    Then no pull request mutations should be requested
    And the branch check should succeed

  Scenario Outline: Preserve common pull request eligibility
    Given a pull request from "a fork" for branch "contribution" is "<condition>"
    And branch "contribution" has the "automerge" label
    When the worker checks auto-merge branches
    Then no pull request mutations should be requested
    And the branch check should succeed

    Examples:
      | condition |
      | a draft   |
      | closed    |

  Scenario Outline: Leave pending requirements to GitHub
    Given a pull request from "a fork" for branch "contribution" is "<condition>"
    And branch "contribution" has the "automerge" label
    When the worker checks auto-merge branches
    Then branch "contribution" should have squash auto-merge enabled
    And the branch check should succeed
    And only one merge action should occur for branch "contribution"

    Examples:
      | condition            |
      | of unknown readiness |
      | missing mergeability |
      | conflicting          |
      | failing checks       |

  Scenario: Rebase a labeled branch in the same repository on a later check
    Given a pull request from "the repository" for branch "contribution" is "behind its base"
    And branch "contribution" has the "automerge" label
    When the worker checks auto-merge branches
    Then branch "contribution" should have squash auto-merge enabled
    And branch "contribution" should be rebased
    And the branch check should succeed
    And enabling auto-merge and rebasing should occur in separate checks

  Scenario: Leave a labeled fork branch under the contributor's control
    Given a pull request from "a fork" for branch "contribution" is "behind its base"
    And branch "contribution" has the "automerge" label
    When the worker checks auto-merge branches
    Then branch "contribution" should have squash auto-merge enabled
    And only one merge action should occur for branch "contribution"
    And no rebase should be requested

  Scenario: Preserve an existing fork auto-merge
    Given a pull request from "a fork" for branch "contribution" is "behind with auto-merge enabled"
    And branch "contribution" has the "automerge" label
    When the worker checks auto-merge branches
    Then no pull request mutations should be requested
    And the branch check should succeed

  Scenario: Preserve repository opt-in
    Given the repository has auto-merge disabled
    And a pull request from "a fork" for branch "contribution" is "ready to merge"
    And branch "contribution" has the "automerge" label
    When the worker checks auto-merge branches
    Then no pull request mutations should be requested
    And the branch check should succeed

  Scenario: Respect the repository's merge methods
    Given the repository has squash merging disabled
    And a pull request from "a fork" for branch "contribution" is "ready to merge"
    And branch "contribution" has the "automerge" label
    When the worker checks auto-merge branches
    Then the branch check should fail without affecting worker health
    And no pull request mutations should be requested

  Scenario Outline: Recheck labeled pull requests before acting
    Given a pull request from "a fork" for branch "contribution" is "<condition>"
    And branch "contribution" has the "automerge" label
    And its "<field>" changes before labeled automation
    When the worker checks auto-merge branches
    Then no pull request mutations should be requested

    Examples:
      | condition          | field           |
      | ready to merge     | identity        |
      | ready to merge     | number          |
      | ready to merge     | head commit     |
      | ready to merge     | head branch     |
      | ready to merge     | base branch     |
      | ready to merge     | head repository |
      | ready to merge     | base repository |
      | ready to merge     | state           |
      | ready to merge     | draft           |
      | ready to merge     | labels          |
      | waiting for checks | head commit     |
      | waiting for checks | labels          |

  Scenario: Enable labeled auto-merge when requirements change before merging
    Given a pull request from "a fork" for branch "contribution" is "ready to merge"
    And branch "contribution" has the "automerge" label
    And its "readiness" changes before labeled automation
    When the worker checks auto-merge branches
    Then branch "contribution" should have squash auto-merge enabled
    And only one merge action should occur for branch "contribution"

  Scenario Outline: Guard labeled actions against a concurrent push
    Given a pull request from "a fork" for branch "contribution" is "<condition>"
    And branch "contribution" has the "automerge" label
    And the labeled operation fails with "concurrent push"
    When the worker checks auto-merge branches
    Then the branch check should fail without affecting worker health
    And no merge actions should be accepted

    Examples:
      | condition          |
      | ready to merge     |
      | waiting for checks |

  Scenario Outline: Respect revoked auto-merge authorization
    Given a pull request from "a fork" for branch "contribution" is "<condition>"
    And branch "contribution" has the "automerge" label
    And GitHub disabled auto-merge for branch "contribution" "<revocation>"
    When the worker checks auto-merge branches
    Then no pull request mutations should be requested
    And auto-merge should remain disabled for branch "contribution"
    And the branch check should succeed

    Examples:
      | condition          | revocation                                  |
      | waiting for checks | after the contributor pushes                |
      | ready to merge     | after the contributor pushes                |
      | waiting for checks | after the maintainer disables it            |
      | waiting for checks | at the same time as the label is reapplied   |
      | waiting for checks | and a different label is applied            |

  Scenario: Preserve revocation across a worker restart
    Given the worker previously enabled labeled auto-merge before the contributor updated branch "contribution"
    When the worker checks auto-merge branches
    Then auto-merge should remain disabled for branch "contribution"
    And only one merge action should occur for branch "contribution"
    And only one mutation request should have been sent
    And the branch check should succeed

  Scenario: Renew authorization by reapplying the label
    Given a pull request from "a fork" for branch "contribution" is "waiting for checks"
    And branch "contribution" has the "automerge" label
    And GitHub disabled auto-merge for branch "contribution" "and the label is reapplied"
    And the automation events span multiple pages
    When the worker checks auto-merge branches
    Then branch "contribution" should have squash auto-merge enabled
    And only one merge action should occur for branch "contribution"
    And the branch check should succeed

  Scenario: Read all history before deciding that the label authorizes a merge
    Given a pull request from "a fork" for branch "contribution" is "ready to merge"
    And branch "contribution" has the "automerge" label
    And GitHub disabled auto-merge for branch "contribution" "after the maintainer disables it"
    And the automation events span multiple pages
    When the worker checks auto-merge branches
    Then no pull request mutations should be requested
    And the branch check should succeed

  Scenario Outline: Reject incomplete or untrusted automation history
    Given a pull request from "a fork" for branch "contribution" is "ready to merge"
    And branch "contribution" has the "automerge" label
    And GitHub disabled auto-merge for branch "contribution" "and the label is reapplied"
    And the automation events span multiple pages
    And the labeled operation fails with "<failure>"
    When the worker checks auto-merge branches
    Then the branch check should fail without affecting worker health
    And no pull request mutations should be requested

    Examples:
      | failure                   |
      | history HTTP              |
      | history page              |
      | history GraphQL           |
      | history rate limit        |
      | malformed history         |
      | mismatched history        |
      | incomplete history        |
      | missing history events    |
      | missing history timestamp |
      | missing history label     |
      | unknown history event     |
      | invalid history cursor    |

  Scenario: Require evidence that the label was applied
    Given a pull request from "a fork" for branch "contribution" is "ready to merge"
    And branch "contribution" has the "automerge" label
    And the labeled operation fails with "empty history"
    When the worker checks auto-merge branches
    Then no pull request mutations should be requested
    And the branch check should succeed

  Scenario: Finish pull request pagination before labeled mutations
    Given a pull request from "a fork" for branch "first" is "waiting for checks"
    And branch "first" has the "automerge" label
    And a pull request from "the repository" for branch "second" is "ready to merge"
    And branch "second" has the "automerge" label
    And the pull requests span multiple pages
    When the worker checks auto-merge branches
    Then branch "first" should have squash auto-merge enabled
    And branch "second" should be squash-merged
    And the branch check should succeed

  Scenario: Continue after another labeled pull request fails
    Given a pull request from "a fork" for branch "blocked" is "waiting for checks"
    And branch "blocked" has the "automerge" label
    And GitHub rejects automation on branch "blocked"
    And a pull request from "a fork" for branch "ready" is "ready to merge"
    And branch "ready" has the "automerge" label
    When the worker checks auto-merge branches
    Then the branch check should fail
    And branch "ready" should be squash-merged

  Scenario: Preserve the automatic Dependabot policy on a labeled pull request
    Given the Dependabot pull request for branch "dependencies" is "waiting for checks"
    And branch "dependencies" has the "automerge" label
    And the labeled operation fails with "history GraphQL"
    When the worker checks auto-merge branches
    Then branch "dependencies" should have squash auto-merge enabled
    And only one merge action should occur for branch "dependencies"
    And the branch check should succeed

  Scenario Outline: Refuse a fork contribution with incomplete pull request metadata
    Given a pull request from "a fork" for branch "contribution" is "waiting for checks"
    And branch "contribution" has the "automerge" label
    And the labeled operation fails with "<metadata>"
    When the worker checks auto-merge branches
    Then the branch check should fail without affecting worker health
    And no pull request mutations should be requested

    Examples:
      | metadata                |
      | missing head repository |
      | missing node ID         |

  Scenario Outline: Hold labeled pull requests from either source
    Given a pull request from "<source>" for branch "contribution" is "<condition>"
    And branch "contribution" has the "automerge" label
    And branch "contribution" has the "hold" label
    When the worker checks auto-merge branches
    Then no pull request mutations should be requested
    And the branch check should succeed

    Examples:
      | source         | condition          |
      | the repository | waiting for checks |
      | the repository | ready to merge     |
      | a fork         | waiting for checks |
      | a fork         | ready to merge     |

  Scenario Outline: Observe a hold added before the final labeled action
    Given a pull request from "a fork" for branch "contribution" is "<condition>"
    And branch "contribution" has the "automerge" label
    And the hold label is added before automation
    When the worker checks auto-merge branches
    Then no pull request mutations should be requested
    And branch "contribution" should retain the "hold" label
    And the branch check should succeed

    Examples:
      | condition          |
      | waiting for checks |
      | ready to merge     |

  Scenario: Removing a hold does not renew revoked fork authorization
    Given a pull request from "a fork" for branch "contribution" is "waiting for checks"
    And branch "contribution" has the "automerge" label
    And GitHub disabled auto-merge for branch "contribution" "after the maintainer disables it"
    And branch "contribution" has the "hold" label
    And the worker has repeatedly respected the hold on branch "contribution"
    When the hold label is removed from branch "contribution"
    Then branch "contribution" should receive no worker mutations
    And auto-merge should remain disabled for branch "contribution"
    And no pull request mutations should be requested

  Scenario: Removing a hold does not renew a removed positive label
    Given a pull request from "a fork" for branch "contribution" is "waiting for checks"
    And branch "contribution" has the "automerge" label
    And branch "contribution" has the "hold" label
    And the automerge label was removed from branch "contribution"
    And the worker has repeatedly respected the hold on branch "contribution"
    When the hold label is removed from branch "contribution"
    Then branch "contribution" should receive no worker mutations
    And branch "contribution" should have no automerge label
    And no pull request mutations should be requested
