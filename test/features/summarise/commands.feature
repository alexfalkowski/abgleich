@github @manual @commands
Feature: Maintainer-requested summaries
  Maintainers can request one advisory explanation of a pull request.
  Requests respect repository consent and never change pull request automation.

  Background:
    Given AI summaries are configured for this repository

  Scenario Outline: Request a summary of a contribution
    Given a pull request from "<source>" for branch "contribution" is "waiting for checks"
    When a maintainer with "write" access comments "/ab summarise"
    Then the request should be accepted
    And the summary should identify the request and the pull request revision
    And the operator should see that summary generation succeeded
    And the summary should leave pull request automation unchanged

    Examples:
      | source         |
      | the repository |
      | a fork         |

  Scenario: Keep an existing summary after new commits and a restart
    Given a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the pull request has already been summarised
    And new commits arrived before the worker restarted
    When a maintainer with "write" access comments "/ab summarise"
    Then the original summary should be kept without another AI request

  Scenario: Share one summary between maintainers who ask at the same time
    Given a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the AI service is slow to answer
    When two maintainers request a summary before either receives it
    Then the summary should identify the request and the pull request revision
    And exactly 1 request should reach the AI service

  Scenario: Keep a posted summary when GitHub reports a publication error
    Given a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And GitHub reports an error after publishing the summary
    And the pull request has already been summarised
    And the worker has restarted
    When a maintainer with "write" access comments "/ab summarise"
    Then the original summary should be kept without another AI request

  Scenario: Use a summary that arrives while a maintainer is waiting
    Given a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And a maintainer is waiting for a summary
    And an App summary arrives while they wait
    When Abgleich processes the waiting request
    Then the existing summary should satisfy the waiting request

  Scenario: Keep a summary that arrives while the AI service is answering
    Given a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And another App summary arrives while the AI service is answering
    When a maintainer with "write" access comments "/ab summarise"
    Then the existing App summary should remain the only summary
    And exactly 1 request should reach the AI service

  Scenario: Decline a new request while four maintainers are already waiting
    Given four maintainers are waiting for their summaries
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    When a maintainer with "write" access comments "/ab summarise"
    Then the request should be temporarily declined
    And ordinary automation should continue while those requests wait

  Scenario: Respect a repository that has not enabled summaries
    Given a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the repository has not opted in to AI summaries
    When a maintainer with "write" access comments "/ab summarise"
    Then no source should be shared with the AI service

  Scenario: Require write access to request a summary
    Given a pull request from "the repository" for branch "contribution" is "waiting for checks"
    When a reader requests a summary
    Then the request should be refused without write access
    And no source should be shared with the AI service

  Scenario Outline: Ignore contributions that are not ready for a summary
    Given a pull request from "the repository" for branch "contribution" is "<state>"
    When a maintainer with "write" access comments "/ab summarise"
    Then no source should be shared with the AI service

    Examples:
      | state   |
      | a draft |
      | closed  |

  Scenario: Ignore a request that was edited before delivery
    Given a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And its summary request was edited before delivery
    When GitHub delivers that comment
    Then no source should be shared with the AI service

  Scenario Outline: Explain when the AI service cannot supply a usable answer
    Given a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the AI service supplies "<answer>"
    When a maintainer with "write" access comments "/ab summarise"
    Then the maintainer should receive failure feedback without a summary
    And the operator should see that summary generation failed

    Examples:
      | answer                  |
      | a refusal               |
      | an unfinished answer    |
      | no answer               |
      | an answer arriving late |

  Scenario: Request a summary after the AI service recovers
    Given a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the operator configures "three retry attempts" for the AI service
    And an earlier request failed while the AI service was busy
    When a maintainer with "write" access comments "/ab summarise"
    Then the summary should identify the request and the pull request revision
    And exactly 2 requests should reach the AI service
    And the worker should remain healthy

  Scenario: Answer a status request while a summary is being prepared
    Given a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the AI service is slow to answer
    When a maintainer requests status while waiting for a summary
    Then the request should be accepted
    And the maintainer should receive status before the AI service finishes

  Scenario Outline: Ask for a fresh request when the contribution changes
    Given a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the revision changes through "<change>" while the summary is prepared
    When a maintainer with "write" access comments "/ab summarise"
    Then the maintainer should be asked to request a summary of the new revision

    Examples:
      | change                     |
      | a new pull request commit  |
      | a new target branch commit |

  Scenario Outline: Respect withdrawn permission before posting an answer
    Given a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And summary permission is withdrawn through "<change>" while the answer is prepared
    When a maintainer with "write" access comments "/ab summarise"
    Then no summary should be posted after permission is withdrawn

    Examples:
      | change               |
      | loss of write access |
      | an edited request    |
      | repository opt-out   |

  Scenario: Reissue a request that was interrupted by a restart
    Given a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And a summary request was interrupted by a worker restart
    When a maintainer with "write" access comments "/ab summarise"
    Then the summary should identify the request and the pull request revision

  Scenario Outline: Limit the source shared with the AI service
    Given a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the contribution contains "<changes>"
    When a maintainer with "write" access comments "/ab summarise"
    Then only permitted source should be shared with an explanation of "<omission>"

    Examples:
      | changes                        | omission            |
      | more than twenty changed files | omitted files       |
      | large combined changes         | omitted patches     |
      | an oversized individual change | omitted patches     |
      | a change without a text diff   | unavailable changes |

  Scenario Outline: Ignore comments that only look like an App summary
    Given a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And a comment resembling an App summary was posted by "<author>"
    When a maintainer with "write" access comments "/ab summarise"
    Then the summary should identify the request and the pull request revision

    Examples:
      | author      |
      | a human     |
      | another App |

  Scenario: Show model text without activating markup or notifying users
    Given a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the AI service includes markup and mentions in its answer
    When a maintainer with "write" access comments "/ab summarise"
    Then the summary should show the answer without active markup or mentions

  Scenario Outline: Respect changes before sharing any source
    Given a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the request changes through "<change>" before any source is shared
    When a maintainer with "write" access comments "/ab summarise"
    Then no source should be shared with the AI service
    And no completed summary should be posted

    Examples:
      | change                    |
      | repository opt-out        |
      | loss of write access      |
      | a new pull request commit |

  Scenario: Allow summaries when automatic merging is disabled
    Given a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the repository allows summaries but has automatic merging disabled
    When a maintainer with "write" access comments "/ab summarise"
    Then the summary should identify the request and the pull request revision
