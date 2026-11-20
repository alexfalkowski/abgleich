@manual @commands @testreview
Feature: Review whether tests protect changed behavior
  Maintainers can request an assessment of test quality through the existing review workflow.

  Background:
    Given AI test reviews are configured for this repository

  Scenario: Explain a weakened behavioral assertion
    Given a pull request from "the repository" for branch "contribution" is "waiting for checks"
    When a maintainer with "write" access comments "/ab testreview"
    Then the request should be accepted
    And the test review should identify the request and revisions
    And the test reviewer should receive the weakened assertion
    And the test review decision should be "CHANGES_REQUESTED"
    And the test reviewer should have read-only access
    And source and findings should remain absent from worker logs
    And exactly 1 request should reach the AI service

  Scenario Outline: Select a verdict from the completed assessment
    Given a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the AI test review answer has "<answer>"
    When a maintainer with "write" access comments "/ab testreview"
    Then the test review decision should be "<decision>"

    Examples:
      | answer      | decision  |
      | no findings | APPROVED  |
      | incomplete  | COMMENTED |

  Scenario: Request the only configured review through the aggregate command
    Given a pull request from "the repository" for branch "contribution" is "waiting for checks"
    When a maintainer with "write" access comments "/ab review"
    Then the command comment should show "receipt and acceptance" feedback
    And the test review should identify the request and revisions
    And exactly 1 request should reach the AI service

  Scenario: Reuse an individual result through the aggregate command after restart
    Given a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the pull request has already received a test review
    And the worker has restarted
    When a maintainer with "write" access comments "/ab review"
    Then the original test review should remain without another model call
    And exactly 1 request should reach the AI service

  Scenario: Decline requests without write permission
    Given a pull request from "the repository" for branch "contribution" is "waiting for checks"
    When a maintainer with "read" access comments "/ab testreview"
    Then the request should be refused without write access
    And no test review source should be shared

  Scenario: Decline fork contributions
    Given a pull request from "a fork" for branch "contribution" is "waiting for checks"
    When a maintainer with "write" access comments "/ab testreview"
    Then the command comment should show "receipt and rejection" feedback
    And no test review source should be shared
    And no native review should be submitted

  Scenario Outline: Recheck eligibility before sharing source
    Given a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the test review request loses eligibility through "<change>" before review preparation
    When a maintainer with "write" access comments "/ab testreview"
    Then no test review source should be shared
    And no completed test review should be posted

    Examples:
      | change         |
      | request edited |
      | base changed   |

  Scenario Outline: Discard a result after consent or revisions change
    Given a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the test review request loses eligibility through "<change>" during review preparation
    When a maintainer with "write" access comments "/ab testreview"
    Then no completed test review should be posted
    And exactly 1 request should reach the AI service

    Examples:
      | change             |
      | permission revoked |
      | head changed       |

  Scenario: Request three independent reviews through one comment
    Given a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And all three review commands are enabled
    When a maintainer with "write" access comments "/ab review"
    Then the code review should identify the request and revisions
    And the security review should identify the request and revisions
    And the test review should identify the request and revisions
    And exactly 3 requests should reach the AI service

  Scenario Outline: Preserve every feature's findings regardless of publication order
    Given a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And all three review commands are enabled
    And the pull request has these earlier review assessments:
      | kind     | answer          |
      | <first>  | inline findings |
      | <second> | no findings     |
    And the reviewed contribution has new commits
    And the worker has restarted
    And the AI <kind> review answer has "no findings"
    When a maintainer with "write" access comments "/ab <command>"
    Then the <kind> review decision should be "COMMENTED"
    And its decision should explain that the other review still requests changes

    Examples:
      | first    | second   | kind     | command    |
      | test     | security | code     | codereview |
      | security | test     | code     | codereview |
      | code     | test     | security | secreview  |
      | code     | security | test     | testreview |

  Scenario: Preserve security findings after a low-severity code reassessment
    Given a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And all three review commands are enabled
    And the pull request has these earlier review assessments:
      | kind     | answer          |
      | security | inline findings |
      | code     | inline findings |
    And the reviewed contribution has new commits
    And the pull request has these earlier review assessments:
      | kind | answer                |
      | code | low-severity findings |
    And the AI test review answer has "no findings"
    When a maintainer with "write" access comments "/ab testreview"
    Then the test review decision should be "COMMENTED"
    And its decision should explain that the other review still requests changes

  Scenario: Preserve findings after their review feature is disabled
    Given a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And all three review commands are enabled
    And the pull request has already received a test review
    And the worker restarts with only code reviews enabled
    And the AI code review answer has "no findings"
    When a maintainer with "write" access comments "/ab codereview"
    Then the code review decision should be "COMMENTED"
    And its decision should explain that the other review still requests changes

  Scenario: Preserve findings from a review feature absent from the current worker
    Given a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And a retired review feature has outstanding findings
    And the AI test review answer has "no findings"
    When a maintainer with "write" access comments "/ab testreview"
    Then the test review decision should be "COMMENTED"
    And its decision should explain that the other review still requests changes

  Scenario: Allow approval after every feature has cleared its own findings
    Given a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And all three review commands are enabled
    And the pull request has these earlier review assessments:
      | kind     | answer          |
      | code     | inline findings |
      | security | inline findings |
      | test     | inline findings |
    And the reviewed contribution has new commits
    And the pull request has these earlier review assessments:
      | kind     | answer      |
      | code     | no findings |
      | security | no findings |
    And the AI test review answer has "no findings"
    When a maintainer with "write" access comments "/ab testreview"
    Then the test review decision should be "APPROVED"

  Scenario: An incomplete reassessment preserves that feature's earlier findings
    Given a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And all three review commands are enabled
    And the pull request has already received a test review
    And the reviewed contribution has new commits
    And the pull request has these earlier review assessments:
      | kind     | answer      |
      | test     | incomplete  |
      | security | no findings |
    And the AI code review answer has "no findings"
    When a maintainer with "write" access comments "/ab codereview"
    Then the code review decision should be "COMMENTED"
    And its decision should explain that the other review still requests changes

  Scenario: Respect dismissal of a feature's findings
    Given a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And all three review commands are enabled
    And the pull request has already received a test review
    And a maintainer dismissed the latest review findings
    And the pull request has these earlier review assessments:
      | kind     | answer      |
      | security | no findings |
    And the AI code review answer has "no findings"
    When a maintainer with "write" access comments "/ab codereview"
    Then the code review decision should be "APPROVED"

  Scenario Outline: Ignore unauthenticated test-review markers
    Given a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And all three review commands are enabled
    And a native test review lookalike was posted by "<author>"
    And the AI code review answer has "no findings"
    When a maintainer with "write" access comments "/ab codereview"
    Then the code review decision should be "APPROVED"

    Examples:
      | author      |
      | a human     |
      | another App |
