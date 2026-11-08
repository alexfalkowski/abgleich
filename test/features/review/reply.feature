@manual @commands
Feature: Render review replies safely with provenance
  Replies retain request, revision, and token-usage details while escaping untrusted model text.

  @codereview
  Scenario Outline: Show actual usage without estimating money
    Given AI code reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the AI service reports "<information>" token usage
    When a maintainer with "write" access comments "/ab codereview"
    Then the code review should show "<expected>" in its usage details

    Examples:
      | information    | expected    |
      | ordinary input | test-review |
      | cached input   | 20          |
      | cache writes   | 30          |
      | no usage       | Unavailable |

  @review-results @codereview
  Scenario: Publish an overview when no supported findings remain
    Given AI code reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the reviewer reports "no supported findings"
    When a maintainer with "write" access comments "/ab codereview"
    Then the code review should describe no findings within the inspected scope
    And the native review should have no inline findings

  @review-results
  Scenario Outline: Keep review examples readable and model markup inert in every body
    Given AI <kind> reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the reviewer reports "<answer>"
    When a maintainer with "write" access comments "/ab <command>"
    Then the <kind> review should escape model markup and preserve examples
    And the finding "<placement>" should escape model markup and preserve examples

    @codereview
    Examples:
      | kind | command    | answer                                          | placement   |
      | code | codereview | examples with untrusted markup                  | inline      |
      | code | codereview | examples with untrusted markup outside the diff | in overview |

    @secreview
    Examples:
      | kind     | command   | answer                                          | placement   |
      | security | secreview | examples with untrusted markup                  | inline      |
      | security | secreview | examples with untrusted markup outside the diff | in overview |

  @review-results @secreview
  Scenario: Report unavailable total usage when one source round omits it
    Given repository source and public Go dependencies are available for review
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the AI service omits usage for one round of investigation
    When a maintainer with "write" access comments "/ab secreview"
    Then the security review should show "Unavailable" in its usage details
    And public dependency access should omit credentials

  @codereview
  Scenario: Keep review examples readable and escape untrusted prose
    Given AI code reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the AI code review answer has "hostile text"
    When a maintainer with "write" access comments "/ab codereview"
    Then the code review should escape model markup and preserve examples
