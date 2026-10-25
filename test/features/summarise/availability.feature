@github @manual @commands
Feature: Summary service availability
  Operators can bound how long maintainers wait for an answer.
  Repeated failures can temporarily stop new requests to the AI service.

  Background:
    Given AI summaries are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"

  Scenario: Stop waiting when the operator's request limit expires
    Given the operator configures "a 50ms request limit" for the AI service
    And the AI service is slow to answer
    When a maintainer with "write" access comments "/ab summarise"
    Then the request should be accepted
    And the maintainer should receive failure feedback without a summary

  Scenario: Temporarily decline summaries after an AI service failure
    Given the operator configures "a pause after one failed request" for the AI service
    And an earlier request failed while the AI service was busy
    When a maintainer with "write" access comments "/ab summarise"
    Then the request should be accepted
    And the maintainer should be asked to retry without another AI request
