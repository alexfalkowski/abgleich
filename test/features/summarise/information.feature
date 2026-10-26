@github @manual @commands @summaries
Feature: Summaries based on complete pull request information
  Maintainers can request summaries of long discussions.
  Unavailable or inconsistent information prevents source from being shared.

  Background:
    Given AI summaries are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"

  Scenario: Find an earlier summary in a long discussion
    Given the pull request has 100 earlier comments
    And an App summary follows those comments
    When a maintainer with "write" access comments "/ab summarise"
    Then the request should be accepted
    And the command comment should show "receipt and acceptance" feedback
    And the existing App summary should remain the only summary
    And the AI service should remain unused

  Scenario: Decline when the rest of a discussion cannot be retrieved
    Given the pull request has 100 earlier comments
    And GitHub cannot retrieve the rest of the discussion
    When a maintainer with "write" access comments "/ab summarise"
    Then the request should be temporarily declined
    And no source should be shared with the AI service
    And no summary should be posted

  Scenario: Reject discussion comments belonging to another pull request
    Given GitHub supplies discussion comments for another pull request
    When a maintainer with "write" access comments "/ab summarise"
    Then the request should be temporarily declined
    And no source should be shared with the AI service
    And no summary should be posted

  Scenario: Explain when the rest of the changes cannot be retrieved
    Given GitHub can retrieve only the first part of the changes
    When a maintainer with "write" access comments "/ab summarise"
    Then the request should be accepted
    And the maintainer should be told that the changes could not be read

  Scenario Outline: Decline when GitHub cannot describe the changes consistently
    Given GitHub provides "<information>"
    When a maintainer with "write" access comments "/ab summarise"
    Then the request should be accepted
    And the maintainer should be told that the changes could not be read

    Examples:
      | information                    |
      | an unidentified changed file   |
      | repeated changes               |
      | an incomplete list of changes  |
      | an unconfirmed list of changes |
