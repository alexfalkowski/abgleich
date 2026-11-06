@github @manual @commands @review
Feature: Reduce GitHub source reads across reviews
  Maintainers can request independent reviews without downloading the same source again.
  Reused source remains tied to its repository and verified contents.

  Background:
    Given repository source and public Go dependencies are available for review
    And review source caching is enabled
    And the reviewer needs "a batch of related files"
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"

  Scenario Outline: Reuse source between independent review commands
    Given the pull request has already received a <previous> review
    When a maintainer with "write" access comments "/ab <command>"
    Then the <kind> review should account for all reused source
    And GitHub should have served 1 source tree and 2 source downloads

    @codereview
    Examples:
      | previous | command    | kind     |
      | security | codereview | code     |

    @secreview
    Examples:
      | previous | command    | kind     |
      | code     | secreview  | security |

  @codereview
  Scenario: Reuse unchanged files after new commits
    Given the pull request has already received a code review
    And the reviewed contribution has new commits
    When a maintainer with "write" access comments "/ab codereview"
    Then the code review should account for all reused source
    And GitHub should have served 2 source trees and 2 source downloads

  @codereview
  Scenario: Preserve excluded path evidence when another review reuses the tree
    Given related source is "behind a symbolic link"
    And the reviewer needs "a file beneath an excluded path"
    And the pull request has already received a code review
    When a maintainer with "write" access comments "/ab secreview"
    Then the code review should disclose "helpers/guard.go/private.go"
    And the code review decision should be "COMMENTED"
    And the security review should disclose "helpers/guard.go/private.go"
    And the security review decision should be "COMMENTED"
    And the excluded repository path "helpers/guard.go/private.go" should not be fetched
    And the symbolic link should not be followed
    And GitHub should have served 1 source tree and 1 source download

  @codereview
  Scenario: Fetch changed source after new commits
    Given the pull request has already received a code review
    And the contribution updates its related helper
    When a maintainer with "write" access comments "/ab codereview"
    Then the code review should account for all reused source
    And the reviewer should receive the updated helper
    And GitHub should have served 2 source trees and 3 source downloads

  @codereview
  Scenario: Keep source from separate repositories independent
    Given the pull request has already received a code review
    And another repository has a contribution with identical files
    When the maintainer requests code review in the other repository
    Then the code review should account for all reused source
    And GitHub should have served 2 source trees and 4 source downloads

  @codereview
  Scenario: Exclude a symbolic link even when its earlier contents are cached
    Given the pull request has already received a code review
    And the reviewed contribution has new commits
    And related source is "behind a symbolic link"
    When a maintainer with "write" access comments "/ab codereview"
    Then the code review should exclude the previously read symbolic link
    And GitHub should have served 2 source trees and 2 source downloads

  @secreview
  Scenario: Recover after a failed source download
    Given a previous code review could not retrieve repository source
    When a maintainer with "write" access comments "/ab secreview"
    Then the security review should account for all reused source
    And GitHub should have served 1 source tree and 3 source downloads

  @secreview
  Scenario: Disclose binary exclusions again when source is reused
    Given related source is "binary"
    And the pull request has already received a code review
    When a maintainer with "write" access comments "/ab secreview"
    Then the code review decision should be "APPROVED"
    And the code review should disclose the excluded binary file "main.go"
    And the security review should disclose the excluded binary file "main.go"
    And the security review decision should be "APPROVED"
    And the security review should have no context limitations
    And GitHub should have served 1 source tree and 2 source downloads

  @secreview
  Scenario: Recheck consent even when the requested source is already available
    Given the pull request has already received a code review
    And the security requester loses write access during investigation
    When a maintainer with "write" access comments "/ab secreview"
    Then the security review should stop before sharing reused related source
    And GitHub should have served 1 source tree and 2 source downloads

  @codereview
  Scenario: Keep independent reviews available together
    When maintainers request both reviews together
    Then the code review should account for all reused source
    And the security review should account for all reused source

  @secreview
  Scenario: Keep reviews available without shared caching
    Given review source caching is omitted
    And the pull request has already received a code review
    When a maintainer with "write" access comments "/ab secreview"
    Then the security review should account for all reused source
    And GitHub should have served 2 source trees and 4 source downloads

  @secreview
  Scenario: Read complete source that exceeds the cache value limit
    Given review source exceeds the cache value limit
    And the pull request has already received a code review
    When a maintainer with "write" access comments "/ab secreview"
    Then the security review should account for all reused source
    And GitHub should have served 2 source trees and 4 source downloads

  @secreview
  Scenario: Keep reviews available with another shared cache backend
    Given a different shared cache backend is configured
    And the pull request has already received a code review
    When a maintainer with "write" access comments "/ab secreview"
    Then the security review should account for all reused source
    And GitHub should have served 2 source trees and 4 source downloads
