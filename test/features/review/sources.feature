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
    And public dependency access should omit credentials

    Examples:
      | command    | kind     |
      | codereview | code     |
      | secreview  | security |

  Scenario: Read a related source file directly
    Given the reviewer needs "a related source file"
    When a maintainer with "write" access comments "/ab codereview"
    Then the reviewer should receive the complete related file

  Scenario: Locate dependency source inside nested package directories
    Given the reviewer needs "a nested dependency package"
    When a maintainer with "write" access comments "/ab codereview"
    Then the reviewer should discover the dependency packages and inspect one

  Scenario Outline: Disclose unavailable context without assuming its behavior
    Given related source is "<problem>"
    When a maintainer with "write" access comments "/ab <command>"
    Then the <kind> review should disclose "<limitation>"
    And unavailable or unverified source should not be shared

    Examples:
      | command    | kind     | problem                                    | limitation            |
      | codereview | code     | unavailable                                | Source unavailable    |
      | codereview | code     | from another revision                      | Source unavailable    |
      | codereview | code     | binary                                     | Source unavailable    |
      | codereview | code     | unverified against the dependency checksum | checksum mismatch     |
      | codereview | code     | missing part of the repository listing     | repository tree       |
      | codereview | code     | behind a symbolic link                     | helpers               |
      | codereview | code     | managed by a Go workspace                  | workspace or vendored |
      | secreview  | security | vendored in the repository                 | workspace or vendored |

  Scenario: Refuse a dependency redirect to an unrelated origin
    Given related source is "redirected to an unrelated origin"
    When a maintainer with "write" access comments "/ab codereview"
    Then the code review should disclose "public dependency source unavailable"
    And unavailable or unverified source should not be shared

  Scenario Outline: Constrain model-requested source access
    Given the reviewer needs "<lookup>"
    When a maintainer with "write" access comments "/ab <command>"
    Then the <kind> review should disclose "<limitation>"
    And the rejected path or dependency should not be fetched

    Examples:
      | command    | kind     | lookup                        | limitation                            |
      | codereview | code     | a file outside the repository | invalid repository path               |
      | codereview | code     | an undeclared Go dependency   | not required by the reviewed manifest |

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
    And deleted source should be read from the reviewed base revision

  Scenario: Permit a richer review after a patch-only result
    Given a patch-only review was published before source lookup was available
    When a maintainer with "write" access comments "/ab codereview"
    Then the code review should explain the related implementation
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
    And public dependency access should omit credentials

  Scenario: Inspect every file in a source directory
    Given related source is "spread across several files"
    When a maintainer with "write" access comments "/ab codereview"
    Then the code review should explain the related implementation
    And the reviewer should receive every file from the directory

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
    And unavailable or unverified source should not be shared

    Examples:
      | command    | kind     | location                |
      | codereview | code     | beside its manifest     |
      | codereview | code     | in its parent directory |

  Scenario: Keep public dependency lookup available outside an unrelated workspace
    Given the reviewed module has a workspace "in an unrelated directory"
    When a maintainer with "write" access comments "/ab codereview"
    Then the code review should explain the related implementation
    And public dependency access should omit credentials
