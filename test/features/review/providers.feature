@manual @commands @codereview
Feature: Select AI connections independently
  Operators select a named connection, model, and optional effort for each AI feature.

  Scenario: Use the selected connection and effort for every assessment
    Given AI summaries and reviews use separate provider connections
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    When a maintainer requests all AI assessments
    Then every assessment should use its selected connection, model, and effort

  Scenario Outline: Refuse incomplete AI selections at startup
    Given AI summaries and reviews use separate provider connections
    And the AI connection selection has "<problem>"
    When the operator starts the worker
    Then the worker should refuse to start

    Examples:
      | problem                |
      | an unknown connection  |
      | a missing connection   |
      | invalid summary effort |
