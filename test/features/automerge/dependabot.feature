@github @manual @merges @dependabot
Feature: Dependabot merge automation
  Repositories opt in through their auto-merge setting.
  Dependabot pull requests are queued for auto-merge or merged when ready.

  Scenario: Enable auto-merge while Dependabot checks are pending
    Given the Dependabot pull request for branch "dependencies" is "waiting for checks"
    When the worker checks auto-merge branches
    Then branch "dependencies" should have squash auto-merge enabled
    And the branch check should succeed
    And the worker should report "enabled pull request auto-merge" for branch "dependencies"
    And only one Dependabot action should occur for branch "dependencies"

  Scenario Outline: Merge a ready Dependabot pull request
    Given the Dependabot pull request for branch "dependencies" is "<condition>"
    When the worker checks auto-merge branches
    Then branch "dependencies" should be squash-merged
    And the branch check should succeed
    And the worker should report "merged pull request" for branch "dependencies"
    And only one Dependabot action should occur for branch "dependencies"

    Examples:
      | condition                                       |
      | ready to merge                                  |
      | ready with auto-merge enabled                   |
      | failing optional checks                         |
      | failing optional checks with auto-merge enabled |

  Scenario: Defer another ready pull request when a merge advances its base
    Given the Dependabot pull request for branch "first" is "ready with auto-merge enabled"
    And the Dependabot pull request for branch "second" is "ready with auto-merge enabled"
    And merging a pull request makes other branches stale
    When the worker checks auto-merge branches
    Then branch "first" should be squash-merged
    And branch "second" should be rebased
    And branch "second" should wait for checks before merging
    And the branch check should succeed
    And no branch checks should have failed

  Scenario: Rebase a stale Dependabot branch after enabling auto-merge
    Given the Dependabot pull request for branch "dependencies" is "behind its base"
    When the worker checks auto-merge branches
    Then branch "dependencies" should have squash auto-merge enabled
    And branch "dependencies" should be rebased
    And the branch check should succeed
    And enabling auto-merge and rebasing should occur in separate checks

  Scenario Outline: Leave a waiting auto-merge pull request to GitHub
    Given the Dependabot pull request for branch "dependencies" is "<condition>"
    When the worker checks auto-merge branches
    Then no pull request mutations should be requested
    And the branch check should succeed

    Examples:
      | condition                                       |
      | waiting with auto-merge enabled                 |
      | of unknown readiness with auto-merge            |
      | conflicting with auto-merge enabled             |
      | failing required checks with auto-merge enabled |
      | missing mergeability with auto-merge            |

  Scenario Outline: Respect Dependabot eligibility
    Given the Dependabot pull request for branch "dependencies" is "<condition>"
    When the worker checks auto-merge branches
    Then no pull request mutations should be requested
    And the branch check should succeed

    Examples:
      | condition   |
      | a draft     |
      | closed      |
      | from a fork |

  Scenario Outline: Identify Dependabot by its author
    Given a ready pull request for branch "dependabot/go_modules/tool" was opened by "<author>"
    When the worker checks auto-merge branches
    Then no pull request mutations should be requested
    And the branch check should succeed

    Examples:
      | author     |
      | author     |
      | dependabot |

  Scenario: Preserve human auto-merge behavior
    Given a ready pull request for branch "feature" was opened by "author" with auto-merge enabled
    When the worker checks auto-merge branches
    Then no pull request mutations should be requested
    And the branch check should succeed

  Scenario Outline: Ignore repositories without auto-merge
    Given the repository has auto-merge disabled
    And the Dependabot pull request for branch "dependencies" is "<condition>"
    And GitHub has a stale auto-merge branch "feature"
    When the worker checks auto-merge branches
    Then no repositories should have been checked
    And no pull request mutations should be requested
    And the branch check should succeed

    Examples:
      | condition          |
      | waiting for checks |
      | ready to merge     |

  Scenario: Refuse a different merge method when squash merging is disabled
    Given the repository has squash merging disabled
    And the Dependabot pull request for branch "dependencies" is "ready to merge"
    When the worker checks auto-merge branches
    Then the branch check should fail without affecting worker health
    And no pull request mutations should be requested

  Scenario Outline: Recheck a Dependabot pull request before acting
    Given the Dependabot pull request for branch "dependencies" is "<condition>"
    And its "<field>" changes before Dependabot automation
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
      | ready to merge     | author          |
      | ready to merge     | state           |
      | ready to merge     | draft           |
      | waiting for checks | head commit     |
      | waiting for checks | author          |

  Scenario: Enable auto-merge if requirements change before merging
    Given the Dependabot pull request for branch "dependencies" is "ready to merge"
    And its "readiness" changes before Dependabot automation
    When the worker checks auto-merge branches
    Then branch "dependencies" should have squash auto-merge enabled
    And only one Dependabot action should occur for branch "dependencies"

  Scenario Outline: Report rejected or unconfirmed Dependabot actions
    Given the Dependabot pull request for branch "dependencies" is "<condition>"
    And the Dependabot operation fails with "<failure>"
    When the worker checks auto-merge branches
    Then the branch check should fail without affecting worker health
    And no Dependabot actions should be accepted

    Examples:
      | condition          | failure             |
      | waiting for checks | mutation            |
      | waiting for checks | partial mutation    |
      | waiting for checks | mismatched mutation |
      | waiting for checks | empty mutation      |
      | waiting for checks | malformed mutation  |
      | ready to merge     | mutation            |
      | ready to merge     | unconfirmed merge   |
      | ready to merge     | empty mutation      |
      | ready to merge     | malformed mutation  |

  Scenario Outline: Never retry a failed mutation through the HTTP transport
    Given the Dependabot pull request for branch "dependencies" is "<condition>"
    And the Dependabot operation fails with "one-shot mutation HTTP"
    When the worker checks auto-merge branches
    Then the branch check should fail without affecting worker health
    And only one mutation request should have been sent
    And no Dependabot actions should be accepted

    Examples:
      | condition          |
      | waiting for checks |
      | ready to merge     |

  Scenario: Reject a concurrent push during an immediate merge
    Given the Dependabot pull request for branch "dependencies" is "ready to merge"
    And the Dependabot operation fails with "concurrent push"
    When the worker checks auto-merge branches
    Then the branch check should fail without affecting worker health
    And no Dependabot actions should be accepted

  Scenario: Finish pull request pagination before enabling auto-merge
    Given the Dependabot pull request for branch "first" is "waiting for checks"
    And the Dependabot pull request for branch "second" is "waiting for checks"
    And the pull requests span multiple pages
    When the worker checks auto-merge branches
    Then branch "first" should have squash auto-merge enabled
    And branch "second" should have squash auto-merge enabled
    And the branch check should succeed

  Scenario: Process a repeated pull request only once per check
    Given the Dependabot pull request for branch "dependencies" is "behind its base"
    And the Dependabot pull request for branch "last" is "waiting for checks"
    And GitHub repeats a pull request across list pages
    When the worker checks auto-merge branches
    Then branch "dependencies" should have squash auto-merge enabled
    And branch "dependencies" should be rebased
    And branch "last" should have squash auto-merge enabled
    And the branch check should succeed
    And enabling auto-merge and rebasing should occur in separate checks

  Scenario: Do not retry a rejected action when a pull request repeats across pages
    Given the Dependabot pull request for branch "blocked" is "waiting for checks"
    And GitHub rejects Dependabot changes on branch "blocked"
    And the Dependabot pull request for branch "last" is "waiting for checks"
    And GitHub repeats a pull request across list pages
    When the worker checks auto-merge branches
    Then the first check should report only one failure for pull request 1
    And branch "last" should have squash auto-merge enabled

  Scenario: Leave all pull requests alone when a later list page fails
    Given the Dependabot pull request for branch "first" is "waiting for checks"
    And the Dependabot pull request for branch "last" is "waiting for checks"
    And GitHub cannot return a later pull request page
    When the worker checks auto-merge branches
    Then the branch check should fail
    And no pull request mutations should be requested

  Scenario: Continue processing other pull requests after a Dependabot failure
    Given the Dependabot pull request for branch "blocked" is "waiting for checks"
    And GitHub rejects Dependabot changes on branch "blocked"
    And the Dependabot pull request for branch "ready" is "ready to merge"
    When the worker checks auto-merge branches
    Then the branch check should fail
    And branch "ready" should be squash-merged

  Scenario Outline: Recover after an accepted Dependabot response is lost
    Given the worker lost confirmation of a Dependabot action on branch "dependencies" that was "<condition>"
    When GitHub becomes available again
    Then only one Dependabot action should occur for branch "dependencies"
    And only one mutation request should have been sent
    And the branch check should succeed

    Examples:
      | condition          |
      | waiting for checks |
      | ready to merge     |

  Scenario Outline: Leave held Dependabot pull requests alone
    Given the Dependabot pull request for branch "dependencies" is "<condition>"
    And branch "dependencies" has the "hold" label
    When the worker checks auto-merge branches
    Then no pull request mutations should be requested
    And the branch check should succeed

    Examples:
      | condition                     |
      | waiting for checks            |
      | ready to merge                |
      | ready with auto-merge enabled  |

  Scenario: Continue processing an unheld Dependabot pull request
    Given the Dependabot pull request for branch "held" is "ready to merge"
    And branch "held" has the "hold" label
    And the Dependabot pull request for branch "unheld" is "waiting for checks"
    When the worker checks auto-merge branches
    Then branch "unheld" should have squash auto-merge enabled
    And branch "held" should receive no worker mutations
    And only one mutation request should have been sent

  Scenario Outline: Ignore labels other than the exact hold label
    Given the Dependabot pull request for branch "dependencies" is "waiting for checks"
    And branch "dependencies" has the "<label>" label
    When the worker checks auto-merge branches
    Then branch "dependencies" should have squash auto-merge enabled
    And the branch check should succeed

    Examples:
      | label |
      | Hold  |
      | bug   |
      | pause |

  Scenario Outline: Observe a hold added before the final Dependabot action
    Given the Dependabot pull request for branch "dependencies" is "<condition>"
    And the hold label is added before automation
    When the worker checks auto-merge branches
    Then no pull request mutations should be requested
    And branch "dependencies" should retain the "hold" label
    And the branch check should succeed

    Examples:
      | condition          |
      | waiting for checks |
      | ready to merge     |

  Scenario: Resume Dependabot enrollment when a hold is removed
    Given the Dependabot pull request for branch "dependencies" is "waiting for checks"
    And branch "dependencies" has the "hold" label
    And the worker has repeatedly respected the hold on branch "dependencies"
    When the hold label is removed from branch "dependencies"
    Then branch "dependencies" should have squash auto-merge enabled
    And the branch check should succeed

  Scenario: Preserve a current hold across a worker restart
    Given the Dependabot pull request for branch "dependencies" is "ready to merge"
    And branch "dependencies" has the "hold" label
    And the worker has repeatedly respected the hold on branch "dependencies"
    When the worker restarts with the hold still present
    Then branch "dependencies" should receive no worker mutations
    And no pull request mutations should be requested
