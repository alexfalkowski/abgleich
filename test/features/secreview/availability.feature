@github @manual @commands @secreview
Feature: Keep security review recoverable and independent
  Maintainers can retry failed requests and keep using automation while a review is prepared.

  Background:
    Given AI security reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"

  Scenario: Share one review between concurrent maintainer requests
    When two maintainers request security review while a review is being prepared
    Then the security review should identify the request and revisions
    And exactly 1 request should reach the AI service

  Scenario: Cancel unfinished reviews during worker shutdown
    Given the AI service is still preparing the security review
    When the worker restarts
    Then the interrupted security review should be cancelled without publication

  Scenario: Keep branch automation available while a review is being prepared
    Given the AI service is still preparing the security review
    When a maintainer with "write" access comments "/ab status"
    Then the request should be accepted
    And ordinary automation should complete a check while a security review is being prepared
    And the worker should remain healthy

  Scenario: Bound pending work without blocking automation
    Given four maintainers are waiting for security reviews
    When a maintainer with "write" access comments "/ab secreview"
    Then the request should be temporarily declined
    And ordinary automation should continue while those security reviews wait

  Scenario: Request all AI tasks for the same revision
    Given security reviews and summaries are independently enabled
    And the pull request has already been summarised
    And the pull request has already received a code review
    When a maintainer with "write" access comments "/ab secreview"
    Then the security review should identify the request and revisions
    And exactly 3 requests should reach the AI service

  Scenario: Keep security review from suppressing the other AI tasks
    Given security reviews and summaries are independently enabled
    And the pull request has already received a security review
    And the pull request has already received a code review
    When a maintainer with "write" access comments "/ab summarise"
    Then the summary should identify the request and the pull request revision
    And exactly 3 requests should reach the AI service

  Scenario: Refuse a request without an identified head revision
    Given the security review request loses eligibility through "missing head revision" before review preparation
    When a maintainer with "write" access comments "/ab secreview"
    Then no security review source should be shared
    And no completed security review should be posted

  Scenario: Withhold source when the full discussion is unavailable
    Given the pull request has 105 earlier comments
    And GitHub cannot retrieve the complete security review discussion
    When a maintainer with "write" access comments "/ab secreview"
    Then the request should be temporarily declined
    And no security review source should be shared
    And no completed security review should be posted

  Scenario: Share one security review between waiting maintainers
    Given two security review requests are waiting for the same revision
    When Abgleich processes the waiting security review
    Then the security review should identify the request and revisions
    And exactly 1 request should reach the AI service

  Scenario: Allow a fresh human request after provider failure
    Given a security review previously failed at the provider
    When a maintainer with "write" access comments "/ab secreview"
    Then the security review should identify the request and revisions
    And exactly 2 requests should reach the AI service

  Scenario: Review a changed target revision
    Given the pull request has already received a security review
    And the target branch has advanced since the security review
    When a maintainer with "write" access comments "/ab secreview"
    Then both revisions should receive their own security review
