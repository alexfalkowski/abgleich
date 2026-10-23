@github @manual @commands
Feature: Feedback on auto-merge command comments
  Maintainers see when a command is received and whether it was accepted.
  Reactions describe command processing rather than the eventual merge result.

  Background:
    Given GitHub App comment commands are configured

  Scenario: Show receipt while the command is still processing
    Given a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the labeled operation fails with "slow label"
    When a maintainer requests auto-merge while GitHub is still applying the label
    Then receipt should have been visible before the command completed
    And the comment delivery should return HTTP 204
    And the command comment should show "receipt and acceptance" feedback
    And exactly 1 label request should reach GitHub

  Scenario: Acknowledge an accepted request on its original comment
    Given a pull request from "a fork" for branch "contribution" is "waiting for checks"
    When a maintainer with "write" access comments "/ab automerge"
    Then the comment delivery should return HTTP 204
    And the command comment should show "receipt and acceptance" feedback
    And exactly 1 label request should reach GitHub
    And branch "contribution" should have squash auto-merge enabled

  Scenario: Confirm an existing label without requesting it again
    Given a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And branch "contribution" has the "automerge" label
    And the labeled operation fails with "history HTTP"
    When a maintainer with "write" access comments "/ab automerge"
    Then the comment delivery should return HTTP 204
    And the command comment should show "receipt and acceptance" feedback
    And exactly 0 label requests should reach GitHub

  Scenario Outline: Wait for slow outcome feedback after processing the command
    Given a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And GitHub takes longer than one second to acknowledge the command outcome
    When a maintainer with "<permission>" access comments "/ab automerge"
    Then the comment delivery should return HTTP <status>
    And the command comment should show "<feedback>" feedback
    And exactly <labels> label requests should reach GitHub

    Examples:
      | permission | status | feedback               | labels |
      | write      | 204    | receipt and acceptance | 1      |
      | read       | 403    | receipt and rejection  | 0      |

  Scenario: Final feedback cannot prevent completion when GitHub remains slow
    Given a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the command has a short processing deadline
    And GitHub takes longer than one second to acknowledge the command outcome
    When a maintainer with "write" access comments "/ab automerge"
    Then the comment delivery should return HTTP 204
    And the command comment should show "receipt" feedback
    And exactly 1 label request should reach GitHub
    And the reaction failure should be logged

  Scenario: Confirm auto-merge that is already enabled
    Given a pull request from "the repository" for branch "contribution" is "waiting with auto-merge enabled"
    When a maintainer with "write" access comments "/ab automerge"
    Then the comment delivery should return HTTP 204
    And the command comment should show "receipt and acceptance" feedback
    And exactly 0 label requests should reach GitHub

  Scenario: Repeated deliveries retain one receipt and one acceptance reaction
    Given a pull request from "the repository" for branch "contribution" is "waiting for checks"
    When GitHub delivers the same automerge comment twice
    Then the comment delivery should return HTTP 204
    And the command comment should show "receipt and acceptance" feedback
    And exactly 1 label request should reach GitHub

  Scenario: Reject a command from a reader
    Given a pull request from "the repository" for branch "contribution" is "waiting for checks"
    When a maintainer with "read" access comments "/ab automerge"
    Then the comment delivery should return HTTP 403
    And the command comment should show "receipt and rejection" feedback
    And exactly 0 label requests should reach GitHub

  Scenario: An existing label does not authorize a reader's command
    Given a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And branch "contribution" has the "automerge" label
    When a maintainer with "read" access comments "/ab automerge"
    Then the comment delivery should return HTTP 403
    And the command comment should show "receipt and rejection" feedback
    And exactly 0 label requests should reach GitHub

  Scenario: Explain that an ineligible request was received but not accepted
    Given a pull request from "the repository" for branch "contribution" is "a draft"
    When a maintainer with "write" access comments "/ab automerge"
    Then the comment delivery should return HTTP 204
    And the command comment should show "receipt and rejection" feedback
    And exactly 0 label requests should reach GitHub

  Scenario: Reject a command whose authorization was revoked
    Given a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the comment for branch "contribution" was created "before" a "UnlabeledEvent" event
    When GitHub delivers that comment
    Then the comment delivery should return HTTP 204
    And the command comment should show "receipt and rejection" feedback
    And exactly 0 label requests should reach GitHub

  Scenario: Show a failed label request without claiming acceptance
    Given a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the labeled operation fails with "label HTTP"
    When a maintainer with "write" access comments "/ab automerge"
    Then the comment delivery should return HTTP 503
    And the command comment should show "receipt and rejection" feedback
    And exactly 0 label requests should reach GitHub

  Scenario Outline: Do not react to unrelated or untrusted deliveries
    Given a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And its comment delivery has "<problem>"
    When GitHub delivers that comment
    Then the comment delivery should return HTTP <status>
    And the command comment should show "no" feedback
    And exactly 0 label requests should reach GitHub

    Examples:
      | problem              | status |
      | an invalid signature | 401    |
      | another installation | 403    |

  Scenario: Do not react to unrelated comments
    Given a pull request from "the repository" for branch "contribution" is "waiting for checks"
    When a maintainer with "write" access comments "please wait"
    Then the comment delivery should return HTTP 204
    And the command comment should show "no" feedback
    And exactly 0 label requests should reach GitHub

  Scenario Outline: Feedback failures do not prevent the label request
    Given a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the command has a short processing deadline
    And GitHub cannot add comment reactions because of "<problem>"
    When a maintainer with "write" access comments "/ab automerge"
    Then the comment delivery should return HTTP 204
    And the command comment should show "no" feedback
    And exactly 1 label request should reach GitHub
    And the reaction failure should be logged
    And branch "contribution" should have squash auto-merge enabled

    Examples:
      | problem                |
      | an API failure         |
      | a slow response        |
      | a mismatched response  |
      | an incomplete response |
