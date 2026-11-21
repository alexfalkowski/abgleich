@manual @commands @review-results
Feature: Ground review decisions in the inspected changes
  Blockers request changes. Approval requires a complete assessment and preserves outstanding requests for changes from other reviews.

  Scenario Outline: Select the decision from the completed assessment
    Given AI <kind> reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the AI <kind> review answer has "<answer>"
    When a maintainer with "write" access comments "/ab <command>"
    Then the <kind> review decision should be "<decision>"
    And the <kind> review assessment should be "<assessment>"
    And the <kind> review decision should explain "<reason>"
    And no review should change merge automation

    @codereview
    Examples:
      | kind | command    | answer          | decision          | assessment        | reason                      |
      | code | codereview | inline findings | CHANGES_REQUESTED | Findings          | Request changes.            |
      | code | codereview | no findings     | APPROVED          | No findings       | Approve.                    |
      | code | codereview | incomplete      | COMMENTED         | Review unfinished | the reviewer did not finish |

    @secreview
    Examples:
      | kind     | command   | answer          | decision          | assessment        | reason                      |
      | security | secreview | inline findings | CHANGES_REQUESTED | Findings          | Request changes.            |
      | security | secreview | no findings     | APPROVED          | No findings       | Approve.                    |
      | security | secreview | incomplete      | COMMENTED         | Review unfinished | the reviewer did not finish |

  @testreview
  Scenario Outline: Approve a complete review with only low-severity findings
    Given AI test reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the AI test review answer has "<answer>"
    When a maintainer with "write" access comments "/ab testreview"
    Then the test review decision should be "APPROVED"
    And the test review assessment should be "Advisory feedback"
    And the test review decision should explain "Feedback is advisory"
    And the review overview should show the low-severity finding at "<location>"
    And the native review should have no inline findings

    Examples:
      | answer                                 | location          |
      | low-severity findings                  | main.go:2 (RIGHT) |
      | low-severity findings with a path only | main.go           |

  @testreview
  Scenario Outline: Request changes when mixed severities include a blocking finding
    Given AI test reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the AI test review answer has "<answer>"
    When a maintainer with "write" access comments "/ab testreview"
    Then the test review decision should be "CHANGES_REQUESTED"
    And the review overview should show the low-severity finding at "main.go:2 (RIGHT)"
    And the inline findings should be:
      | path      | line | side |
      | worker.go | 2    | LEFT |

    Examples:
      | answer                              |
      | low- and medium-severity findings   |
      | medium- and low-severity findings   |
      | high- and low-severity findings     |
      | critical- and low-severity findings |

  @codereview
  Scenario: Request changes when the blocking finding cannot be placed inline
    Given AI code reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the AI code review answer has "a medium-severity finding outside the diff and a low-severity finding"
    When a maintainer with "write" access comments "/ab codereview"
    Then the code review decision should be "CHANGES_REQUESTED"
    And the code review should have no context limitations
    And the review overview should retain the finding outside the diff at "worker.go:3 (LEFT)"
    And the review overview should show the low-severity finding at "main.go:2 (RIGHT)"
    And the native review should have no inline findings

  @codereview
  Scenario: Withhold approval when a low-severity review is unfinished
    Given AI code reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the AI code review answer has "incomplete with low-severity findings"
    When a maintainer with "write" access comments "/ab codereview"
    Then the code review decision should be "COMMENTED"
    And the code review assessment should be "Review unfinished"
    And the review overview should show the low-severity finding at "main.go:2 (RIGHT)"
    And the native review should have no inline findings
    And the incomplete review should have no success marker

  @codereview
  Scenario: Keep optional browser coverage advisory even at medium severity
    Given AI code reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the AI code review answer has "optional browser coverage"
    When a maintainer with "write" access comments "/ab codereview"
    Then the code review decision should be "APPROVED"
    And the code review assessment should be "Advisory feedback"
    And the review overview should show the advisory about browser coverage
    And the native review should have no inline findings

  @codereview
  Scenario Outline: Request a missing browser check only when the description requires it
    Given AI code reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the PR description "<requirement>" a browser check
    And the AI code review answer has "<answer>"
    When a maintainer with "write" access comments "/ab codereview"
    Then the code review decision should be "<decision>"
    And the coverage finding should open <threads> inline conversations

    Examples:
      | requirement                        | answer                                 | decision          | threads |
      | requires                           | missing required browser coverage      | CHANGES_REQUESTED | 1       |
      | hides an example and then requires | missing required browser coverage      | CHANGES_REQUESTED | 1       |
      | requires a specific variant of     | missing required browser coverage      | APPROVED          | 0       |
      | mentions a declaration for         | missing required browser coverage      | APPROVED          | 0       |
      | has a hidden template example for  | missing required browser coverage      | APPROVED          | 0       |
      | has an inline template example for | missing required browser coverage      | APPROVED          | 0       |
      | has a fenced example for           | missing required browser coverage      | APPROVED          | 0       |
      | has a tilde-fenced example for     | missing required browser coverage      | APPROVED          | 0       |
      | has an indented example for        | missing required browser coverage      | APPROVED          | 0       |
      | suggests                           | a browser suggestion cited as required | APPROVED          | 0       |

  @codereview
  Scenario: Stop a queued review of a description the author has edited
    Given AI code reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the PR description "requires" a browser check
    And another code review is still being prepared
    And a maintainer's code review request is waiting
    When the author removes the browser requirement before the queued review starts
    Then the maintainer should be asked for a new review of the updated description
    And no native review should be submitted for the contribution
    And exactly 1 request should reach the AI service

  @codereview
  Scenario: Stop a stale review before sending the description to the AI service
    Given AI code reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the PR description "requires" a browser check
    And the author removes the browser requirement before source submission
    When a maintainer with "write" access comments "/ab codereview"
    Then the maintainer should be asked for a new review of the updated description
    And no native review should be submitted
    And exactly 0 requests should reach the AI service

  @codereview
  Scenario: Discard a review when the author removes the coverage requirement during inference
    Given AI code reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the PR description "requires" a browser check
    And the AI code review answer has "missing required browser coverage"
    And the author removes the browser requirement during review
    When a maintainer with "write" access comments "/ab codereview"
    Then the maintainer should be asked for a new review of the updated description
    And no native review should be submitted

  @codereview
  Scenario: Review the edited description when a new request arrives during inference
    Given AI code reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the PR description "requires" a browser check
    And the AI code review answer has "missing required browser coverage"
    And a code review of the original description is in progress
    When a maintainer removes the browser requirement and requests another code review
    Then the maintainer should be asked for a new review of the updated description
    And only the fresh code review should publish a verdict

  @codereview
  Scenario: Stop investigating when the author edits the description between provider turns
    Given repository source and public Go dependencies are available for review
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the PR description "requires" a browser check
    And the reviewer needs "a related source file"
    And the author removes the browser requirement during review
    When a maintainer with "write" access comments "/ab codereview"
    Then the maintainer should be asked for a new review of the updated description
    And the code review should stop before sharing related source

  @codereview
  Scenario: Discard an approval when the author edits the description during verdict history checks
    Given AI code reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And both review commands are enabled
    And the AI security review answer has "no findings"
    And the pull request has already received a security review
    And the AI code review answer has "no findings"
    And the author edits the description while the worker checks prior review authorship
    When a maintainer with "write" access comments "/ab codereview"
    Then the maintainer should be asked for a new review of the updated description
    And native review history should remain unchanged

  @testreview
  Scenario Outline: Request changes when a rewritten test no longer detects a material regression
    Given AI test reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the AI test review answer has "<answer>"
    When a maintainer with "write" access comments "/ab testreview"
    Then the test review decision should be "CHANGES_REQUESTED"
    And the inline findings should be:
      | path           | line | side   |
      | worker_test.go | 3    | <side> |

    Examples:
      | answer                                       | side  |
      | a changed test that misses a regression      | RIGHT |
      | a removed assertion that misses a regression | LEFT  |

  @testreview
  Scenario Outline: Keep a test regression advisory when it lacks a changed line or supporting evidence
    Given AI test reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the AI test review answer has "<answer>"
    When a maintainer with "write" access comments "/ab testreview"
    Then the test review decision should be "APPROVED"
    And the test review assessment should be "Advisory feedback"
    And the native review should have no inline findings

    Examples:
      | answer                                          |
      | an unchanged test line that misses a regression |
      | a changed test finding without an explanation   |

  Scenario Outline: Withhold approval when the worker records missing context
    Given AI <kind> reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the review cannot inspect "<context>"
    And the AI <kind> review answer has "<answer>"
    When a maintainer with "write" access comments "/ab <command>"
    Then the <kind> review decision should be "COMMENTED"
    And the <kind> review assessment should be "Review completed with limitations"
    And the <kind> review decision should explain "worker recorded the limitations below"
    And the <kind> review should disclose "<limitation>"
    And the <kind> review should identify "<limitation>" before its overview
    And the incomplete review should have no success marker

    @codereview
    Examples:
      | kind | command    | context            | answer                | limitation                        |
      | code | codereview | changed-file patch | no findings           | patch for main.go                 |
      | code | codereview | changed-file patch | low-severity findings | patch for main.go                 |
      | code | codereview | changed-file list  | no findings           | GitHub returned no changed files. |

    @secreview
    Examples:
      | kind     | command   | context            | answer      | limitation                        |
      | security | secreview | changed-file patch | no findings | patch for main.go                 |
      | security | secreview | changed-file list  | no findings | GitHub returned no changed files. |

  Scenario Outline: Disclose source limitations even when the reviewer did not finish
    Given AI <kind> reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the review cannot inspect "changed-file patch"
    And the AI <kind> review answer has "incomplete"
    When a maintainer with "write" access comments "/ab <command>"
    Then the <kind> review assessment should be "Review unfinished"
    And the <kind> review overview should explain remaining work
    And the <kind> review decision should be "COMMENTED"
    And the <kind> review should identify "patch for main.go" before its overview
    And the incomplete review should have no success marker

    @codereview
    Examples:
      | kind | command    |
      | code | codereview |

    @secreview
    Examples:
      | kind     | command   |
      | security | secreview |

  @codereview
  Scenario: Withhold approval for missing text patches alongside an excluded binary
    Given AI code reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the review cannot inspect "patches and binary source"
    And the AI code review answer has "no findings"
    When a maintainer with "write" access comments "/ab codereview"
    Then the code review assessment should be "Review completed with limitations"
    And the code review decision should be "COMMENTED"
    And the code review should identify "patch for main.go" before its overview
    And the code review should disclose the excluded binary file "image.png"

  Scenario Outline: Approve a completed text review while excluding binary changes
    Given AI <kind> reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the contribution has a "<status>" binary file
    And the AI <kind> review answer has "no findings"
    When a maintainer with "write" access comments "/ab <command>"
    Then the <kind> review decision should be "APPROVED"
    And the <kind> review should disclose the excluded binary file "image.png"
    And the <kind> review should have no context limitations

    @codereview
    Examples:
      | kind | command    | status  |
      | code | codereview | changed |
      | code | codereview | deleted |

    @secreview
    Examples:
      | kind     | command   | status  |
      | security | secreview | changed |
      | security | secreview | deleted |

  Scenario Outline: Preserve the reviewer's decision when binary files are excluded
    Given AI <kind> reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the contribution has a "changed" binary file
    And the AI <kind> review answer has "<answer>"
    When a maintainer with "write" access comments "/ab <command>"
    Then the <kind> review decision should be "<decision>"
    And the <kind> review should disclose the excluded binary file "image.png"

    @codereview
    Examples:
      | kind | command    | answer          | decision          |
      | code | codereview | inline findings | CHANGES_REQUESTED |
      | code | codereview | incomplete      | COMMENTED         |

    @secreview
    Examples:
      | kind     | command   | answer          | decision          |
      | security | secreview | inline findings | CHANGES_REQUESTED |
      | security | secreview | incomplete      | COMMENTED         |

  Scenario Outline: Keep the other review's findings blocking after a restart
    Given AI <kind> reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And both review commands are enabled
    And the pull request has already received a <other> review
    And the worker has restarted
    And the AI <kind> review answer has "no findings"
    When a maintainer with "write" access comments "/ab <command>"
    Then the <kind> review decision should be "COMMENTED"
    And its decision should explain that the other review still requests changes

    @codereview
    Examples:
      | kind | command    | other    |
      | code | codereview | security |

    @secreview
    Examples:
      | kind     | command   | other |
      | security | secreview | code  |

  @codereview
  Scenario: Preserve security findings when the author pushes new commits
    Given AI security reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And both review commands are enabled
    And the pull request has already received a security review
    And the reviewed contribution has new commits
    And the AI code review answer has "no findings"
    When a maintainer with "write" access comments "/ab codereview"
    Then the code review decision should be "COMMENTED"
    And its decision should explain that the other review still requests changes

  @codereview
  Scenario: Preserve a pre-upgrade request for changes with only a low-severity finding
    Given AI security reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And both review commands are enabled
    And an earlier security review requested changes for a low-severity finding before the upgrade
    And the AI code review answer has "no findings"
    When a maintainer with "write" access comments "/ab codereview"
    Then the code review decision should be "COMMENTED"
    And its decision should explain that the other review still requests changes
    And the earlier security review should still request changes

  @secreview
  Scenario: Reuse a pre-upgrade request for changes on the same revisions
    Given AI security reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And an earlier security review requested changes for a low-severity finding before the upgrade
    And the AI security review answer has "no findings"
    When a maintainer with "write" access comments "/ab secreview"
    Then the command comment should show "receipt and acceptance" feedback
    And reusing the earlier security review should need no model call
    And native review history should remain unchanged

  Scenario Outline: Request changes after the other review has approved
    Given AI <kind> reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And both review commands are enabled
    And the AI <other> review answer has "no findings"
    And the pull request has already received a <other> review
    And the AI <kind> review answer has "inline findings"
    When a maintainer with "write" access comments "/ab <command>"
    Then the <kind> review decision should be "CHANGES_REQUESTED"

    @codereview
    Examples:
      | kind | command    | other    |
      | code | codereview | security |

    @secreview
    Examples:
      | kind     | command   | other |
      | security | secreview | code  |

  @secreview
  Scenario Outline: Allow approval after both reviewers clear their earlier requests for changes
    Given AI security reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And both review commands are enabled
    And both reviews previously requested changes before a code review of new commits reported "<answer>"
    And the AI security review answer has "no findings"
    When a maintainer with "write" access comments "/ab secreview"
    Then the security review decision should be "APPROVED"

    Examples:
      | answer                |
      | no findings           |
      | low-severity findings |

  @secreview
  Scenario: Honor a pre-upgrade review comment that cleared an earlier request for changes
    Given AI security reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And both review commands are enabled
    And both reviews previously requested changes before a code review of new commits reported "low-severity findings"
    And the clearing code review used the pre-upgrade assessment label
    And the AI security review answer has "no findings"
    When a maintainer with "write" access comments "/ab secreview"
    Then the security review decision should be "APPROVED"

  @secreview
  Scenario Outline: Ignore requests for changes from unauthenticated lookalikes
    Given AI security reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And both review commands are enabled
    And a native code review lookalike was posted by "<author>"
    And the AI security review answer has "no findings"
    When a maintainer with "write" access comments "/ab secreview"
    Then the security review decision should be "APPROVED"

    Examples:
      | author      |
      | a human     |
      | another App |

  @codereview
  Scenario: Respect a maintainer's dismissal of the other review's findings
    Given AI code reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And both review commands are enabled
    And the pull request has already received a security review
    And a maintainer dismissed the latest review findings
    And the AI code review answer has "no findings"
    When a maintainer with "write" access comments "/ab codereview"
    Then the code review decision should be "APPROVED"

  @codereview
  Scenario: Withhold approval when the other review's authorship cannot be verified
    Given AI code reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And both review commands are enabled
    And the pull request has already received a security review
    And native review authorship returns "missing results"
    And the AI code review answer has "no findings"
    When a maintainer with "write" access comments "/ab codereview"
    Then the code review should fail because authorship cannot be verified
    And native review history should remain unchanged
    And exactly 2 requests should reach the AI service

  Scenario Outline: Keep supported findings when some source is unavailable
    Given AI <kind> reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the review cannot inspect "changed-file patch"
    And the AI <kind> review answer has "<answer>"
    When a maintainer with "write" access comments "/ab <command>"
    Then the <kind> review decision should be "CHANGES_REQUESTED"

    @codereview
    Examples:
      | kind | command    | answer                   |
      | code | codereview | inline findings          |
      | code | codereview | incomplete with findings |

    @secreview
    Examples:
      | kind     | command   | answer                   |
      | security | secreview | inline findings          |
      | security | secreview | incomplete with findings |

  @codereview
  Scenario Outline: An incomplete security reassessment leaves earlier findings blocking
    Given AI security reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And both review commands are enabled
    And the pull request has already received a security review
    And the reviewed contribution has new commits
    And the AI security review answer has "<answer>"
    And the pull request has already received a security review
    And the AI code review answer has "no findings"
    When a maintainer with "write" access comments "/ab codereview"
    Then the code review decision should be "COMMENTED"
    And its decision should explain that the other review still requests changes

    Examples:
      | answer                                |
      | incomplete                            |
      | incomplete with low-severity findings |
