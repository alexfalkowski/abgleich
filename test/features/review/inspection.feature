@manual @commands @review-results
Feature: Inspect related repository files and directories
  Reviewers can discover and read related source while invalid, absent, and excluded paths remain explicit.

  Background:
    Given repository source and public Go dependencies are available for review
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"

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

  @codereview
  Scenario: Inspect a large directory without a local file cap
    Given related source is "spread across many files"
    And the reviewer needs "a large directory"
    When a maintainer with "write" access comments "/ab codereview"
    Then the reviewer should receive all 105 files from the directory
    And the review should use at most 4 GitHub source reads

  @codereview
  Scenario: Read an empty regular file
    Given the reviewer needs "a related source file"
    And related source is "empty related file"
    When a maintainer with "write" access comments "/ab codereview"
    Then the reviewer should receive an empty regular file

  @codereview
  Scenario: Constrain model-requested repository source access
    Given the reviewer needs "a file outside the repository"
    When a maintainer with "write" access comments "/ab codereview"
    Then the code review should disclose "invalid repository path"
    And the rejected path or dependency should not be fetched

  @codereview
  Scenario: Inspect every file in a source directory
    Given related source is "spread across several files"
    When a maintainer with "write" access comments "/ab codereview"
    Then the code review should explain the related implementation
    And the reviewer should receive complete files at the reviewed revisions
    And the reviewer should receive all 3 files from the directory

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
