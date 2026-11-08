@manual @commands
Feature: Keep review source and results authorized
  Consent, current permissions, pull request eligibility, and revisions remain valid at every review boundary.

  @codereview
  Scenario: Ignore a request without an identified target revision
    Given AI code reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the pull request has no target branch revision
    When a maintainer with "write" access comments "/ab codereview"
    Then no code review source should be shared
    And no completed code review should be posted

  @codereview
  Scenario Outline: Reject eligibility changes before review preparation
    Given AI code reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the code review request loses eligibility through "<change>" before review preparation
    When a maintainer with "write" access comments "/ab codereview"
    Then no code review source should be shared
    And no completed code review should be posted

    Examples:
      | change                |
      | request edited        |
      | request deleted       |
      | closed                |
      | draft                 |
      | retargeted            |
      | base changed          |
      | missing base revision |

  @codereview
  Scenario Outline: Reject eligibility changes during review preparation
    Given AI code reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the code review request loses eligibility through "<change>" during review preparation
    When a maintainer with "write" access comments "/ab codereview"
    Then no completed code review should be posted
    And exactly 1 request should reach the AI service

    Examples:
      | change                |
      | request deleted       |
      | closed                |
      | draft                 |
      | retargeted            |
      | missing base revision |

  @review-results
  Scenario Outline: Recheck consent and revisions before reading related files together
    Given repository source and public Go dependencies are available for review
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the reviewer needs "a batch of related files"
    And review authorization changes through "<change>" during investigation
    When a maintainer with "write" access comments "/ab <command>"
    Then the <kind> review should stop before sharing related source

    @codereview
    Examples:
      | command    | kind     | change                   |
      | codereview | code     | permission revoked       |
      | codereview | code     | head changed             |
      | codereview | code     | request edited           |
      | codereview | code     | request identity changed |

    @secreview
    Examples:
      | command    | kind     | change                   |
      | secreview  | security | base changed             |
      | secreview  | security | request deleted          |

  @review-results
  Scenario Outline: Stop source lookup when consent or the reviewed revision changes
    Given repository source and public Go dependencies are available for review
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And review authorization changes through "<change>" during investigation
    When a maintainer with "write" access comments "/ab <command>"
    Then the <kind> review should stop before sharing related source

    @codereview
    Examples:
      | command    | kind     | change             |
      | codereview | code     | permission revoked |
      | codereview | code     | head changed       |

    @secreview
    Examples:
      | command    | kind     | change             |
      | secreview  | security | base changed       |

  @codereview
  Scenario: Keep review available on hold without repository opt-in
    Given AI code reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the reviewed repository has hold and automatic merging disabled
    When a maintainer with "write" access comments "/ab codereview"
    Then the code review should identify the request and revisions

  @codereview
  Scenario: Require current write access
    Given AI code reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    When a maintainer with "read" access comments "/ab codereview"
    Then the request should be refused without write access
    And no code review source should be shared

  @codereview
  Scenario Outline: Ignore contributions that are not reviewable
    Given AI code reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "<state>"
    When a maintainer with "write" access comments "/ab codereview"
    Then no code review source should be shared

    Examples:
      | state   |
      | a draft |
      | closed  |

  @codereview
  Scenario Outline: Recheck permission and revisions before source submission
    Given AI code reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the request changes through "<change>" before any source is shared
    When a maintainer with "write" access comments "/ab codereview"
    Then no code review source should be shared
    And no completed code review should be posted

    Examples:
      | change                    |
      | loss of write access      |
      | a new pull request commit |

  @codereview
  Scenario Outline: Recheck authority before publishing findings
    Given AI code reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And permission for the review is withdrawn through "<change>" while the answer is prepared
    When a maintainer with "write" access comments "/ab codereview"
    Then no completed code review should be posted
    And exactly 1 request should reach the AI service

    Examples:
      | change               |
      | loss of write access |
      | an edited request    |

  @secreview
  Scenario: Refuse a request without an identified head revision
    Given AI security reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the security review request loses eligibility through "missing head revision" before review preparation
    When a maintainer with "write" access comments "/ab secreview"
    Then no security review source should be shared
    And no completed security review should be posted
