@manual @commands @review-results @codereview
Feature: Consult the declared Go language reference
  Language references match the reviewed manifest and remain best effort when external documentation is unavailable.

  Background:
    Given repository source and public Go dependencies are available for review
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"

  Scenario: Check language rules using the version declared by the repository
    Given the reviewer needs "the declared Go language rules"
    When a maintainer with "write" access comments "/ab codereview"
    Then the code review should receive the declared Go language reference

  Scenario: Preserve language context when the reviewer requests it again
    Given the reviewer needs "the declared Go language rules again"
    When a maintainer with "write" access comments "/ab codereview"
    Then the code review should receive the declared Go language reference
    And both language lookups should receive the same complete reference

  Scenario: Complete a confident review without external language documentation
    Given external Go reference documentation is unavailable
    When a maintainer with "write" access comments "/ab codereview"
    Then the code review should disclose external source note "Go 1.22 release notes"
    And the code review decision should be "APPROVED"
