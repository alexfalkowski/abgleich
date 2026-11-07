@github @manual @commands @codereview
Feature: Request a code review with a GitHub verdict
  Maintainers can request a first review of GitHub pull request patches.
  Reviews report findings and select a verdict without requesting merge automation.

  Background:
    Given AI code reviews are configured for this repository

  Scenario: Review the patches of a contribution
    Given a pull request from "the repository" for branch "contribution" is "waiting for checks"
    When a maintainer with "write" access comments "/ab codereview"
    Then the request should be accepted
    And the code review should identify the request and revisions
    And the code reviewer should receive "the change to main.go"
    And the code reviewer should receive "the updated Worker implementation"
    And exactly 1 request should reach the AI service
    And source and findings should remain absent from worker logs

  Scenario Outline: Decline fork contributions without generating a code review
    Given a pull request from "a fork" for branch "contribution" is "<condition>"
    When a maintainer with "write" access comments "/ab codereview"
    Then the comment delivery should return HTTP 204
    And the command comment should show "receipt and rejection" feedback
    And no code review source should be shared
    And no native review should be submitted

    Examples:
      | condition          |
      | waiting for checks |
      | ready to merge     |

  Scenario: Report a clean inspected scope
    Given a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the AI code review answer has "no findings"
    When a maintainer with "write" access comments "/ab codereview"
    Then the code review should describe no findings within the inspected scope

  Scenario: Request the only configured review through the aggregate command
    Given a pull request from "the repository" for branch "contribution" is "waiting for checks"
    When a maintainer with "write" access comments "/ab review"
    Then the command comment should show "receipt and acceptance" feedback
    And the code review should identify the request and revisions
    And exactly 1 request should reach the AI service

  Scenario Outline: Ignore unsupported command spellings and arguments
    Given a pull request from "the repository" for branch "contribution" is "waiting for checks"
    When a maintainer with "write" access comments "<command>"
    Then no code review source should be shared

    Examples:
      | command               |
      | /ab code_review       |
      | /ab review please     |
      | /ab codereview please |
      | /ab secreview         |

