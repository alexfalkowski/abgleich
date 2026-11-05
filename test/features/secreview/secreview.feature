@github @manual @commands @secreview
Feature: Request a security review with a GitHub verdict
  Maintainers can request a first security review of GitHub pull request patches.
  Reviews report findings and select a verdict without requesting merge automation.

  Background:
    Given AI security reviews are configured for this repository

  Scenario Outline: Review the patches of a contribution
    Given a pull request from "<source>" for branch "contribution" is "waiting for checks"
    When a maintainer with "write" access comments "/ab secreview"
    Then the request should be accepted
    And the security review should identify the request and revisions
    And the security reviewer should receive "the change to main.go"
    And the security reviewer should receive "the updated Lookup implementation"
    And exactly 1 request should reach the AI service
    And the security reviewer should have read-only access
    And source and findings should remain absent from worker logs

    Examples:
      | source         |
      | the repository |
      | a fork         |

  Scenario: Report a clean inspected scope
    Given a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the contribution uses parameterized SQL safely
    And the AI security review answer has "no findings"
    When a maintainer with "write" access comments "/ab secreview"
    Then the security review should describe no findings within the inspected scope

  Scenario Outline: Ignore unsupported command spellings and arguments
    Given a pull request from "the repository" for branch "contribution" is "waiting for checks"
    When a maintainer with "write" access comments "<command>"
    Then no security review source should be shared

    Examples:
      | command              |
      | /ab sec_review       |
      | /ab security review  |
      | /ab secreview please |
      | /ab securityreview   |
      | /ab codereview       |
