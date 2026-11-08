@manual @commands
Feature: Process accepted reviews independently
  Manual reviews start promptly, share pending work, and leave ordinary automation available.

  @codereview
  Scenario: Share one code review between concurrent maintainer requests
    Given AI code reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    When two maintainers request code review while a review is being prepared
    Then the code review should identify the request and revisions
    And exactly 1 request should reach the AI service

  @codereview
  Scenario: Publish one review when GitHub delivers a request twice
    Given AI code reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    When GitHub delivers the same code review request twice
    Then the code review should identify the request and revisions
    And exactly 1 request should reach the AI service

  @codereview
  Scenario: Answer status requests while a review is being prepared
    Given AI code reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    When the maintainer asks for status while code review is running
    Then exactly 1 request should reach the AI service
    And the worker should remain healthy

  @codereview
  Scenario: Bound pending code review work without blocking automation
    Given AI code reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And four maintainers are waiting for code reviews
    When a maintainer with "write" access comments "/ab codereview"
    Then the request should be temporarily declined
    And ordinary automation should continue while those code reviews wait

  @codereview
  Scenario: Reissue code review after pending work is lost on restart
    Given AI code reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And a code review request was interrupted by restart
    When a maintainer with "write" access comments "/ab codereview"
    Then the code review should identify the request and revisions
    And exactly 1 request should reach the AI service

  Scenario Outline: Accept and complete a review while a repository sweep waits for GitHub
    Given AI <kind> reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And <kind> reviews use infrequent thread maintenance
    And a repository sweep is waiting for GitHub
    When a maintainer with "write" access comments "/ab <command>"
    Then the comment delivery should return HTTP 204
    And the command comment should show "receipt and acceptance" feedback
    And a <kind> review should arrive within 2 seconds
    And the review should finish before repository scanning resumes
    And the <kind> review should identify the request and revisions
    And exactly 1 request should reach the AI service
    And the worker should remain healthy

    @codereview
    Examples:
      | kind     | command    |
      | code     | codereview |

    @secreview
    Examples:
      | kind     | command    |
      | security | secreview  |

  Scenario Outline: Process a manual request between scheduled maintenance runs
    Given AI <kind> reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And <kind> reviews use infrequent thread maintenance
    When a maintainer with "write" access comments "/ab <command>"
    Then a <kind> review should arrive within 2 seconds
    And the <kind> review should identify the request and revisions
    And manual <kind> review should leave scheduled maintenance pending
    And exactly 1 request should reach the AI service

    @codereview
    Examples:
      | kind     | command    |
      | code     | codereview |

    @secreview
    Examples:
      | kind     | command    |
      | security | secreview  |

  Scenario Outline: Drain requests queued behind active inference
    Given AI <kind> reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And four maintainers are waiting for <kind> reviews
    When the AI service completes the active review
    Then all 4 accepted <kind> reviews should finish promptly
    And exactly 4 requests should reach the AI service
    And the worker should remain healthy

    @codereview
    Examples:
      | kind     |
      | code     |

    @secreview
    Examples:
      | kind     |
      | security |

  @secreview
  Scenario: Share one security review between concurrent maintainer requests
    Given AI security reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    When two maintainers request security review while a review is being prepared
    Then the security review should identify the request and revisions
    And exactly 1 request should reach the AI service

  @secreview
  Scenario: Keep branch automation available while a review is being prepared
    Given AI security reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the AI service is still preparing the security review
    When a maintainer with "write" access comments "/ab status"
    Then the request should be accepted
    And ordinary automation should complete a check while a security review is being prepared
    And the worker should remain healthy

  @secreview
  Scenario: Bound pending security review work without blocking automation
    Given AI security reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And four maintainers are waiting for security reviews
    When a maintainer with "write" access comments "/ab secreview"
    Then the request should be temporarily declined
    And ordinary automation should continue while those security reviews wait

  @secreview
  Scenario: Share one security review between waiting maintainers
    Given AI security reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And two security review requests are waiting for the same revision
    When Abgleich processes the waiting security review
    Then the security review should identify the request and revisions
    And exactly 1 request should reach the AI service
