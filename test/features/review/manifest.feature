@manual @commands @review-results
Feature: Resolve dependencies from reviewed Go manifests
  Reviewed requirements, replacements, checksums, workspaces, and vendoring determine which dependency source is eligible.

  Background:
    Given repository source and public Go dependencies are available for review
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"

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
  Scenario: Constrain model-requested dependency source access
    Given the reviewer needs "an undeclared Go dependency"
    When a maintainer with "write" access comments "/ab codereview"
    Then the code review should disclose external source note "not required by the reviewed manifest"
    And the code review decision should be "APPROVED"
    And the rejected path or dependency should not be fetched

  @codereview
  Scenario: Inspect the version-specific replacement before a wildcard replacement
    Given related source is "selected by a version-specific replacement"
    When a maintainer with "write" access comments "/ab codereview"
    Then the code review should explain the related implementation
    And the reviewer should receive complete files at the reviewed revisions
    And public dependency access should omit credentials

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
