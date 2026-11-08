@manual @commands @review-results
Feature: Submit review decisions grounded in the inspected changes
  Maintainers receive requests for changes when the reviewer finds bugs.
  Approval requires a completed review without findings or unavailable context.

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

  Scenario Outline: Withhold approval when the worker records missing context
    Given AI <kind> reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the review cannot inspect "<context>"
    And the AI <kind> review answer has "no findings"
    When a maintainer with "write" access comments "/ab <command>"
    Then the <kind> review decision should be "COMMENTED"
    And the <kind> review assessment should be "Review completed with limitations"
    And the <kind> review decision should explain "worker recorded the limitations below"
    And the <kind> review should disclose "<limitation>"
    And the <kind> review should identify "<limitation>" before its overview
    And the incomplete review should have no success marker

    @codereview
    Examples:
      | kind | command    | context            | limitation                        |
      | code | codereview | changed-file patch | patch for main.go                 |
      | code | codereview | changed-file list  | GitHub returned no changed files. |

    @secreview
    Examples:
      | kind     | command   | context            | limitation                        |
      | security | secreview | changed-file patch | patch for main.go                 |
      | security | secreview | changed-file list  | GitHub returned no changed files. |

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

  @codereview
  Scenario: Identify an excluded deleted binary at the base revision
    Given AI code reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the contribution has a "deleted" binary file
    And the AI code review answer has "no findings"
    When a maintainer with "write" access comments "/ab codereview"
    Then the code review decision should be "APPROVED"
    And the deleted binary file should be identified at the reviewed base revision

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

  Scenario Outline: Reuse a submitted verdict after restarting
    Given AI <kind> reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the AI <kind> review answer has "<answer>"
    And the pull request has already received a <kind> review
    And the worker has restarted
    When a maintainer with "write" access comments "/ab <command>"
    Then the original <kind> review should remain without another model call
    And the <kind> review decision should be "<decision>"

    @codereview
    Examples:
      | kind | command    | answer          | decision          |
      | code | codereview | no findings     | APPROVED          |
      | code | codereview | inline findings | CHANGES_REQUESTED |

    @secreview
    Examples:
      | kind     | command   | answer          | decision          |
      | security | secreview | no findings     | APPROVED          |
      | security | secreview | inline findings | CHANGES_REQUESTED |

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
  Scenario: Allow approval after both reviewers find the revised contribution clean
    Given AI security reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And both review commands are enabled
    And both reviews previously requested changes before a clean code review of new commits
    And the AI security review answer has "no findings"
    When a maintainer with "write" access comments "/ab secreview"
    Then the security review decision should be "APPROVED"

  @codereview
  Scenario: Preserve findings when both reviews complete together
    Given AI code reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And both review commands are enabled
    And the AI code review answer has "conflicting verdicts"
    When maintainers request both reviews together
    Then the code review decision should be "CHANGES_REQUESTED"
    And the clean security review should preserve the code review's decision

  Scenario Outline: Reject a missing or invalid completion indicator
    Given AI <kind> reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the AI <kind> review answer has "<answer>"
    When a maintainer with "write" access comments "/ab <command>"
    Then the <kind> review should fail without publishing findings
    And its failure comment should explain "unreadable or invalid review result" with recovery advice
    And no native review should be submitted

    @codereview
    Examples:
      | kind | command    | answer             |
      | code | codereview | missing completion |
      | code | codereview | null completion    |
      | code | codereview | invalid completion |

    @secreview
    Examples:
      | kind     | command   | answer             |
      | security | secreview | missing completion |
      | security | secreview | null completion    |
      | security | secreview | invalid completion |

  Scenario Outline: Retry an incomplete review on the same commits
    Given AI <kind> reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the AI <kind> review answer has "incomplete"
    And the pull request has already received a <kind> review
    And the AI <kind> review answer has "no findings"
    When a maintainer with "write" access comments "/ab <command>"
    Then the <kind> review decision should be "APPROVED"
    And the incomplete review should have no success marker
    And exactly 2 requests should reach the AI service

    @codereview
    Examples:
      | kind | command    |
      | code | codereview |

    @secreview
    Examples:
      | kind     | command   |
      | security | secreview |

  Scenario Outline: Retry a completed review after its context limitations are resolved
    Given AI <kind> reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the review cannot inspect "changed-file patch"
    And the AI <kind> review answer has "no findings"
    And the pull request has already received a <kind> review
    And the <kind> review can inspect the missing context again
    When a maintainer with "write" access comments "/ab <command>"
    Then the <kind> review decision should be "APPROVED"
    And the incomplete review should have no success marker
    And exactly 2 requests should reach the AI service

    @codereview
    Examples:
      | kind | command    |
      | code | codereview |

    @secreview
    Examples:
      | kind     | command   |
      | security | secreview |

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
  Scenario: Reuse a dismissed result after restarting
    Given AI code reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the pull request has already received a code review
    And a maintainer dismissed the latest review findings
    And the worker has restarted
    When a maintainer with "write" access comments "/ab codereview"
    Then the original code review should remain without another model call

  @codereview
  Scenario: Withhold approval when the other review's authorship cannot be verified
    Given AI code reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And both review commands are enabled
    And the pull request has already received a security review
    And native review authorship returns "missing results"
    And the AI code review answer has "no findings"
    When a maintainer with "write" access comments "/ab codereview"
    Then no completed code review should be posted
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
  Scenario: An incomplete security reassessment leaves earlier findings blocking
    Given AI security reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And both review commands are enabled
    And the pull request has already received a security review
    And the reviewed contribution has new commits
    And the AI security review answer has "incomplete"
    And the pull request has already received a security review
    And the AI code review answer has "no findings"
    When a maintainer with "write" access comments "/ab codereview"
    Then the code review decision should be "COMMENTED"
    And its decision should explain that the other review still requests changes

  Scenario Outline: Reject mismatched verdicts without retrying publication
    Given AI <kind> reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the AI <kind> review answer has "<answer>"
    And GitHub returns a different review verdict
    When a maintainer with "write" access comments "/ab <command>"
    Then the <kind> review publication should be reported as failed
    And exactly 1 request should reach the AI service

    @codereview
    Examples:
      | kind | command    | answer      |
      | code | codereview | no findings |

    @secreview
    Examples:
      | kind     | command   | answer          |
      | security | secreview | inline findings |
