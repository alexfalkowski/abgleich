@manual @commands
Feature: Explain failed reviews
  Maintainers can see why a review failed and operators can trace the failed provider request.

  Scenario Outline: Explain a rejected provider request without exposing its contents
    Given AI <kind> reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the AI provider rejects the review because of "<problem>"
    When a maintainer with "write" access comments "/ab <command>"
    Then the <kind> review should fail without publishing findings
    And its failure comment should explain "<reason>" with recovery advice
    And its folded failure details should identify the request and revisions
    And the provider failure diagnostics should be:
      | http_status         | <status> |
      | provider_code       | <code>   |
      | provider_type       | <type>   |
      | provider_request_id | known    |
    And provider error contents should remain private
    And exactly 1 request should reach the AI service

    @codereview
    Examples:
      | kind     | command    | problem                  | reason                   | status | code                      | type                  |
      | code     | codereview | rate limiting            | rate limited             | 429    | rate_limit_exceeded       | rate_limit_error      |
      | code     | codereview | invalid credentials      | rejected its credentials | 401    | invalid_api_key           | authentication_error  |
      | code     | codereview | insufficient quota       | quota or spending limit  | 429    | insufficient_quota        | insufficient_quota    |
      | code     | codereview | too much source          | context window           | 400    | context_length_exceeded   | invalid_request_error |
      | code     | codereview | invalid reasoning state  | encrypted reasoning      | 400    | invalid_encrypted_content | invalid_request_error |
      | code     | codereview | denied access            | denied access            | 403    |                           | permission_error      |
      | code     | codereview | an unavailable model     | model or endpoint        | 404    | model_not_found           | invalid_request_error |
      | code     | codereview | unrecognized diagnostics | rejected the request     | 400    |                           |                       |

    @secreview
    Examples:
      | kind     | command    | problem                  | reason                   | status | code                      | type                  |
      | security | secreview  | an unavailable provider  | server error             | 503    | server_error              | server_error          |

  @codereview
  Scenario Outline: Explain failures without a provider HTTP error
    Given AI code reviews are configured for this repository
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    And the AI service supplies "<answer>"
    When a maintainer with "write" access comments "/ab codereview"
    Then the code review should fail without publishing findings
    And its failure comment should explain "<reason>" with recovery advice
    And its folded failure details should identify the request and revisions
    And exactly 1 request should reach the AI service

    Examples:
      | answer                  | reason                        |
      | a refusal               | refused the request           |
      | an unfinished answer    | incomplete or failed response |
      | no answer               | returned no response          |
      | an answer arriving late | exceeded its deadline         |
