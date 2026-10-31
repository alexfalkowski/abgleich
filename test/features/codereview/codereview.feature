@github @manual @commands @codereview
Feature: Request an advisory code review
  Maintainers can request a first review of GitHub pull request patches.
  Reviews remain advisory and do not change pull request automation.

  Background:
    Given AI code reviews are configured for this repository

  Scenario Outline: Review the patches of a contribution
    Given a pull request from "<source>" for branch "contribution" is "waiting for checks"
    When a maintainer with "write" access comments "/ab codereview"
    Then the request should be accepted
    And the code review should identify the request and revisions
    And the review context should include "File: main.go"
    And the review context should include "func Worker() bool { return false }"
    And exactly 1 request should reach the AI service
    And source and findings should remain absent from worker logs

    Examples:
      | source         |
      | the repository |
      | a fork         |

  Scenario: Report a clean inspected scope
    Given a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the AI review answer has "no findings"
    When a maintainer with "write" access comments "/ab codereview"
    Then the review should describe no findings within the inspected scope

  Scenario: Keep advisory review available on hold without repository opt-in
    Given a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the reviewed repository has hold and automatic merging disabled
    When a maintainer with "write" access comments "/ab codereview"
    Then the code review should identify the request and revisions

  Scenario: Keep a completed review after restart
    Given a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the pull request has already received a code review
    And the worker has restarted
    When a maintainer with "write" access comments "/ab codereview"
    Then the original code review should remain without another model call

  Scenario: Recover the published result after an ambiguous GitHub response
    Given a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And GitHub reports an error after publishing the summary
    And the pull request has already received a code review
    And the worker has restarted
    When a maintainer with "write" access comments "/ab codereview"
    Then the original code review should remain without another model call

  Scenario Outline: Ignore unauthenticated lookalikes
    Given a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And a code review lookalike was posted by "<author>"
    When a maintainer with "write" access comments "/ab codereview"
    Then the code review should identify the request and revisions
    And exactly 2 requests should reach the AI service

    Examples:
      | author      |
      | a human     |
      | another App |

  Scenario: Look through an entire long discussion for a completed review
    Given a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the pull request has 105 earlier comments
    And the pull request has already received a code review
    When a maintainer with "write" access comments "/ab codereview"
    Then the original code review should remain without another model call

  Scenario: Require current write access
    Given a pull request from "the repository" for branch "contribution" is "waiting for checks"
    When a maintainer with "read" access comments "/ab codereview"
    Then the request should be refused without write access
    And no code review source should be shared

  Scenario Outline: Ignore contributions that are not reviewable
    Given a pull request from "the repository" for branch "contribution" is "<state>"
    When a maintainer with "write" access comments "/ab codereview"
    Then no code review source should be shared

    Examples:
      | state   |
      | a draft |
      | closed  |

  Scenario Outline: Ignore unsupported command spellings and arguments
    Given a pull request from "the repository" for branch "contribution" is "waiting for checks"
    When a maintainer with "write" access comments "<command>"
    Then no code review source should be shared

    Examples:
      | command               |
      | /ab code_review       |
      | /ab review            |
      | /ab codereview please |
      | /ab secreview         |

  Scenario Outline: Recheck permission and revisions before source submission
    Given a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the request changes through "<change>" before any source is shared
    When a maintainer with "write" access comments "/ab codereview"
    Then no code review source should be shared
    And no completed code review should be posted

    Examples:
      | change                    |
      | loss of write access      |
      | a new pull request commit |

  Scenario Outline: Discard a result after either branch changes
    Given a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the revision changes through "<change>" while the summary is prepared
    When a maintainer with "write" access comments "/ab codereview"
    Then the maintainer should be asked for a code review of the new revision
    And exactly 1 request should reach the AI service

    Examples:
      | change                     |
      | a new pull request commit  |
      | a new target branch commit |

  Scenario Outline: Recheck authority before publishing findings
    Given a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And summary permission is withdrawn through "<change>" while the answer is prepared
    When a maintainer with "write" access comments "/ab codereview"
    Then no completed code review should be posted
    And exactly 1 request should reach the AI service

    Examples:
      | change               |
      | loss of write access |
      | an edited request    |

  Scenario: Escape untrusted prose and redact credentials
    Given a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the AI review answer has "hostile text"
    When a maintainer with "write" access comments "/ab codereview"
    Then the review should escape model markup and redact the provider credential

