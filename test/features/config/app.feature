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

  Scenario: Renew installation tokens between comment commands without restarting
    Given GitHub App comment commands are configured
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And an earlier command completed before its installation token approached expiry
    When a maintainer with "write" access comments "@abgleich automerge"
    Then the comment delivery should return HTTP 204
    And the installation token should be renewed
    And exactly 2 label requests should reach GitHub
    And branch "contribution" should have squash auto-merge enabled

  Scenario Outline: Block automation when installation access fails
    Given GitHub App comment commands are configured
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the labeled operation fails with "<failure>"
    When a maintainer with "write" access comments "@abgleich automerge"
    Then the comment delivery should return HTTP 503
    And exactly 0 label requests should reach GitHub
    And the branch check should fail without affecting worker health

    Examples:
      | failure                 |
      | installation token HTTP |
      | authentication          |

  Scenario: Reject an installation belonging to another owner
    Given GitHub App comment commands are configured
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the App installation belongs to account "someone"
    When a maintainer with "write" access comments "@abgleich automerge"
    Then the comment delivery should return HTTP 503
    And exactly 0 label requests should reach GitHub
    And the branch check should fail without affecting worker health

  Scenario: Reject an incomplete installation repository page
    Given the labeled operation fails with "incomplete discovery"
    When the worker checks auto-merge branches
    Then the branch check should fail without affecting worker health
    And no repositories should have been checked
