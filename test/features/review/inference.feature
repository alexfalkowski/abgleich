@manual @commands
Feature: Complete bounded review conversations
  Review conversations can investigate related source, fail visibly, and stop during worker shutdown.

  @codereview
  Scenario Outline: Fail visibly when the provider cannot answer
    Given AI code reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the AI service supplies "<answer>"
    When a maintainer with "write" access comments "/ab codereview"
    Then the code review should fail without publishing findings
    And exactly 1 request should reach the AI service

    Examples:
      | answer                  |
      | a refusal               |
      | an unfinished answer    |
      | no answer               |
      | an answer arriving late |

  @codereview
  Scenario: Report provider context limits without truncating or retrying
    Given AI code reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the AI provider rejects the code review context as too large
    When a maintainer with "write" access comments "/ab codereview"
    Then the code review should fail without publishing findings
    And the code reviewer should receive "the change to test.go"
    And exactly 1 request should reach the AI service

  @codereview
  Scenario: Cancel unfinished code reviews during worker shutdown
    Given AI code reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the AI service is still preparing the code review
    When the worker restarts
    Then the interrupted code review should be cancelled without publication

  @codereview
  Scenario Outline: Explain when the reviewer cannot complete an answer
    Given AI code reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the AI code review answer has "<answer>"
    When a maintainer with "write" access comments "/ab codereview"
    Then the code review should fail without publishing findings
    And exactly 1 request should reach the AI service

    Examples:
      | answer                                      |
      | an empty answer                             |
      | a refusal alongside an answer               |
      | a request outside the approved capabilities |

  @review-results @secreview
  Scenario: Finish a review that needs several rounds of investigation
    Given repository source and public Go dependencies are available for review
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the reviewer needs "several rounds of investigation"
    When a maintainer with "write" access comments "/ab secreview"
    Then the security review should use all requested source

  @review-results @codereview
  Scenario: Answer source lookups requested together
    Given repository source and public Go dependencies are available for review
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the reviewer needs "several related files"
    When a maintainer with "write" access comments "/ab codereview"
    Then the code review should use all requested source

  @secreview
  Scenario: Cancel unfinished security reviews during worker shutdown
    Given AI security reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the AI service is still preparing the security review
    When the worker restarts
    Then the interrupted security review should be cancelled without publication

  @secreview
  Scenario: Allow a fresh human request after provider failure
    Given AI security reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And a security review previously failed at the provider
    When a maintainer with "write" access comments "/ab secreview"
    Then the security review should identify the request and revisions
    And exactly 2 requests should reach the AI service
