@github @manual @commands @summaries
Feature: Provider-reported usage in pull request summaries
  Maintainers can see which model produced a summary and how many tokens it used.
  Unavailable usage does not prevent an advisory summary.

  Background:
    Given AI summaries are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"

  Scenario Outline: Show input and output usage without a dollar estimate
    Given the AI service reports "<usage>" token usage
    When a maintainer with "write" access comments "/ab summarise"
    Then the summary should identify the request and the pull request revision
    And the summary details should include "<tokens>" without a dollar estimate

    Examples:
      | usage          | tokens                                                                 |
      | ordinary input | 100 input (0 cached, 0 cache writes), 40 output including reasoning.   |
      | cached input   | 100 input (20 cached, 0 cache writes), 40 output including reasoning.  |
      | cache writes   | 100 input (20 cached, 30 cache writes), 40 output including reasoning. |

  Scenario: Identify the model returned by the service
    Given the AI service identifies a different model in its response
    When a maintainer with "write" access comments "/ab summarise"
    Then the summary should identify the request and the pull request revision
    And the summary details should include "Model:** other\-model" without a dollar estimate

  Scenario Outline: Publish a summary without inventing missing or inconsistent usage
    Given the AI service reports "<information>" token usage
    When a maintainer with "write" access comments "/ab summarise"
    Then the summary should identify the request and the pull request revision
    And the summary details should include "Tokens:** Unavailable" without a dollar estimate

    Examples:
      | information              |
      | no usage                 |
      | an invalid cached count  |
      | overlapping cache writes |
      | negative usage           |
      | no cache breakdown       |
