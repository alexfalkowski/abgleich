@manual @commands @review-results
Feature: Inspect verified public dependency source
  Dependency lookups share only verified source without credentials and disclose unavailable external context.

  Background:
    Given repository source and public Go dependencies are available for review
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"

  Scenario Outline: Resolve a finding using related source and the declared dependency
    When a maintainer with "write" access comments "/ab <command>"
    Then the <kind> review should explain the related implementation
    And the reviewer should receive complete files at the reviewed revisions
    And public dependency access should omit credentials
    And the <kind> generation log should identify "read_go_dependency" as its largest source response

    @codereview
    Examples:
      | command    | kind     |
      | codereview | code     |

    @secreview
    Examples:
      | command    | kind     |
      | secreview  | security |

  Scenario Outline: Inspect selected dependency files after discovering their paths
    Given the reviewer needs "a nested dependency package"
    When a maintainer with "write" access comments "/ab <command>"
    Then the <kind> reviewer should discover dependency files and inspect only the selected file
    And public dependency access should omit credentials

    @codereview
    Examples:
      | command    | kind |
      | codereview | code |

    @secreview
    Examples:
      | command   | kind     |
      | secreview | security |

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
  Scenario: Refuse a dependency redirect to an unrelated origin
    Given related source is "redirected to an unrelated origin"
    When a maintainer with "write" access comments "/ab codereview"
    Then the code review should disclose external source note "public dependency source unavailable"
    And the code review decision should be "APPROVED"
    And no unverified dependency source should be shared
    And public dependency access should omit credentials

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

  @secreview
  Scenario: Keep an uncertain review incomplete when external source is unavailable
    Given related source is "public dependency unavailable"
    And the reviewer remains uncertain after inspecting available source
    When a maintainer with "write" access comments "/ab secreview"
    Then the security review should disclose external source note "source unavailable"
    And the security review assessment should be "Review unfinished"
    And the security review decision should be "COMMENTED"
    And the incomplete review should have no success marker
