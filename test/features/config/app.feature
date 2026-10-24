@github @manual
Feature: Shared App installation access
  The worker and comment handler use the same App installation.
  Installation tokens renew without restarting the worker.

  Scenario: Discover installation repositories across every page
    Given GitHub has repositories with stale auto-merge branches:
      | repository            | branch  | state  |
      | alexfalkowski/infraops | feature | active |
      | alexfalkowski/another  | another | active |
    And the repositories span multiple pages
    When the worker checks auto-merge branches
    Then branch "feature" should be rebased
    And branch "another" in repository "alexfalkowski/another" should be rebased
    And the branch check should succeed
    And GitHub should receive one installation check per sweep and 0 for comment commands

  Scenario: Check installation access once per comment command
    Given GitHub App comment commands are configured
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    When GitHub delivers the same automerge comment twice
    Then the comment delivery should return HTTP 204
    And exactly 1 label request should reach GitHub
    And the command comment should show "receipt and acceptance" feedback
    And GitHub should receive one installation check per sweep and 2 for comment commands

  Scenario: Poll with App credentials while comment deliveries are disabled
    Given comment deliveries are disabled
    And GitHub has a stale auto-merge branch "feature"
    When the worker checks auto-merge branches
    Then branch "feature" should be rebased
    And the branch check should succeed

  Scenario: Renew installation tokens between scheduled checks
    Given the installation token approaches expiry after one second
    When the worker checks auto-merge branches
    Then the installation token should be renewed
    And the branch check should succeed

  Scenario: Renew an installation token while processing a check
    Given GitHub has repositories with stale auto-merge branches:
      | repository            | branch  | state  |
      | alexfalkowski/infraops | feature | active |
      | alexfalkowski/infraops | second  | active |
      | alexfalkowski/later    | another | active |
    And installation-token renewal becomes "available" during a check
    When the worker checks auto-merge branches
    Then branch "feature" should be rebased
    And branch "second" should be rebased
    And branch "another" in repository "alexfalkowski/later" should be rebased
    And the installation token should be renewed
    And the branch check should succeed

  Scenario: Stop a check at throttled token renewal and recover on a later schedule
    Given GitHub has repositories with stale auto-merge branches:
      | repository            | branch  | state  |
      | alexfalkowski/infraops | feature | active |
      | alexfalkowski/infraops | second  | active |
      | alexfalkowski/later    | another | active |
    And installation-token renewal becomes "rate limited" during a check
    And the first check stopped at throttled installation-token renewal
    When GitHub becomes available again
    Then branch "feature" should be rebased
    And branch "second" should be rebased
    And branch "another" in repository "alexfalkowski/later" should be rebased
    And the branch check should succeed
    And the worker should remain healthy

  Scenario: Continue reporting ordinary token-renewal failures
    Given GitHub has repositories with stale auto-merge branches:
      | repository            | branch  | state  |
      | alexfalkowski/infraops | feature | active |
      | alexfalkowski/infraops | second  | active |
      | alexfalkowski/later    | another | active |
    And installation-token renewal becomes "unavailable" during a check
    When the worker checks auto-merge branches
    Then the first check should report ordinary renewal failures across pull requests and repositories
    And the worker should remain healthy

  Scenario: Renew installation tokens between comment commands without restarting
    Given GitHub App comment commands are configured
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And an earlier command completed before its installation token approached expiry
    When a maintainer with "write" access comments "/ab automerge"
    Then the comment delivery should return HTTP 204
    And the installation token should be renewed
    And exactly 2 label requests should reach GitHub
    And branch "contribution" should have squash auto-merge enabled

  Scenario Outline: Block automation when installation access fails
    Given GitHub App comment commands are configured
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the labeled operation fails with "<failure>"
    When a maintainer with "write" access comments "/ab automerge"
    Then the comment delivery should return HTTP 503
    And exactly 0 label requests should reach GitHub
    And the command comment should show "no" feedback
    And the branch check should fail without affecting worker health

    Examples:
      | failure                 |
      | installation token HTTP |
      | authentication          |
      | mismatched installation |
      | mismatched app          |
      | suspended installation  |
      | empty installation      |
      | null installation       |

  Scenario: Reject an installation whose owner changed after a successful sweep
    Given GitHub App comment commands are configured
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the worker has successfully checked the existing repositories
    And the App installation belongs to account "someone"
    When a maintainer with "write" access comments "/ab automerge"
    Then the comment delivery should return HTTP 503
    And exactly 0 label requests should reach GitHub
    And the command comment should show "no" feedback
    And the branch check should fail without affecting worker health

  Scenario: Reject an incomplete installation repository page
    Given the labeled operation fails with "incomplete discovery"
    When the worker checks auto-merge branches
    Then the branch check should fail without affecting worker health
    And no repositories should have been checked
