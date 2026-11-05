@github @manual @commands @codereview @secreview @review
Feature: Explain canceled reviews
  Maintainers know when an accepted review can no longer be published.

  Scenario Outline: Explain cancellation of work queued behind another review
    Given AI <kind> reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And another <kind> review is still being prepared
    And a maintainer's <kind> review request is waiting
    When the pull request merges before its queued review starts
    Then the <kind> review cancellation should explain "the pull request was merged"
    And the canceled review should not publish findings
    And exactly 1 request should reach the AI service

    Examples:
      | kind     |
      | code     |
      | security |

  Scenario Outline: Explain cancellation before source submission
    Given AI <kind> reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the <kind> review request loses eligibility through "<state>" before review preparation
    When a maintainer with "write" access comments "/ab <command>"
    Then the <kind> review cancellation should explain "<reason>"
    And no <kind> review source should be shared
    And no completed <kind> review should be posted

    Examples:
      | kind     | command    | state  | reason                          |
      | code     | codereview | merged | the pull request was merged     |
      | code     | codereview | closed | the pull request was closed     |
      | code     | codereview | draft  | the pull request became a draft |
      | security | secreview  | merged | the pull request was merged     |
      | security | secreview  | closed | the pull request was closed     |
      | security | secreview  | draft  | the pull request became a draft |

  Scenario Outline: Explain cancellation after generation starts
    Given AI <kind> reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the <kind> review request loses eligibility through "<state>" during review preparation
    When a maintainer with "write" access comments "/ab <command>"
    Then the <kind> review cancellation should explain "<reason>"
    And no completed <kind> review should be posted
    And exactly 1 request should reach the AI service

    Examples:
      | kind     | command    | state  | reason                          |
      | code     | codereview | merged | the pull request was merged     |
      | code     | codereview | closed | the pull request was closed     |
      | code     | codereview | draft  | the pull request became a draft |
      | security | secreview  | merged | the pull request was merged     |
      | security | secreview  | closed | the pull request was closed     |
      | security | secreview  | draft  | the pull request became a draft |

  Scenario Outline: Explain cancellation during source investigation
    Given repository source and public Go dependencies are available for review
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And review authorization changes through "<state>" during investigation
    When a maintainer with "write" access comments "/ab <command>"
    Then the <kind> review cancellation should explain "<reason>"
    And the <kind> review should stop before sharing related source

    Examples:
      | kind     | command    | state  | reason                      |
      | code     | codereview | merged | the pull request was merged |
      | security | secreview  | closed | the pull request was closed |

  Scenario Outline: Respect withdrawn consent when reporting cancellation
    Given AI <kind> reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the <kind> review loses consent through "<change>" before cancellation
    When a maintainer with "write" access comments "/ab <command>"
    Then no <kind> review source should be shared
    And no completed <kind> review should be posted
    And no review cancellation comment should be posted

    Examples:
      | kind     | command    | change             |
      | code     | codereview | permission revoked |
      | code     | codereview | request edited     |
      | code     | codereview | request deleted    |
      | security | secreview  | permission revoked |
      | security | secreview  | request edited     |
      | security | secreview  | request deleted    |

  Scenario Outline: Keep cancellation feedback after GitHub loses its response
    Given AI <kind> reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the <kind> review request loses eligibility through "closed" before review preparation
    And GitHub reports an error after publishing the reply
    When a maintainer with "write" access comments "/ab <command>"
    Then the <kind> review cancellation should explain "the pull request was closed"
    And no <kind> review source should be shared
    And no completed <kind> review should be posted

    Examples:
      | kind     | command    |
      | code     | codereview |
      | security | secreview  |
