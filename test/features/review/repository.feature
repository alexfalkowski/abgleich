@manual @commands @review-results
Feature: Require complete and verified repository source
  Unavailable, inconsistent, or incomplete repository source stops review processing before it can be trusted.

  Background:
    Given repository source and public Go dependencies are available for review
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"

  Scenario Outline: Stop before inference when repository source cannot be retrieved completely
    Given related source is "<problem>"
    When a maintainer with "write" access comments "/ab <command>"
    Then <kind> review collection should fail before sharing source

    @codereview
    Examples:
      | command    | kind     | problem                                |
      | codereview | code     | unavailable                            |
      | codereview | code     | from another revision                  |
      | codereview | code     | missing part of the repository listing |
      | codereview | code     | truncated                              |
      | codereview | code     | missing binary metadata                |
      | codereview | code     | missing text                           |
      | codereview | code     | missing byte size                      |
      | codereview | code     | not a blob                             |
      | codereview | code     | from another repository                |
      | codereview | code     | malformed response                     |
      | codereview | code     | missing repository                     |
      | codereview | code     | query errors                           |

    @secreview
    Examples:
      | command    | kind     | problem                                |
      | secreview  | security | missing truncation metadata            |
      | secreview  | security | incorrect byte size                    |
      | secreview  | security | digest mismatch                        |
      | secreview  | security | missing response data                  |

  Scenario Outline: Stop investigation instead of publishing with incomplete repository source
    Given the reviewer needs "a batch of related files"
    And related source becomes "<problem>" during investigation
    When a maintainer with "write" access comments "/ab <command>"
    Then the <kind> review should fail without publishing findings
    And exactly 1 request should reach the AI service
    And no unverified helper source should be shared

    @codereview
    Examples:
      | command    | kind     | problem                 |
      | codereview | code     | unavailable             |
      | codereview | code     | missing related file    |
      | codereview | code     | missing text            |

    @secreview
    Examples:
      | command    | kind     | problem                 |
      | secreview  | security | query errors            |
      | secreview  | security | from another repository |
      | secreview  | security | digest mismatch         |

  @codereview
  Scenario: Distinguish missing text from an empty file
    Given the reviewer needs "a related source file"
    And related source is "missing empty file text"
    When a maintainer with "write" access comments "/ab codereview"
    Then the code review should fail without publishing findings
    And exactly 1 request should reach the AI service
