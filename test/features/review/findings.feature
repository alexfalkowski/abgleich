@manual @commands @review-results
Feature: Validate and place review findings
  Complete findings appear beside changed lines when possible and remain in the overview when placement is unavailable.

  Scenario Outline: Place findings on both sides of a contribution
    Given AI <kind> reviews are configured for this repository
    And a pull request from "<source>" for branch "contribution" is "waiting for checks"
    And the reviewer reports "findings on added and removed lines"
    When a maintainer with "write" access comments "/ab <command>"
    Then the <kind> review should contain one overview
    And the inline findings should be:
      | path      | line | side  |
      | main.go   | 2    | RIGHT |
      | worker.go | 2    | LEFT  |
    And each inline finding should separate its title, trigger, impact, and suggested fix
    And no successful review should be posted as an ordinary comment

    @codereview
    Examples:
      | kind | command    | source         |
      | code | codereview | the repository |

    @secreview
    Examples:
      | kind     | command   | source         |
      | security | secreview | the repository |

  Scenario Outline: Preserve findings whose locations cannot be placed
    Given AI <kind> reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the contribution renames a reviewed file
    And the reviewer reports "a finding outside the changed lines"
    When a maintainer with "write" access comments "/ab <command>"
    Then the <kind> review should contain one overview
    And the inline findings should be:
      | path       | line | side  |
      | renamed.go | 2    | RIGHT |
      | worker.go  | 2    | LEFT  |
    And its overview should retain the finding at the unavailable location

    @codereview
    Examples:
      | kind | command    |
      | code | codereview |

    @secreview
    Examples:
      | kind     | command   |
      | security | secreview |

  @codereview
  Scenario: Report malformed model output without submitting a review
    Given AI code reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the reviewer reports "an unreadable answer"
    When a maintainer with "write" access comments "/ab codereview"
    Then the code review should fail without publishing findings
    And its failure comment should explain "unreadable or invalid review result" with recovery advice
    And exactly 1 request should reach the AI service
    And no native review should be submitted

  Scenario Outline: Reject findings without complete explanations
    Given AI <kind> reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the reviewer provides a finding with "<problem>"
    When a maintainer with "write" access comments "/ab <command>"
    Then the <kind> review should fail without publishing findings
    And its failure comment should explain "unreadable or invalid review result" with recovery advice
    And exactly 1 request should reach the AI service
    And no native review should be submitted

    @codereview
    Examples:
      | kind | command    | problem             |
      | code | codereview | missing title       |
      | code | codereview | missing severity    |
      | code | codereview | missing trigger     |
      | code | codereview | missing impact      |
      | code | codereview | missing fix         |
      | code | codereview | blank title         |
      | code | codereview | blank trigger       |
      | code | codereview | blank impact        |
      | code | codereview | blank fix           |
      | code | codereview | an invalid severity |

    @secreview
    Examples:
      | kind     | command   | problem             |
      | security | secreview | missing title       |
      | security | secreview | missing severity    |
      | security | secreview | missing trigger     |
      | security | secreview | missing impact      |
      | security | secreview | missing fix         |
      | security | secreview | blank title         |
      | security | secreview | blank trigger       |
      | security | secreview | blank impact        |
      | security | secreview | blank fix           |
      | security | secreview | an invalid severity |

  Scenario Outline: Reject a missing or invalid completion indicator
    Given AI <kind> reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the AI <kind> review answer has "<answer>"
    When a maintainer with "write" access comments "/ab <command>"
    Then the <kind> review should fail without publishing findings
    And its failure comment should explain "unreadable or invalid review result" with recovery advice
    And no native review should be submitted

    @codereview
    Examples:
      | kind | command    | answer             |
      | code | codereview | missing completion |
      | code | codereview | null completion    |
      | code | codereview | invalid completion |

    @secreview
    Examples:
      | kind     | command   | answer             |
      | security | secreview | missing completion |
      | security | secreview | null completion    |
      | security | secreview | invalid completion |
