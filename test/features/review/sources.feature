@github @manual @commands @codereview @secreview
Feature: Follow changed behavior into related source
  Maintainers receive reviews grounded in complete changed files and related source.
  Source permissions and the reviewed revisions are respected during investigation.

  Background:
    Given repository source and public Go dependencies are available for review
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"

  Scenario Outline: Resolve a finding using related source and the declared dependency
    When a maintainer with "write" access comments "/ab <command>"
    Then the <kind> review should explain the related implementation
    And the reviewer should receive complete files at the reviewed revisions
    And public dependency access should omit credentials

    Examples:
      | command    | kind     |
      | codereview | code     |
      | secreview  | security |

  Scenario: Read a related source file directly
    Given the reviewer needs "a related source file"
    When a maintainer with "write" access comments "/ab codereview"
    Then the reviewer should receive the complete related file

  Scenario Outline: Inspect related files together within the GitHub request budget
    Given the reviewer needs "a batch of related files"
    And the reviewed repository has hold and automatic merging disabled
    When a maintainer with "write" access comments "/ab <command>"
    Then the <kind> reviewer should receive the related files together
    And the review should use at most 6 pull request checks
    And the review should use at most 2 GitHub source reads

    Examples:
      | command    | kind     |
      | codereview | code     |
      | secreview  | security |

  Scenario: Preserve useful context when a related path is invalid
    Given the reviewer needs "a batch with an invalid path"
    When a maintainer with "write" access comments "/ab codereview"
    Then the code reviewer should receive the related files together
    And the code review should disclose "invalid repository path"
    And the rejected path or dependency should not be fetched

  Scenario: Reject malformed requests for related files
    Given the reviewer needs "a malformed batch"
    When a maintainer with "write" access comments "/ab codereview"
    Then the code review should fail without publishing findings

  Scenario Outline: Recheck consent and revisions before reading related files together
    Given the reviewer needs "a batch of related files"
    And review authorization changes through "<change>" during investigation
    When a maintainer with "write" access comments "/ab <command>"
    Then the <kind> review should stop before sharing related source

    Examples:
      | command    | kind     | change                   |
      | codereview | code     | permission revoked       |
      | codereview | code     | head changed             |
      | secreview  | security | base changed             |
      | codereview | code     | request edited           |
      | secreview  | security | request deleted          |
      | codereview | code     | request identity changed |

  Scenario: Locate dependency source inside nested package directories
    Given the reviewer needs "a nested dependency package"
    When a maintainer with "write" access comments "/ab codereview"
    Then the reviewer should discover the dependency packages and inspect one

  Scenario Outline: Stop before inference when repository source cannot be retrieved completely
    Given related source is "<problem>"
    When a maintainer with "write" access comments "/ab <command>"
    Then <kind> review collection should fail before sharing source

    Examples:
      | command    | kind     | problem                                |
      | codereview | code     | unavailable                            |
      | codereview | code     | from another revision                  |
      | codereview | code     | missing part of the repository listing |
      | codereview | code     | truncated                              |
      | codereview | code     | missing binary metadata                |
      | secreview  | security | missing truncation metadata            |
      | codereview | code     | missing text                           |
      | codereview | code     | missing byte size                      |
      | secreview  | security | incorrect byte size                    |
      | codereview | code     | not a blob                             |
      | secreview  | security | digest mismatch                        |
      | codereview | code     | from another repository                |
      | codereview | code     | malformed response                     |
      | secreview  | security | missing response data                  |
      | codereview | code     | missing repository                     |
      | codereview | code     | query errors                           |

  Scenario Outline: Disclose unsupported source and public dependency limitations
    Given related source is "<problem>"
    When a maintainer with "write" access comments "/ab <command>"
    Then the <kind> review should disclose "<limitation>"
    And no unverified <scope> source should be shared
    And the public dependency proxy should receive <proxy requests> requests without credentials

    Examples:
      | command    | kind     | problem                                    | limitation            | scope      | proxy requests |
      | codereview | code     | binary                                     | Source unavailable    | repository | 0              |
      | codereview | code     | unverified against the dependency checksum | checksum mismatch     | dependency | 1              |
      | codereview | code     | managed by a Go workspace                  | workspace or vendored | dependency | 0              |
      | secreview  | security | vendored in the repository                 | workspace or vendored | dependency | 0              |

  Scenario: Refuse source behind a symbolic link
    Given related source is "behind a symbolic link"
    When a maintainer with "write" access comments "/ab codereview"
    Then the code review should disclose "helpers"
    And no unverified helper source should be shared
    And the symbolic link should not be followed

  Scenario Outline: Stop investigation instead of publishing with incomplete repository source
    Given the reviewer needs "a batch of related files"
    And related source becomes "<problem>" during investigation
    When a maintainer with "write" access comments "/ab <command>"
    Then the <kind> review should fail without publishing findings
    And exactly 1 request should reach the AI service
    And no unverified helper source should be shared

    Examples:
      | command    | kind     | problem                 |
      | codereview | code     | unavailable             |
      | secreview  | security | query errors            |
      | codereview | code     | missing related file    |
      | secreview  | security | from another repository |
      | codereview | code     | missing text            |
      | secreview  | security | digest mismatch         |

  Scenario Outline: Stop when a reference lookup cannot read its reviewed manifest or checksum
    Given the reviewer needs "<lookup>"
    And related source becomes "<problem>" during investigation
    When a maintainer with "write" access comments "/ab <command>"
    Then the <kind> review should fail without publishing findings
    And exactly 1 request should reach the AI service
    And the public dependency proxy should receive 0 requests without credentials

    Examples:
      | command    | kind     | lookup                         | problem               |
      | codereview | code     | the declared Go language rules | unavailable           |
      | secreview  | security | a nested dependency package    | unavailable           |
      | codereview | code     | a nested dependency package    | missing checksum file |

  Scenario: Inspect a large directory without a local file cap
    Given related source is "spread across many files"
    And the reviewer needs "a large directory"
    When a maintainer with "write" access comments "/ab codereview"
    Then the reviewer should receive all 105 files from the directory
    And the review should use at most 2 GitHub source reads

  Scenario: Read an empty regular file
    Given the reviewer needs "a related source file"
    And related source is "empty related file"
    When a maintainer with "write" access comments "/ab codereview"
    Then the reviewer should receive an empty regular file

  Scenario: Distinguish missing text from an empty file
    Given the reviewer needs "a related source file"
    And related source is "missing empty file text"
    When a maintainer with "write" access comments "/ab codereview"
    Then the code review should fail without publishing findings
    And exactly 1 request should reach the AI service

  Scenario: Refuse a dependency redirect to an unrelated origin
    Given related source is "redirected to an unrelated origin"
    When a maintainer with "write" access comments "/ab codereview"
    Then the code review should disclose "public dependency source unavailable"
    And no unverified dependency source should be shared
    And public dependency access should omit credentials

  Scenario Outline: Constrain model-requested source access
    Given the reviewer needs "<lookup>"
    When a maintainer with "write" access comments "/ab <command>"
    Then the <kind> review should disclose "<limitation>"
    And the rejected path or dependency should not be fetched

    Examples:
      | command    | kind | lookup                        | limitation                            |
      | codereview | code | a file outside the repository | invalid repository path               |
      | codereview | code | an undeclared Go dependency   | not required by the reviewed manifest |

  Scenario Outline: Stop source lookup when consent or the reviewed revision changes
    Given review authorization changes through "<change>" during investigation
    When a maintainer with "write" access comments "/ab <command>"
    Then the <kind> review should stop before sharing related source

    Examples:
      | command    | kind     | change             |
      | codereview | code     | permission revoked |
      | codereview | code     | head changed       |
      | secreview  | security | base changed       |

  Scenario: Report unavailable total usage when one source round omits it
    Given the AI service omits usage for one round of investigation
    When a maintainer with "write" access comments "/ab secreview"
    Then the security review should show "Unavailable" in its usage details
    And public dependency access should omit credentials

  Scenario: Inspect deleted source at the reviewed base
    Given the contribution contains "a deleted source file"
    When a maintainer with "write" access comments "/ab codereview"
    Then the code review should explain the related implementation
    And the reviewer should receive complete files at the reviewed revisions
    And deleted source should be read from the reviewed base revision

  Scenario: Permit a richer review after a patch-only result
    Given a patch-only review was published before source lookup was available
    When a maintainer with "write" access comments "/ab codereview"
    Then the code review should explain the related implementation
    And the reviewer should receive complete files at the reviewed revisions
    And the source-enabled review should coexist with the old patch-only review

  Scenario: Finish a review that needs several rounds of investigation
    Given the reviewer needs "several rounds of investigation"
    When a maintainer with "write" access comments "/ab secreview"
    Then the security review should use all requested source

  Scenario: Answer source lookups requested together
    Given the reviewer needs "several related files"
    When a maintainer with "write" access comments "/ab codereview"
    Then the code review should use all requested source

  Scenario: Stop source exploration when another matching result appears
    Given the pull request has already received a code review
    And the previous App code review is temporarily absent from the discussion
    And the existing App code review becomes visible while a review is being prepared
    When a maintainer with "write" access comments "/ab codereview"
    Then the existing App code review should remain the only review
    And exactly 5 requests should reach the AI service

  Scenario: Check language rules using the version declared by the repository
    Given the reviewer needs "the declared Go language rules"
    When a maintainer with "write" access comments "/ab codereview"
    Then the code review should receive the declared Go language reference

  Scenario: Preserve language context when the reviewer requests it again
    Given the reviewer needs "the declared Go language rules again"
    When a maintainer with "write" access comments "/ab codereview"
    Then the code review should receive the declared Go language reference
    And both language lookups should receive the same complete reference

  Scenario: Inspect the version-specific replacement before a wildcard replacement
    Given related source is "selected by a version-specific replacement"
    When a maintainer with "write" access comments "/ab codereview"
    Then the code review should explain the related implementation
    And the reviewer should receive complete files at the reviewed revisions
    And public dependency access should omit credentials

  Scenario: Inspect every file in a source directory
    Given related source is "spread across several files"
    When a maintainer with "write" access comments "/ab codereview"
    Then the code review should explain the related implementation
    And the reviewer should receive complete files at the reviewed revisions
    And the reviewer should receive all 3 files from the directory

  Scenario Outline: Configuring dependency lookup does not enable reviews
    Given dependency lookup is configured but reviews are disabled
    When a maintainer with "write" access comments "/ab <command>"
    Then the command comment should show "receipt and rejection" feedback
    And the AI service should remain unused

    Examples:
      | command    |
      | codereview |
      | secreview  |

  Scenario Outline: Inspect workspace dependencies through repository source
    Given the reviewed module has a workspace "<location>"
    When a maintainer with "write" access comments "/ab <command>"
    Then the <kind> review should disclose "workspace or vendored"
    And no unverified dependency source should be shared
    And the public dependency proxy should receive 0 requests without credentials

    Examples:
      | command    | kind | location                |
      | codereview | code | beside its manifest     |
      | codereview | code | in its parent directory |

  Scenario: Keep public dependency lookup available outside an unrelated workspace
    Given the reviewed module has a workspace "in an unrelated directory"
    When a maintainer with "write" access comments "/ab codereview"
    Then the code review should explain the related implementation
    And the reviewer should receive complete files at the reviewed revisions
    And public dependency access should omit credentials

  Scenario Outline: Complete a review after correcting a nonexistent dependency path
    Given the reviewer needs "a corrected dependency path"
    When a maintainer with "write" access comments "/ab <command>"
    Then the <kind> review should finish after correcting the dependency path
    And the <kind> review decision should be "APPROVED"
    And public dependency access should omit credentials

    Examples:
      | command    | kind     |
      | codereview | code     |
      | secreview  | security |
