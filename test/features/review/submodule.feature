@manual @commands @review-results
Feature: Inspect shared tooling pinned by a submodule
  Reviewers can inspect accessible public same-owner submodules at the version used by the pull request.

  Background:
    Given repository source and public Go dependencies are available for review
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"

  Scenario Outline: Discover and read the pinned shared tooling
    Given shared tooling is available from the submodule URL "git@github.com:alexfalkowski/bin.git"
    When a maintainer with "write" access comments "/ab <command>"
    Then the <kind> reviewer should receive the pinned shared tooling
    And the <kind> review decision should be "APPROVED"
    And the <kind> review should have no context limitations

    @codereview
    Examples:
      | command    | kind |
      | codereview | code |

    @secreview
    Examples:
      | command   | kind     |
      | secreview | security |

  @codereview
  Scenario Outline: Disclose shared tooling that cannot be read safely
    Given the shared tooling submodule is "<condition>"
    When a maintainer with "write" access comments "/ab codereview"
    Then the code review should disclose "bin"
    And the code review decision should be "COMMENTED"
    And the incomplete review should have no success marker
    And no shared tooling source should be submitted

    Examples:
      | condition                   |
      | owned by someone else       |
      | outside GitHub              |
      | using HTTPS                 |
      | using an SSH URL            |
      | using a relative URL        |
      | inaccessible to the App     |
      | private                     |
      | without visibility metadata |
      | at a different commit       |
      | at a different path         |

  @codereview
  Scenario: Discard a review when the pinned shared tooling tree cannot be read
    Given the shared tooling submodule is "missing its tree"
    When a maintainer with "write" access comments "/ab codereview"
    Then the code review should fail without publishing findings

  @codereview
  Scenario: Read parent repository source without opening its submodule
    Given shared tooling is available from the submodule URL "git@github.com:alexfalkowski/bin.git"
    And the reviewer needs "a related source file"
    When a maintainer with "write" access comments "/ab codereview"
    Then the reviewer should receive the complete related file
    And the review should use at most 2 GitHub source reads

  @codereview
  Scenario: Read submodule paths through normal file lookup
    Given shared tooling is available from the submodule URL "git@github.com:alexfalkowski/bin.git"
    And the reviewer needs "shared tooling paths"
    When a maintainer with "write" access comments "/ab codereview"
    Then the reviewer should receive shared tooling through normal file lookup
    And the code review decision should be "APPROVED"
    And the code review should have no context limitations

  @codereview
  Scenario: Correct a nonexistent path inside the shared tooling
    Given shared tooling is available from the submodule URL "git@github.com:alexfalkowski/bin.git"
    And the reviewer needs "a corrected submodule path"
    When a maintainer with "write" access comments "/ab codereview"
    Then the code review should finish after correcting the submodule path
    And the code review decision should be "APPROVED"

  @codereview
  Scenario: Recheck consent before opening the submodule
    Given shared tooling is available from the submodule URL "git@github.com:alexfalkowski/bin.git"
    And the reviewer needs "a corrected submodule path"
    And review authorization changes through "permission revoked" during investigation
    When a maintainer with "write" access comments "/ab codereview"
    Then the code review should stop before sharing related source
    And the review should use at most 1 GitHub source reads

  @codereview
  Scenario: Reject a path escaping the submodule
    Given shared tooling is available from the submodule URL "git@github.com:alexfalkowski/bin.git"
    And the reviewer needs "an invalid submodule path"
    When a maintainer with "write" access comments "/ab codereview"
    Then the code review should disclose "invalid submodule path"
    And the code review decision should be "COMMENTED"
    And no shared tooling source should be submitted

  @codereview
  Scenario: Leave nested submodules outside the source scope
    Given shared tooling is available from the submodule URL "git@github.com:alexfalkowski/bin.git"
    And the reviewer needs "a nested submodule"
    When a maintainer with "write" access comments "/ab codereview"
    Then the code review should disclose "nested/private.go"
    And the code review decision should be "COMMENTED"
    And the excluded repository path "nested" should not be fetched
    And no shared tooling source should be submitted

  Scenario Outline: Complete a review after inspecting changed shared tooling
    Given shared tooling is available from the submodule URL "git@github.com:alexfalkowski/bin.git"
    And the contribution updates the shared tooling submodule
    When a maintainer with "write" access comments "/ab <command>"
    Then the <kind> reviewer should receive the pinned shared tooling
    And the <kind> review decision should be "APPROVED"
    And the <kind> review should have no context limitations

    @codereview
    Examples:
      | command    | kind |
      | codereview | code |

    @secreview
    Examples:
      | command   | kind     |
      | secreview | security |

  @codereview
  Scenario Outline: Withhold approval until changed shared tooling source is inspected
    Given shared tooling is available from the submodule URL "git@github.com:alexfalkowski/bin.git"
    And the contribution updates the shared tooling submodule
    And the reviewer needs "<lookup>"
    When a maintainer with "write" access comments "/ab codereview"
    Then the code review should disclose "bin (regular file unavailable)"
    And the code review decision should be "COMMENTED"
    And the incomplete review should have no success marker

    Examples:
      | lookup                |
      | a related source file |
      | a submodule directory |

  @codereview
  Scenario: Preserve a missing patch limitation after inspecting changed shared tooling
    Given shared tooling is available from the submodule URL "git@github.com:alexfalkowski/bin.git"
    And the contribution updates the shared tooling submodule
    And the changed submodule patch is unavailable
    When a maintainer with "write" access comments "/ab codereview"
    Then the code review should disclose "patch for bin"
    And the code review decision should be "COMMENTED"
    And the incomplete review should have no success marker
