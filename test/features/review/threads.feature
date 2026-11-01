@github @manual @commands @codereview @secreview @review
Feature: Retire outdated advisory conversations
  Existing App conversations are maintained without another review request or AI inference.

  Scenario Outline: Retire only conversations made outdated by new commits
    Given AI <kind> reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the pull request has these existing review conversations:
      | conversation | kind   | state   |
      | stale        | <kind> | current |
      | current      | <kind> | current |
    When new commits make only conversation "stale" outdated
    Then review conversations should be:
      | conversation | resolved |
      | stale        | true     |
      | current      | false    |
    And conversation "stale" should receive exactly 1 resolution request across later checks
    And no AI inference should be requested for conversation maintenance

    Examples:
      | kind     |
      | code     |
      | security |

  Scenario: Respect conversation ownership and enabled review namespaces
    Given AI code reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the pull request has these existing review conversations:
      | conversation       | kind     | state                         |
      | owned              | code     | outdated                      |
      | human-root         | code     | another root author           |
      | other-app-review   | code     | another review author         |
      | ordinary-review    | code     | no result marker              |
      | security-disabled  | security | outdated                      |
      | already-resolved   | code     | resolved                      |
      | permission-missing | code     | cannot resolve                |
    When scheduled review maintenance runs
    Then review conversations should be:
      | conversation       | resolved |
      | owned              | true     |
      | human-root         | false    |
      | other-app-review   | false    |
      | ordinary-review    | false    |
      | security-disabled  | false    |
      | already-resolved   | true     |
      | permission-missing | false    |
    And only conversation "owned" should receive a resolution request
    And no AI inference should be requested for conversation maintenance

  Scenario: Rediscover older conversations after restart
    Given AI code reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And 100 current conversations precede an outdated App conversation
    When the worker restarts with those conversations already on GitHub
    Then review conversations should be:
      | conversation | resolved |
      | older-page   | true     |
    And conversation "older-page" should receive exactly 1 resolution request across later checks
    And no AI inference should be requested for conversation maintenance

  Scenario Outline: Respect the refreshed conversation state
    Given AI code reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the pull request has these existing review conversations:
      | conversation | kind | state    |
      | changed      | code | outdated |
    And conversation "changed" becomes "<state>" before resolution
    When scheduled review maintenance runs
    Then conversation "changed" should receive no resolution requests
    And no AI inference should be requested for conversation maintenance

    Examples:
      | state          |
      | current        |
      | resolved       |
      | cannot resolve |

  Scenario: Recover a failed resolution while other conversations progress
    Given AI code reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the pull request has these existing review conversations:
      | conversation | kind | state    |
      | unavailable  | code | outdated |
      | available    | code | outdated |
    And GitHub cannot resolve conversation "unavailable" while other conversations progress
    When GitHub becomes able to resolve conversations again
    Then review conversations should be:
      | conversation | resolved |
      | unavailable  | true     |
      | available    | true     |
    And no AI inference should be requested for conversation maintenance

  Scenario: Read remote state after losing a resolution response
    Given AI code reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the pull request has these existing review conversations:
      | conversation | kind | state    |
      | unconfirmed  | code | outdated |
    And GitHub loses the response after resolving a conversation
    When scheduled review maintenance runs
    Then review conversations should be:
      | conversation | resolved |
      | unconfirmed  | true     |
    And conversation "unconfirmed" should receive exactly 1 resolution request across later checks
    And no AI inference should be requested for conversation maintenance

  Scenario: Maintain both enabled features on a draft fork without merge automation
    Given AI security reviews are configured for this repository
    And security reviews and summaries are independently enabled
    And a pull request from "a fork" for branch "contribution" is "a draft"
    And the reviewed repository has hold and automatic merging disabled
    And the pull request has these existing review conversations:
      | conversation | kind     | state    |
      | code         | code     | outdated |
      | security     | security | outdated |
    When scheduled review maintenance runs
    Then review conversations should be:
      | conversation | resolved |
      | code         | true     |
      | security     | true     |
    And no AI inference should be requested for conversation maintenance

  Scenario: Leave conversations on closed pull requests untouched
    Given AI code reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "closed"
    And the pull request has these existing review conversations:
      | conversation | kind | state    |
      | closed       | code | outdated |
    When scheduled review maintenance runs
    Then conversation "closed" should receive no resolution requests
    And no AI inference should be requested for conversation maintenance
