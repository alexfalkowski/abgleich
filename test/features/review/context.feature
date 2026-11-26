@manual @commands @codereview
Feature: Collect complete changed-file review context
  Reviews receive all available patches and changed-file text at the reviewed head or base revision.

  @review-results
  Scenario: Consider the repository's accepted design during review
    Given AI code reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the contribution proposes a change to the repository's accepted design
    When a maintainer with "write" access comments "/ab codereview"
    Then the reviewer should receive the accepted design as source evidence
    And the published review should record the inspected design files

  @review-results
  Scenario Outline: Publish a limited review when part of the accepted design cannot be read
    Given AI code reviews are configured for this repository
    And the GitHub circuit breaker opens after one failure
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the accepted design file is "<change>" in the contribution
    And GitHub reports "<problem>" for one accepted design file
    And the reviewer needs "the omitted design and a related source file"
    When a maintainer with "write" access comments "/ab codereview"
    Then the code review decision should be "COMMENTED"
    And the code review assessment should be "Review completed with limitations"
    And the reviewer and maintainer should receive the readable design and the omission
    And the source tool should reuse the omitted design

    Examples:
      | change    | problem    |
      | unchanged | truncation |
      | unchanged | a delay    |
      | changed   | truncation |
      | changed   | a delay    |

  @review-results
  Scenario: Withhold approval when the target base's design cannot be verified
    Given AI code reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And GitHub cannot list accepted design at the target base
    And the AI code review answer has "no findings"
    When a maintainer with "write" access comments "/ab codereview"
    Then the code review decision should be "COMMENTED"
    And the review should disclose the missing target-base design

  @review-results
  Scenario: Withhold approval when a binary file replaces accepted design text
    Given AI code reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the contribution replaces an accepted design file with a binary
    And the AI code review answer has "no findings"
    When a maintainer with "write" access comments "/ab codereview"
    Then the code review decision should be "COMMENTED"
    And the code review assessment should be "Review completed with limitations"
    And the binary replacement should be disclosed as unavailable design text

  Scenario Outline: Review every change in a large contribution
    Given AI code reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the contribution contains "<changes>"
    When a maintainer with "write" access comments "/ab codereview"
    Then the code review should identify the request and revisions
    And the code reviewer should receive "<evidence>"

    Examples:
      | changes                        | evidence                        |
      | more than twenty changed files | the final changed file          |
      | large combined changes         | the end of the combined changes |
      | an oversized individual change | the end of the large change     |

  Scenario: Disclose a missing patch
    Given AI code reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the contribution contains "a change without a text diff"
    And the AI code review answer has "no findings"
    When a maintainer with "write" access comments "/ab codereview"
    Then the code review should describe no findings within the inspected scope
    And the code reviewer should receive "the missing patch notice"

  Scenario: Withhold a review when some changes cannot be read
    Given AI code reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And GitHub can retrieve only the first part of the changes
    When a maintainer with "write" access comments "/ab codereview"
    Then code review collection should fail before sharing source

  @review-results
  Scenario: Inspect deleted source at the reviewed base
    Given repository source and public Go dependencies are available for review
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the contribution contains "a deleted source file"
    When a maintainer with "write" access comments "/ab codereview"
    Then the code review should explain the related implementation
    And the reviewer should receive complete files at the reviewed revisions
    And deleted source should be read from the reviewed base revision

  @review-results
  Scenario: Identify an excluded deleted binary at the base revision
    Given AI code reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the contribution has a "deleted" binary file
    And the AI code review answer has "no findings"
    When a maintainer with "write" access comments "/ab codereview"
    Then the code review decision should be "APPROVED"
    And the deleted binary file should be identified at the reviewed base revision
