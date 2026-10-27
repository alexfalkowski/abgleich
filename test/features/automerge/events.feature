@github @manual @reconciliation
Feature: Event-triggered auto-merge checks
  GitHub notifications prompt fresh checks of the affected repository.
  Scheduled sweeps recover missed events and serialize with notification work.

  Background:
    Given GitHub event checks are configured between hourly sweeps
    And the worker has successfully checked the existing repositories

  Scenario Outline: React before the next scheduled sweep
    Given GitHub has a stale auto-merge branch "feature"
    When GitHub delivers a "<event>" notification with action "<action>"
    Then the event delivery should return HTTP 204
    And branch "feature" should be rebased
    And exactly 1 event checks should finish
    And no further scheduled sweep should have completed

    Examples:
      | event        | action              |
      | pull_request | opened              |
      | pull_request | reopened            |
      | pull_request | synchronize         |
      | pull_request | ready_for_review    |
      | pull_request | converted_to_draft  |
      | pull_request | edited              |
      | pull_request | labeled             |
      | pull_request | unlabeled           |
      | pull_request | auto_merge_enabled  |
      | pull_request | auto_merge_disabled |
      | pull_request | closed              |
      | push         |                     |

  Scenario Outline: Enable or merge a Dependabot pull request on notification
    Given the Dependabot pull request for branch "feature" is "<condition>"
    When GitHub delivers a "pull_request" notification with action "opened"
    Then the event delivery should return HTTP 204
    And the worker should report "<action>" for branch "feature"
    And exactly 1 event checks should finish
    And only one mutation request should have been sent

    Examples:
      | condition          | action                          |
      | waiting for checks | enabled pull request auto-merge |
      | ready to merge     | merged pull request             |

  Scenario Outline: Respect Dependabot repository exclusions on notification
    Given Dependabot automation excludes repositories:
      | repository             |
      | alexfalkowski/infraops |
    And the worker restarted with the current configuration
    And the Dependabot pull request for branch "feature" is "<condition>"
    When GitHub delivers a "pull_request" notification with action "opened"
    Then the event delivery should return HTTP 204
    And exactly 1 event checks should finish
    And no event mutations should reach GitHub

    Examples:
      | condition          |
      | waiting for checks |
      | ready to merge     |

  Scenario Outline: Apply current labeled policy to a repository notification
    Given a pull request from "<source>" for branch "feature" is "<condition>"
    And branch "feature" has the "automerge" label
    When GitHub delivers a "pull_request" notification with action "labeled"
    Then the event delivery should return HTTP 204
    And the worker should report "<action>" for branch "feature"
    And exactly 1 event checks should finish
    And only one mutation request should have been sent

    Examples:
      | source         | condition          | action                          |
      | the repository | waiting for checks | enabled pull request auto-merge |
      | the repository | ready to merge     | merged pull request             |
      | a fork         | waiting for checks | enabled pull request auto-merge |
      | a fork         | ready to merge     | merged pull request             |

  Scenario: Check only the affected repository
    Given GitHub has repositories with stale auto-merge branches:
      | repository             | branch  | state   |
      | alexfalkowski/infraops | feature | active  |
      | alexfalkowski/private  | other   | private |
    When GitHub delivers a "push" notification with action ""
    Then branch "feature" should be rebased
    And exactly 1 event checks should finish
    And branch "other" should await its next scheduled sweep

  Scenario Outline: Ignore unrelated notifications
    Given GitHub has a stale auto-merge branch "feature"
    When GitHub delivers a "<event>" notification with action "<action>"
    Then the event delivery should return HTTP 204
    And the notification should cause no repository check

    Examples:
      | event               | action          |
      | ping                |                 |
      | pull_request        | assigned        |
      | pull_request        | a_future_action |
      | pull_request_review | submitted       |
      | check_run           | created         |
      | check_suite         | completed       |
      | status              | pending         |
      | tag push            |                 |

  Scenario Outline: Reject untrusted repository notifications
    Given GitHub has a stale auto-merge branch "feature"
    And the repository notification has "<problem>"
    When GitHub delivers the repository notification
    Then the event delivery should return HTTP <status>
    And the notification should cause no repository check

    Examples:
      | problem                 | status |
      | an invalid signature    | 401    |
      | a missing signature     | 401    |
      | another installation    | 403    |
      | no installation         | 403    |
      | another owner           | 403    |
      | no repository           | 403    |
      | inconsistent repository | 403    |
      | an invalid JSON body    | 400    |

  Scenario Outline: Re-read current policy before acting on a notification
    Given GitHub has a stale auto-merge branch "feature"
    And the event repository is now "<condition>"
    When GitHub delivers a "push" notification with action ""
    Then the event delivery should return HTTP 204
    And exactly 1 event checks should finish
    And no event mutations should reach GitHub

    Examples:
      | condition           |
      | on hold             |
      | opted out           |
      | archived            |
      | disabled            |
      | no longer installed |
      | suspended           |

  Scenario: Resolve repository names without case sensitivity
    Given GitHub has a stale auto-merge branch "feature"
    And the repository notification uses uppercase names
    When GitHub delivers the repository notification
    Then the event delivery should return HTTP 204
    And branch "feature" should be rebased

  Scenario: Finish discovery before acting on a notification
    Given GitHub has repositories with stale auto-merge branches:
      | repository             | branch  | state   |
      | alexfalkowski/infraops | feature | active  |
      | alexfalkowski/private  | other   | private |
    And the repositories span multiple pages
    And a later repository page is unavailable
    When GitHub delivers a "push" notification with action ""
    Then the event check should fail
    And no event mutations should reach GitHub

  Scenario: Finish PR pagination before acting on a notification
    Given GitHub has a stale auto-merge branch "feature"
    And GitHub has a stale auto-merge branch "later"
    And GitHub cannot return a later pull request page
    When GitHub delivers a "push" notification with action ""
    Then the event check should fail
    And no event mutations should reach GitHub

  Scenario: Combine notifications while a scheduled sweep is running
    Given a scheduled sweep is waiting for its repository snapshot
    And GitHub has a stale auto-merge branch "feature"
    When GitHub delivers repeated notifications before the sweep finishes
    Then every repository notification should return HTTP 204
    And branch "feature" should be rebased
    And exactly 1 event checks should finish
    And no extra event checks should follow
    And event and scheduled checks should not overlap

  Scenario: Preserve notifications arriving during an event check
    Given an event check is waiting for its repository snapshot
    When a new branch "later" and repeated notifications arrive during the check
    Then every repository notification should return HTTP 204
    And branch "later" should be rebased
    And exactly 2 event checks should finish
    And no extra event checks should follow
    And event and scheduled checks should not overlap

  Scenario: Accept repository notifications while a scheduled sweep is running
    Given a scheduled sweep is waiting for its repository snapshot
    When notifications for 129 distinct repositories and duplicates arrive before the sweep finishes
    Then every repository notification should return HTTP 204
    And exactly 129 event checks should finish
    And no extra event checks should follow
    And event and scheduled checks should not overlap

  Scenario: Recover a failed event check through the schedule
    Given the event check for branch "feature" failed
    When the worker resumes its regular schedule after GitHub recovers
    Then branch "feature" should be rebased
    And the branch check should succeed

  Scenario: Cancel an event check during shutdown
    Given an event check for branch "feature" is waiting for GitHub
    When the worker restarts
    Then the interrupted event check should be cancelled
    And branch "feature" should be rebased

  Scenario: Continue processing notifications after an event check times out
    Given the event check for branch "feature" timed out
    When GitHub recovers and sends another repository notification
    Then the event delivery should return HTTP 204
    And branch "feature" should be rebased
    And exactly 2 event checks should finish
    And only one mutation request should have been sent
    And the worker should remain healthy

  Scenario: Serialize events with later scheduled sweeps
    Given event checks and sweeps are both enabled on the regular schedule
    And GitHub has a stale auto-merge branch "feature"
    And GitHub responds after the schedule interval
    When GitHub delivers a "push" notification with action ""
    Then branch "feature" should be rebased
    And exactly 1 event checks should finish
    And branch checks should not overlap
