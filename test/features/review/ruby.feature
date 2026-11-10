@manual @commands @review-results
Feature: Review Ruby with relevant guidance and pinned dependency source
  Reviews select the repository's language guidance and inspect verified public gems without executing Ruby.

  Background:
    Given repository source and public Go dependencies are available for review
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"

  Scenario Outline: Supply only applicable language and test guidance
    Given the reviewed project uses "<project>"
    When a maintainer with "write" access comments "/ab <command>"
    Then the <kind> reviewer should receive guides for "<guides>"
    And the reviewer should receive only source tools for "<guides>"

    @codereview
    Examples:
      | project     | command    | kind | guides             |
      | Go          | codereview | code | Go                 |
      | Ruby        | codereview | code | Ruby, Cucumber     |
      | Go and Ruby | codereview | code | Go, Ruby, Cucumber |

    @secreview
    Examples:
      | project | command   | kind     | guides         |
      | Ruby    | secreview | security | Ruby, Cucumber |

  Scenario Outline: Discover and read only the requested gem source at its locked version
    Given the reviewed project uses "Ruby"
    And the reviewer needs "Ruby gems"
    When a maintainer with "write" access comments "/ab <command>"
    Then the <kind> reviewer should inspect the locked Ruby gem with "reviewed lockfile checksum" verification
    And the <kind> review decision should be "APPROVED"

    @codereview
    Examples:
      | command    | kind |
      | codereview | code |

    @secreview
    Examples:
      | command   | kind     |
      | secreview | security |

  @codereview
  Scenario: Verify an older lockfile's gem against registry metadata
    Given the reviewed project uses "Ruby"
    And Ruby dependency lookup encounters "registry checksum only"
    When a maintainer with "write" access comments "/ab codereview"
    Then the code reviewer should inspect the locked Ruby gem with "RubyGems registry checksum" verification

  @codereview
  Scenario: Select the explicitly requested locked platform
    Given the reviewed project uses "Ruby"
    And the reviewer needs "Ruby platform gem"
    When a maintainer with "write" access comments "/ab codereview"
    Then the code reviewer should inspect the locked Ruby gem with "reviewed lockfile checksum" verification
    And RubyGems should receive the locked artifact "example-1.2.3-x86_64-linux.gem"

  @codereview
  Scenario Outline: Withhold unverified or unsupported dependency source
    Given the reviewed project uses "Ruby"
    And Ruby dependency lookup encounters "<condition>"
    When a maintainer with "write" access comments "/ab codereview"
    Then the code review should disclose external source note "<limitation>"
    And unverified Ruby source should be withheld after <requests> registry requests
    And the code review decision should be "APPROVED"

    Examples:
      | condition                   | limitation                | requests |
      | private registry            | unsupported registry      | 0        |
      | Git dependency              | repository source         | 0        |
      | duplicate dependency        | ambiguous locked          | 0        |
      | invalid checksum            | invalid Ruby checksum     | 0        |
      | lock checksum mismatch      | gem checksum mismatch     | 2        |
      | registry checksum mismatch  | gem checksum mismatch     | 2        |
      | invalid archive             | invalid gem source path   | 2        |
      | symlink                     | invalid gem source path   | 2        |
      | hardlink                    | invalid gem source path   | 2        |
      | duplicate archive entry     | invalid gem source path   | 2        |
      | persistent rate limit       | RubyGems rate limited     | 1        |
      | rate limit beyond deadline  | RubyGems rate limited     | 1        |
      | rate limit once             | RubyGems rate limited     | 1        |
      | rate limit date             | RubyGems rate limited     | 1        |
      | persistent registry failure | public source unavailable | 2        |

  @codereview
  Scenario Outline: Retry transient registry failures through the HTTP client
    Given the reviewed project uses "Ruby"
    And Ruby dependency lookup encounters "<condition>"
    When a maintainer with "write" access comments "/ab codereview"
    Then Ruby source lookup should recover through the HTTP client
    And the code review decision should be "APPROVED"

    Examples:
      | condition                  |
      | rate limit without hint    |
      | temporary registry failure |

  @codereview
  Scenario: Continue the other review when one registry request is rate limited
    Given the reviewed project uses "Ruby"
    And Ruby dependency lookup encounters "rate limit once"
    When a maintainer with "write" access comments "/ab review"
    Then a rate-limited gem lookup should not block the other review

  @codereview
  Scenario: Correct an absent gem path without limiting the assessment
    Given the reviewed project uses "Ruby"
    And the reviewer needs "Ruby missing path"
    When a maintainer with "write" access comments "/ab codereview"
    Then the code reviewer should inspect the locked Ruby gem with "reviewed lockfile checksum" verification

  @codereview
  Scenario: Refuse a gem absent from the reviewed lockfile
    Given the reviewed project uses "Ruby"
    And the reviewer needs "Ruby undeclared gem"
    When a maintainer with "write" access comments "/ab codereview"
    Then the code review should disclose external source note "dependency or platform is not locked"
    And unverified Ruby source should be withheld after 0 registry requests

  @codereview
  Scenario Outline: Keep oversized or corrupt gems from exhausting the worker
    Given the reviewed project uses "Ruby"
    And Ruby dependency lookup encounters "<condition>"
    When a maintainer with "write" access comments "/ab codereview"
    Then the code review should disclose external source note "<limitation>"
    And unverified Ruby source should be withheld after <requests> registry requests
    And the code review decision should be "APPROVED"
    And the worker should remain healthy

    Examples:
      | condition                     | limitation                       | requests |
      | highly compressed archive     | Ruby gem byte budget exceeded    | 2        |
      | expansion during listing      | Ruby gem byte budget exceeded    | 2        |
      | oversized requested file      | Ruby gem byte budget exceeded    | 2        |
      | excess data after archive end | Ruby gem byte budget exceeded    | 2        |
      | oversized download            | Ruby gem byte budget exceeded    | 2        |
      | download without a size       | Ruby gem byte budget exceeded    | 2        |
      | oversized metadata            | Ruby gem byte budget exceeded    | 1        |
      | corrupt compression trailer   | invalid gem compression checksum | 2        |

  @codereview
  Scenario: Read the selected source without retaining unrelated gem files
    Given the reviewed project uses "Ruby"
    And Ruby dependency lookup encounters "large unrelated file"
    When a maintainer with "write" access comments "/ab codereview"
    Then the code reviewer should inspect the locked Ruby gem with "reviewed lockfile checksum" verification
    And the worker should remain healthy

  @codereview
  Scenario: Keep repeated cached lookups within the same review budget
    Given the reviewed project uses "Ruby"
    And Ruby dependency lookup encounters "repeated archive lookup"
    When a maintainer with "write" access comments "/ab codereview"
    Then the code review should disclose external source note "Ruby gem byte budget exceeded"
    And Ruby archive rescans should share the review byte budget
    And the worker should remain healthy

  @codereview
  Scenario: Cancel an unfinished gem download during worker shutdown
    Given the reviewed project uses "Ruby"
    And Ruby dependency source is still downloading
    When the worker restarts
    Then the interrupted code review should be cancelled without publication
