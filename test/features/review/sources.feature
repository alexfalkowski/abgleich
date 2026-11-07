@github @manual @commands @review
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

    @codereview
    Examples:
      | command    | kind     |
      | codereview | code     |

    @secreview
    Examples:
      | command    | kind     |
      | secreview  | security |

  @codereview
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

    @codereview
    Examples:
      | command    | kind     |
      | codereview | code     |

    @secreview
    Examples:
      | command    | kind     |
      | secreview  | security |

  @codereview
  Scenario: Preserve useful context when a related path is invalid
    Given the reviewer needs "a batch with an invalid path"
    When a maintainer with "write" access comments "/ab codereview"
    Then the code reviewer should receive the related files together
    And the code review should disclose "invalid repository path"
    And the rejected path or dependency should not be fetched

  @codereview
  Scenario: Reject malformed requests for related files
    Given the reviewer needs "a malformed batch"
    When a maintainer with "write" access comments "/ab codereview"
    Then the code review should fail without publishing findings

  Scenario Outline: Recheck consent and revisions before reading related files together
    Given the reviewer needs "a batch of related files"
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

  @codereview
  Scenario: Locate dependency source inside nested package directories
    Given the reviewer needs "a nested dependency package"
    When a maintainer with "write" access comments "/ab codereview"
    Then the reviewer should discover the dependency packages and inspect one

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

  Scenario Outline: Complete a confident review when external dependency source is unavailable
    Given related source is "<problem>"
    When a maintainer with "write" access comments "/ab <command>"
    Then the <kind> review should disclose external source note "<limitation>"
    And the <kind> review decision should be "APPROVED"
    And no unverified <scope> source should be shared
    And the public dependency proxy should receive <proxy requests> requests without credentials

    @codereview
    Examples:
      | command    | kind     | problem                                    | limitation            | scope      | proxy requests |
      | codereview | code     | unverified against the dependency checksum | checksum mismatch     | dependency | 1              |
      | codereview | code     | managed by a Go workspace                  | workspace or vendored | dependency | 0              |
      | codereview | code     | public dependency unavailable              | source unavailable    | dependency | 1              |

    @secreview
    Examples:
      | command    | kind     | problem                                    | limitation            | scope      | proxy requests |
      | secreview  | security | vendored in the repository                 | workspace or vendored | dependency | 0              |
      | secreview  | security | public dependency rate limited             | source unavailable    | dependency | 1              |

  @codereview
  Scenario: Approve after a related directory lookup excludes an unchanged binary asset
    Given related source is "a binary asset"
    And the reviewer needs "a large directory"
    When a maintainer with "write" access comments "/ab codereview"
    Then the code review decision should be "APPROVED"
    And the code review should disclose the excluded binary file "helpers/favicon.png"
    And the code review should have no context limitations

  @codereview
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

  Scenario Outline: Stop when a reference lookup cannot read its reviewed manifest or checksum
    Given the reviewer needs "<lookup>"
    And related source becomes "<problem>" during investigation
    When a maintainer with "write" access comments "/ab <command>"
    Then the <kind> review should fail without publishing findings
    And exactly 1 request should reach the AI service
    And the public dependency proxy should receive 0 requests without credentials

    @codereview
    Examples:
      | command    | kind     | lookup                         | problem               |
      | codereview | code     | the declared Go language rules | unavailable           |
      | codereview | code     | a nested dependency package    | missing checksum file |

    @secreview
    Examples:
      | command    | kind     | lookup                         | problem               |
      | secreview  | security | a nested dependency package    | unavailable           |

  @codereview
  Scenario: Inspect a large directory without a local file cap
    Given related source is "spread across many files"
    And the reviewer needs "a large directory"
    When a maintainer with "write" access comments "/ab codereview"
    Then the reviewer should receive all 105 files from the directory
    And the review should use at most 2 GitHub source reads

  @codereview
  Scenario: Read an empty regular file
    Given the reviewer needs "a related source file"
    And related source is "empty related file"
    When a maintainer with "write" access comments "/ab codereview"
    Then the reviewer should receive an empty regular file

  @codereview
  Scenario: Distinguish missing text from an empty file
    Given the reviewer needs "a related source file"
    And related source is "missing empty file text"
    When a maintainer with "write" access comments "/ab codereview"
    Then the code review should fail without publishing findings
    And exactly 1 request should reach the AI service

  @codereview
  Scenario: Refuse a dependency redirect to an unrelated origin
    Given related source is "redirected to an unrelated origin"
    When a maintainer with "write" access comments "/ab codereview"
    Then the code review should disclose external source note "public dependency source unavailable"
    And the code review decision should be "APPROVED"
    And no unverified dependency source should be shared
    And public dependency access should omit credentials

  @codereview
  Scenario: Constrain model-requested repository source access
    Given the reviewer needs "a file outside the repository"
    When a maintainer with "write" access comments "/ab codereview"
    Then the code review should disclose "invalid repository path"
    And the rejected path or dependency should not be fetched

  @codereview
  Scenario: Constrain model-requested dependency source access
    Given the reviewer needs "an undeclared Go dependency"
    When a maintainer with "write" access comments "/ab codereview"
    Then the code review should disclose external source note "not required by the reviewed manifest"
    And the code review decision should be "APPROVED"
    And the rejected path or dependency should not be fetched

  Scenario Outline: Stop source lookup when consent or the reviewed revision changes
    Given review authorization changes through "<change>" during investigation
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

  @secreview
  Scenario: Report unavailable total usage when one source round omits it
    Given the AI service omits usage for one round of investigation
    When a maintainer with "write" access comments "/ab secreview"
    Then the security review should show "Unavailable" in its usage details
    And public dependency access should omit credentials

  @codereview
  Scenario: Inspect deleted source at the reviewed base
    Given the contribution contains "a deleted source file"
    When a maintainer with "write" access comments "/ab codereview"
    Then the code review should explain the related implementation
    And the reviewer should receive complete files at the reviewed revisions
    And deleted source should be read from the reviewed base revision

  @codereview
  Scenario: Permit a richer review after a patch-only result
    Given a patch-only review was published before source lookup was available
    When a maintainer with "write" access comments "/ab codereview"
    Then the code review should explain the related implementation
    And the reviewer should receive complete files at the reviewed revisions
    And the source-enabled review should coexist with the old patch-only review

  @secreview
  Scenario: Finish a review that needs several rounds of investigation
    Given the reviewer needs "several rounds of investigation"
    When a maintainer with "write" access comments "/ab secreview"
    Then the security review should use all requested source

  @codereview
  Scenario: Answer source lookups requested together
    Given the reviewer needs "several related files"
    When a maintainer with "write" access comments "/ab codereview"
    Then the code review should use all requested source

  @codereview
  Scenario: Stop source exploration when another matching result appears
    Given the pull request has already received a code review
    And the previous App code review is temporarily absent from the discussion
    And the existing App code review becomes visible while a review is being prepared
    When a maintainer with "write" access comments "/ab codereview"
    Then the existing App code review should remain the only review
    And exactly 5 requests should reach the AI service

  @codereview
  Scenario: Check language rules using the version declared by the repository
    Given the reviewer needs "the declared Go language rules"
    When a maintainer with "write" access comments "/ab codereview"
    Then the code review should receive the declared Go language reference

  @codereview
  Scenario: Preserve language context when the reviewer requests it again
    Given the reviewer needs "the declared Go language rules again"
    When a maintainer with "write" access comments "/ab codereview"
    Then the code review should receive the declared Go language reference
    And both language lookups should receive the same complete reference

  @codereview
  Scenario: Inspect the version-specific replacement before a wildcard replacement
    Given related source is "selected by a version-specific replacement"
    When a maintainer with "write" access comments "/ab codereview"
    Then the code review should explain the related implementation
    And the reviewer should receive complete files at the reviewed revisions
    And public dependency access should omit credentials

  @codereview
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

    @codereview
    Examples:
      | command    |
      | codereview |

    @secreview
    Examples:
      | command    |
      | secreview  |

  Scenario Outline: Inspect workspace dependencies through repository source
    Given the reviewed module has a workspace "<location>"
    When a maintainer with "write" access comments "/ab <command>"
    Then the <kind> review should disclose external source note "workspace or vendored"
    And the <kind> review decision should be "APPROVED"
    And no unverified dependency source should be shared
    And the public dependency proxy should receive 0 requests without credentials

    @codereview
    Examples:
      | command    | kind | location                |
      | codereview | code | beside its manifest     |
      | codereview | code | in its parent directory |

  @codereview
  Scenario: Keep public dependency lookup available outside an unrelated workspace
    Given the reviewed module has a workspace "in an unrelated directory"
    When a maintainer with "write" access comments "/ab codereview"
    Then the code review should explain the related implementation
    And the reviewer should receive complete files at the reviewed revisions
    And public dependency access should omit credentials

  Scenario Outline: Complete a review after correcting a nonexistent repository path
    Given the reviewer needs "<lookup>"
    When a maintainer with "write" access comments "/ab <command>"
    Then the <kind> review should finish after correcting the repository path "<path>"
    And the <kind> review decision should be "APPROVED"

    @codereview
    Examples:
      | command    | kind | lookup                                | path               |
      | codereview | code | a corrected repository file path      | helpers/missing.go |
      | codereview | code | a corrected repository directory path | missing/helpers    |

    @secreview
    Examples:
      | command   | kind     | lookup                                | path               |
      | secreview | security | a corrected repository file path      | helpers/missing.go |
      | secreview | security | a corrected repository directory path | missing/helpers    |

  Scenario Outline: Keep excluded repository paths from permitting approval
    Given related source is "<source>"
    And the reviewer needs "<lookup>"
    When a maintainer with "write" access comments "/ab <command>"
    Then the <kind> review should disclose "<path>"
    And the <kind> review decision should be "COMMENTED"
    And the incomplete review should have no success marker
    And the excluded repository path "<path>" should not be fetched
    And no unverified helper source should be shared

    @codereview
    Examples:
      | command    | kind | source                 | lookup                          | path                        |
      | codereview | code | behind a symbolic link | a file beneath an excluded path | helpers/guard.go/private.go |
      | codereview | code | a submodule            | a related source file           | helpers/guard.go            |

    @secreview
    Examples:
      | command   | kind     | source             | lookup                          | path                        |
      | secreview | security | a submodule        | a file beneath an excluded path | helpers/guard.go/private.go |
      | secreview | security | an empty directory | an empty source directory       | helpers/empty               |

  Scenario Outline: Complete a review after correcting a nonexistent dependency path
    Given the reviewer needs "a corrected dependency path"
    When a maintainer with "write" access comments "/ab <command>"
    Then the <kind> review should finish after correcting the dependency path
    And the <kind> review decision should be "APPROVED"
    And public dependency access should omit credentials

    @codereview
    Examples:
      | command    | kind     |
      | codereview | code     |

    @secreview
    Examples:
      | command    | kind     |
      | secreview  | security |

  @codereview
  Scenario: Complete a confident review without external language documentation
    Given external Go reference documentation is unavailable
    When a maintainer with "write" access comments "/ab codereview"
    Then the code review should disclose external source note "Go 1.22 release notes"
    And the code review decision should be "APPROVED"

  @secreview
  Scenario: Keep an uncertain review incomplete when external source is unavailable
    Given related source is "public dependency unavailable"
    And the reviewer remains uncertain after inspecting available source
    When a maintainer with "write" access comments "/ab secreview"
    Then the security review should disclose external source note "source unavailable"
    And the security review assessment should be "Review unfinished"
    And the security review decision should be "COMMENTED"
    And the incomplete review should have no success marker
