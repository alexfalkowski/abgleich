@github @manual @commands
Feature: Comment requests for auto-merge
  Maintainers request existing label-based automation through signed App deliveries.
  Old comments do not renew a request after its authorization is revoked.

  Background:
    Given GitHub App comment commands are configured

  Scenario Outline: Request automation on an eligible contribution
    Given a pull request from "<source>" for branch "contribution" is "waiting for checks"
    When a maintainer with "write" access comments "  /ab automerge  "
    Then the comment delivery should return HTTP 204
    And the command comment should show "receipt and acceptance" feedback
    And branch "contribution" should have squash auto-merge enabled
    And exactly 1 label request should reach GitHub

    Examples:
      | source         |
      | the repository |
      | a fork         |

  Scenario Outline: Enforce the comment author's current permissions
    Given a pull request from "the repository" for branch "contribution" is "waiting for checks"
    When a maintainer with "<permission>" access comments "/ab automerge"
    Then the comment delivery should return HTTP <status>
    And exactly <requests> label requests should reach GitHub

    Examples:
      | permission | status | requests |
      | admin      | 204    | 1        |
      | read       | 403    | 0        |
      | none       | 403    | 0        |

  Scenario Outline: Ignore unrelated comment text
    Given a pull request from "the repository" for branch "contribution" is "waiting for checks"
    When a maintainer with "write" access comments "<comment>"
    Then the comment delivery should return HTTP 204
    And the command comment should show "no" feedback
    And exactly 0 label requests should reach GitHub

    Examples:
      | comment              |
      | /ab automerge please |
      | /ab unknown          |
      | /ab                  |
      | /abautomerge         |
      | /ab  automerge       |
      | /AB automerge        |
      | /ab AUTOMERGE        |
      | please /ab automerge |
      | automerge            |
      | /abgleich automerge  |
      | @ab automerge        |
      | @abgleich automerge  |

  Scenario Outline: Ignore an ineligible pull request
    Given a pull request from "the repository" for branch "contribution" is "<state>"
    When a maintainer with "write" access comments "/ab automerge"
    Then the comment delivery should return HTTP 204
    And exactly 0 label requests should reach GitHub

    Examples:
      | state   |
      | a draft |
      | closed  |

  Scenario: Ignore a pull request without its GraphQL identity
    Given a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the labeled operation fails with "missing node ID"
    When a maintainer with "write" access comments "/ab automerge"
    Then the comment delivery should return HTTP 204
    And exactly 0 label requests should reach GitHub

  Scenario: A duplicate comment delivery leaves the existing label alone
    Given a pull request from "the repository" for branch "contribution" is "waiting for checks"
    When GitHub delivers the same automerge comment twice
    Then the comment delivery should return HTTP 204
    And exactly 1 label request should reach GitHub

  Scenario: Concurrent duplicate deliveries produce one label request
    Given a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the labeled operation fails with "slow label"
    When GitHub delivers the same automerge comment concurrently
    Then the comment delivery should return HTTP 204
    And exactly 1 label request should reach GitHub

  Scenario: An existing label needs no new authorization request
    Given a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And branch "contribution" has the "automerge" label
    And the labeled operation fails with "history HTTP"
    When a maintainer with "write" access comments "/ab automerge"
    Then the comment delivery should return HTTP 204
    And exactly 0 label requests should reach GitHub

  Scenario: Respect repository opt-in
    Given a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the repository has auto-merge disabled
    When a maintainer with "write" access comments "/ab automerge"
    Then the comment delivery should return HTTP 204
    And exactly 0 label requests should reach GitHub

  Scenario: Respect the repository's squash-merge setting
    Given a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the repository has squash merging disabled
    When a maintainer with "write" access comments "/ab automerge"
    Then the comment delivery should return HTTP 204
    And exactly 0 label requests should reach GitHub

  Scenario: A removed label stays removed after a worker restart and redelivery
    Given an earlier comment labeled branch "contribution" before its label was removed
    When GitHub redelivers that comment after the worker restarts
    Then the comment delivery should return HTTP 204
    And branch "contribution" should have no automerge label
    And exactly 1 label request should reach GitHub

  Scenario Outline: Only fresh commands can renew revoked authorization
    Given a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the comment for branch "contribution" was created "<timing>" a "<event>" event
    And the automation events span multiple pages
    When GitHub delivers that comment
    Then the comment delivery should return HTTP 204
    And exactly <requests> label requests should reach GitHub

    Examples:
      | timing           | event                  | requests |
      | before           | UnlabeledEvent         | 0        |
      | at the same time | UnlabeledEvent         | 0        |
      | before           | AutoMergeDisabledEvent | 0        |
      | after            | UnlabeledEvent         | 1        |
      | after            | AutoMergeDisabledEvent | 1        |

  Scenario Outline: Reject untrusted webhook deliveries
    Given a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And its comment delivery has "<problem>"
    When GitHub delivers that comment
    Then the comment delivery should return HTTP <status>
    And exactly 0 label requests should reach GitHub

    Examples:
      | problem              | status |
      | an invalid signature | 401    |
      | a missing signature  | 401    |
      | another installation | 403    |
      | another owner        | 403    |
      | missing PR identity  | 400    |
      | an invalid JSON body | 400    |
      | an oversized body    | 413    |

  Scenario Outline: Acknowledge signed events that cannot request automation
    Given a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And its comment delivery is "<event>"
    When GitHub delivers that comment
    Then the comment delivery should return HTTP 204
    And exactly 0 label requests should reach GitHub

    Examples:
      | event             |
      | an edited comment |
      | an issue comment  |

  Scenario Outline: Re-read current comments before authorizing a label
    Given a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And its original command comment now has "<change>"
    When GitHub delivers that comment
    Then the comment delivery should return HTTP <status>
    And exactly 0 label requests should reach GitHub

    Examples:
      | change               | status |
      | different text       | 204    |
      | an unknown command   | 204    |
      | extra command text   | 204    |
      | an edit timestamp    | 204    |
      | no edit timestamp    | 204    |
      | no author login      | 204    |
      | a bot author         | 204    |
      | another pull request | 503    |
      | another author       | 503    |

  Scenario Outline: Fail without labeling when GitHub reads fail
    Given a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the labeled operation fails with "<failure>"
    When a maintainer with "write" access comments "/ab automerge"
    Then the comment delivery should return HTTP 503
    And exactly 0 label requests should reach GitHub

    Examples:
      | failure         |
      | comment HTTP    |
      | permission HTTP |
      | history HTTP    |
      | comment timeout |

  Scenario Outline: Recheck the PR immediately before labeling
    Given a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And its "<field>" changes before labeled automation
    When a maintainer with "write" access comments "/ab automerge"
    Then the comment delivery should return HTTP <status>
    And exactly 0 label requests should reach GitHub

    Examples:
      | field       | status |
      | head commit | 409    |
      | draft       | 204    |

  Scenario: Recover a lost label response by reading the label on redelivery
    Given a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the labeled operation fails with "lost label response"
    When GitHub redelivers an automerge comment whose first response was lost
    Then the comment delivery should return HTTP 204
    And exactly 1 label request should reach GitHub

  Scenario: A command deadline bounds installation token acquisition
    Given a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And installation token acquisition exceeds the command deadline
    When a maintainer requests auto-merge while token acquisition is slow
    Then the comment delivery should return HTTP 503
    And the command should finish within one second
    And exactly 0 label requests should reach GitHub
    And the branch check should fail without affecting worker health

  Scenario: A command deadline also bounds waiting for a scheduled check's token renewal
    Given a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And installation token acquisition is still running when a command arrives
    When a maintainer requests auto-merge while token acquisition is slow
    Then the comment delivery should return HTTP 503
    And the command should finish within one second
    And exactly 0 label requests should reach GitHub
    And the worker should remain healthy

  Scenario Outline: A comment cannot override a hold
    Given a pull request from "a fork" for branch "contribution" is "<condition>"
    And branch "contribution" has the "hold" label
    When a maintainer with "write" access comments "/ab automerge"
    Then the comment delivery should return HTTP 204
    And the command comment should show "receipt and rejection" feedback
    And exactly 0 label requests should reach GitHub
    And branch "contribution" should retain the "hold" label
    And no pull request mutations should be requested

    Examples:
      | condition                       |
      | waiting for checks              |
      | waiting with auto-merge enabled |

  Scenario: Observe a hold added immediately before the command requests a label
    Given a pull request from "a fork" for branch "contribution" is "waiting for checks"
    And the hold label is added before the command labels the pull request
    When a maintainer with "write" access comments "/ab automerge"
    Then the comment delivery should return HTTP 204
    And the command comment should show "receipt and rejection" feedback
    And exactly 0 label requests should reach GitHub
    And branch "contribution" should retain the "hold" label
    And no pull request mutations should be requested

  Rule: Requested automation status
    Background:
      Given scheduled pull request listing is unavailable

    Scenario Outline: Explain a status snapshot without requesting automation
      Given a pull request from "<source>" for branch "contribution" is "<condition>"
      And branch "contribution" has the "<label>" label
      When a maintainer with "write" access comments "  /ab status  "
      Then the comment delivery should return HTTP 204
      And the status reply should explain "<decision>"
      And the status reply should identify the observed revision and request with auto-merge "<auto-merge>"
      And the command comment should show "receipt and acceptance" feedback
      And the status request should cause no automation mutations

      Examples:
        | source         | condition                       | label     | decision                       | auto-merge |
        | the repository | waiting for checks              | hold      | On hold                        | Disabled   |
        | the repository | a draft                         | automerge | Draft pull request             | Disabled   |
        | the repository | closed                          | automerge | Pull request is closed         | Disabled   |
        | the repository | waiting for checks              | automerge | Would enable squash auto-merge | Disabled   |
        | a fork         | ready to merge                  | automerge | Would squash-merge             | Disabled   |
        | a fork         | failing optional checks         | automerge | Would squash-merge             | Disabled   |
        | the repository | behind with auto-merge enabled  | bug       | Would request a rebase         | Enabled    |
        | the repository | waiting with auto-merge enabled | bug       | Branch is current              | Enabled    |
        | a fork         | behind with auto-merge enabled  | bug       | Fork rebase unsupported        | Enabled    |
        | the repository | waiting for checks              | bug       | No automation requested        | Disabled   |

    Scenario Outline: Explain repository settings that prevent automation
      Given a pull request from "the repository" for branch "contribution" is "waiting for checks"
      And branch "contribution" has the "automerge" label
      And the repository has <setting> disabled
      When a maintainer with "admin" access comments "/ab status"
      Then the comment delivery should return HTTP 204
      And the status reply should explain "<decision>"
      And the status request should cause no automation mutations

      Examples:
        | setting        | decision                       |
        | auto-merge     | Repository automation disabled |
        | squash merging | Squash merging unavailable     |

    Scenario Outline: Explain Dependabot policy before label authorization
      Given the Dependabot pull request for branch "contribution" is "<condition>"
      And branch "contribution" has the "automerge" label
      And GitHub disabled auto-merge for branch "contribution" "after the maintainer disables it"
      When a maintainer with "write" access comments "/ab status"
      Then the comment delivery should return HTTP 204
      And the status reply should explain "<decision>"
      And the status reply should include "Dependabot"
      And the status request should cause no automation mutations

      Examples:
        | condition               | decision                       |
        | waiting for checks      | Would enable squash auto-merge |
        | ready to merge          | Would squash-merge             |
        | failing optional checks | Would squash-merge             |

    Scenario Outline: Explain the need to renew revoked label authorization
      Given a pull request from "a fork" for branch "contribution" is "waiting for checks"
      And branch "contribution" has the "automerge" label
      And GitHub disabled auto-merge for branch "contribution" "<revocation>"
      And the automation events span multiple pages
      When a maintainer with "write" access comments "/ab status"
      Then the comment delivery should return HTTP 204
      And the status reply should explain "Waiting for renewed authorization"
      And the status reply should include "Remove and reapply"
      And the status reply should include "automerge"
      And the status reply should identify the observed revision and request with auto-merge "Disabled"
      And the status request should cause no automation mutations

      Examples:
        | revocation                                 |
        | after the maintainer disables it           |
        | at the same time as the label is reapplied |

    Scenario: A label without its authorization history needs renewal
      Given a pull request from "a fork" for branch "contribution" is "waiting for checks"
      And branch "contribution" has the "automerge" label
      And the labeled operation fails with "empty history"
      When a maintainer with "write" access comments "/ab status"
      Then the comment delivery should return HTTP 204
      And the status reply should explain "Waiting for renewed authorization"
      And the status request should cause no automation mutations

    Scenario Outline: Disclose unavailable evidence instead of guessing a policy outcome
      Given a pull request from "the repository" for branch "contribution" is "<condition>"
      And branch "contribution" has the "<label>" label
      And the labeled operation fails with "<failure>"
      When a maintainer with "write" access comments "/ab status"
      Then the comment delivery should return HTTP 204
      And the status reply should explain "Assessment unavailable"
      And the status reply should include "<evidence>"
      And the status request should cause no automation mutations

      Examples:
        | condition                       | label     | failure            | evidence   |
        | waiting for checks              | automerge | history HTTP       | history    |
        | waiting for checks              | automerge | incomplete history | history    |
        | waiting with auto-merge enabled | bug       | comparison         | comparison |
        | waiting with auto-merge enabled | bug       | missing comparison | comparison |

    Scenario Outline: Do not publish status for unauthorized or unrelated commands
      Given a pull request from "the repository" for branch "contribution" is "waiting for checks"
      When a maintainer with "<permission>" access comments "<command>"
      Then the comment delivery should return HTTP <status>
      And no status reply should be published
      And the status request should cause no automation mutations

      Examples:
        | permission | command           | status |
        | read       | /ab status        | 403    |
        | none       | /ab status        | 403    |
        | write      | /ab status please | 204    |
        | write      | /AB status        | 204    |
        | write      | /ab STATUS        | 204    |

    Scenario Outline: Re-read the original status request before publication
      Given a pull request from "the repository" for branch "contribution" is "waiting for checks"
      And its status request is "<condition>"
      When GitHub delivers that comment
      Then the comment delivery should return HTTP <status>
      And no status reply should be published
      And the status request should cause no automation mutations

      Examples:
        | condition           | status |
        | edited              | 204    |
        | now different text  | 204    |
        | now edited          | 204    |
        | for another PR      | 503    |
        | a bot request       | 204    |
        | from another author | 503    |

    Scenario Outline: Refuse to publish a changing status snapshot
      Given a pull request from "the repository" for branch "contribution" is "waiting for checks"
      And branch "contribution" has the "automerge" label
      And its "<field>" changes during status assessment
      When a maintainer with "write" access comments "/ab status"
      Then the comment delivery should return HTTP 409
      And no status reply should be published
      And the command comment should show "receipt and rejection" feedback
      And the status request should cause no automation mutations

      Examples:
        | field       |
        | head commit |
        | base commit |
        | draft       |
        | state       |
        | readiness   |
        | auto-merge  |
        | labels      |
        | identity    |

    Scenario Outline: Refuse to publish after repository policy changes
      Given a pull request from "the repository" for branch "contribution" is "waiting for checks"
      And its repository "<setting>" setting changes during status assessment
      When a maintainer with "write" access comments "/ab status"
      Then the comment delivery should return HTTP 409
      And no status reply should be published
      And the command comment should show "receipt and rejection" feedback
      And the status request should cause no automation mutations

      Examples:
        | setting        |
        | auto-merge     |
        | squash merging |

    Scenario Outline: Revalidate authorization before publishing the assessment
      Given a pull request from "the repository" for branch "contribution" is "waiting for checks"
      And its status authorization changes during assessment through "<change>"
      When a maintainer with "write" access comments "/ab status"
      Then the comment delivery should return HTTP <status>
      And no status reply should be published
      And the command comment should show "receipt and rejection" feedback
      And the status request should cause no automation mutations

      Examples:
        | change               | status |
        | revoked write access | 403    |
        | an edited comment    | 204    |

    Scenario Outline: Never acknowledge an unconfirmed status publication
      Given a pull request from "the repository" for branch "contribution" is "waiting for checks"
      And the labeled operation fails with "<failure>"
      When a maintainer with "write" access comments "/ab status"
      Then the comment delivery should return HTTP 503
      And exactly 1 status publication attempts should reach GitHub
      And status publication should not be reported as accepted
      And the status request should cause no automation mutations

      Examples:
        | failure             |
        | reply HTTP          |
        | malformed reply     |
        | incomplete reply    |
        | missing reply ID    |
        | wrong reply body    |
        | mismatched reply    |
        | lost reply response |

    Scenario: The command deadline also bounds publishing a status reply
      Given a pull request from "the repository" for branch "contribution" is "waiting for checks"
      And the command has a short processing deadline
      And the labeled operation fails with "reply timeout"
      When a maintainer requests status while publication is slow
      Then the comment delivery should return HTTP 503
      And the command should finish within one second
      And exactly 1 status publication attempts should reach GitHub
      And status publication should not be reported as accepted
      And the status request should cause no automation mutations

    Scenario: Redelivery can publish another revision-specific status reply
      Given a pull request from "the repository" for branch "contribution" is "waiting for checks"
      When GitHub delivers the same status request twice
      Then the comment delivery should return HTTP 204
      And exactly 2 status publication attempts should reach GitHub
      And the status reply should identify the observed revision and request with auto-merge "Disabled"
      And the status request should cause no automation mutations
