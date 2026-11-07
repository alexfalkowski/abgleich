@github @manual @commands @review
Feature: Inspect upstream GitHub source during a review
  Maintainers can receive reviews informed by public upstream implementations.
  Upstream reads use no App credentials and preserve consent and revision checks.

  Background:
    Given repository source and public Go dependencies are available for review
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"

  Scenario Outline: Read public upstream source
    Given the reviewer needs public GitHub source
    When a maintainer with "write" access comments "/ab <command>"
    Then the <kind> review should disclose the inspected upstream revision
    And upstream GitHub requests should omit credentials

    @codereview
    Examples:
      | command    | kind |
      | codereview | code |

    @secreview
    Examples:
      | command   | kind     |
      | secreview | security |

  @codereview
  Scenario: Inspect public upstream source while reviewing a private repository
    Given the reviewed repository is private
    And the reviewer needs public GitHub source
    When a maintainer with "write" access comments "/ab codereview"
    Then the code review should disclose the inspected upstream revision
    And upstream GitHub requests should omit credentials

  @codereview
  Scenario: Keep an upstream tag pinned after listing its directory
    Given the reviewer needs a public GitHub directory
    And the upstream tag moves after its directory is listed
    When a maintainer with "write" access comments "/ab codereview"
    Then the code review should disclose the inspected upstream revision
    And the upstream file should use the first resolved revision

  @codereview
  Scenario: Correct an upstream file path without limiting the review
    Given the reviewer needs public GitHub source with a corrected path
    When a maintainer with "write" access comments "/ab codereview"
    Then the code review should disclose the inspected upstream revision
    And the reviewer should receive an upstream not-found result

  @codereview
  Scenario Outline: Reject an invalid upstream source request
    Given the reviewer requests public GitHub source with an invalid "<field>"
    When a maintainer with "write" access comments "/ab codereview"
    Then the upstream source should be unavailable to the code reviewer
    And no upstream GitHub requests should be sent

    Examples:
      | field      |
      | repository |
      | ref        |
      | path       |

  @codereview
  Scenario Outline: Withhold approval when upstream source cannot be verified
    Given the reviewer needs public GitHub source
    And the public GitHub source has "<problem>"
    When a maintainer with "write" access comments "/ab codereview"
    Then the upstream source should be unavailable to the code reviewer
    And upstream GitHub requests should omit credentials

    Examples:
      | problem                        |
      | null repository response       |
      | empty repository response      |
      | whitespace repository response |
      | private repository             |
      | missing visibility             |
      | mismatched repository          |
      | truncated tree                 |
      | inconsistent tree              |
      | inconsistent blob              |
      | binary                         |
      | symlink                        |
      | submodule                      |
      | file beneath symlink           |
      | file beneath submodule         |

  @codereview
  Scenario: Stop upstream lookup when consent changes
    Given the reviewer needs public GitHub source
    And review authorization changes through "permission revoked" during investigation
    When a maintainer with "write" access comments "/ab codereview"
    Then the code review should stop before sharing related source
    And no upstream GitHub requests should be sent
