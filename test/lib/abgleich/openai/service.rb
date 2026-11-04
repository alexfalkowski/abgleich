# frozen_string_literal: true

module Abgleich
  class OpenAIService < Nonnative::HTTPService
    SUMMARY = <<~TEXT
      The worker now processes requests predictably.

      Changes:
      Requests wait in a background queue.
      Other comment commands stay responsive.

      Where to look:
      test.go: check how requests are processed.
    TEXT

    SUMMARY_OF_CI_WORKFLOW = <<~TEXT
      CI now runs feature groups separately and combines their coverage before release.

      Changes:
      Six feature groups have matching local commands.
      Coverage profiles stay isolated until aggregation.
      Deployment waits for both packages and Docker images.

      Where to look:
      .circleci/config.yml: follow the test groups through coverage and deployment.
      Makefile: find the local command for each group.
      scripts/coverage-workspace: see how coverage is kept separate between groups.
    TEXT

    SUMMARY_WITH_EXTRA_CHANGES = <<~TEXT
      Requests stay queued.

      Changes:
      Requests wait in a queue.
      Commands remain responsive.
      Requests keep their revision.
      Restart drops pending requests.
    TEXT

    SUMMARY_WITH_LIMITS = <<~TEXT
      The worker now processes requests predictably.

      Where to look:
      main.go: handles the incoming requests.
      worker.go: processes requests in the background.
      test.go: tests cover the reported behavior.

      Limits: image.png has no patch. @everyone [details](https://untrusted.invalid)
    TEXT

    CACHE_USAGE = {
      'cached usage' => { cached_tokens: 20, cache_write_tokens: 0 },
      'cache writes' => { cached_tokens: 20, cache_write_tokens: 30 },
      'invalid usage' => { cached_tokens: 101, cache_write_tokens: 0 },
      'overlapping writes' => { cached_tokens: 80, cache_write_tokens: 30 }
    }.freeze

    REVIEW_FAILURES = {
      'rate limiting' => [429, 'rate_limit_exceeded', 'rate_limit_error'],
      'an unavailable provider' => [503, 'server_error', 'server_error'],
      'invalid credentials' => [401, 'invalid_api_key', 'authentication_error'],
      'insufficient quota' => [429, 'insufficient_quota', 'insufficient_quota'],
      'too much source' => [400, 'context_length_exceeded', 'invalid_request_error'],
      'invalid reasoning state' => [400, 'invalid_encrypted_content', 'invalid_request_error'],
      'denied access' => [403, nil, 'permission_error'],
      'an unavailable model' => [404, 'model_not_found', 'invalid_request_error'],
      'unrecognized diagnostics' => [400, 'nonnative-summary-key', 'func Worker() bool']
    }.freeze

    before do
      content_type :json
      halt 401 unless request.env['HTTP_AUTHORIZATION'] == 'Bearer nonnative-summary-key'
    end

    post '/v1/responses' do
      payload = JSON.parse(request.body.read)
      mode = prepare_response(payload)
      sleep 0.8 if mode == 'slow'
      # The 3s test job allows at most 1.5s for inference.
      sleep 2 if mode == 'timeout'
      halt 429, { error: { message: 'Rate limited', type: 'rate_limit_error' } }.to_json if mode == 'rate limit'
      halt 400, { error: { message: 'Context window exceeded', code: 'context_length_exceeded' } }.to_json if mode == 'context limit'
      halt 200, 'null' if mode == 'null response'

      halt 200, source_review_response(payload, mode).to_json if mode&.start_with?('review source')

      output = payload.fetch('model') == 'test-review' ? review_output(payload, mode) : SUMMARY
      output = review_output(payload, mode) if payload.fetch('model') == 'test-security'
      case mode
      when 'hostile prose'
        output = "The worker now processes requests predictably.\n\nChanges:\n" \
                 '<script>alert(1)</script> @everyone [click](https://untrusted.invalid) ![track](https://untrusted.invalid/image)'
      when 'ci workflow' then output = SUMMARY_OF_CI_WORKFLOW
      when 'extra changes' then output = SUMMARY_WITH_EXTRA_CHANGES
      when 'empty changes' then output = "Requests stay queued.\n\nChanges:\n"
      when 'limits note' then output = SUMMARY_WITH_LIMITS
      when 'removed pointer' then output = "The old entry point was removed.\n\nWhere to look:\nmain.go: deleted entry point."
      end
      { id: 'resp_summary', object: 'response', status: mode == 'incomplete' ? 'incomplete' : 'completed',
        model: mode == 'different response model' ? 'other-model' : payload.fetch('model'),
        output: response_output(response_content(output, mode), mode),
        usage: token_usage(mode) }.to_json
    end

    private

    def prepare_response(payload)
      Abgleich.openai.record_request(payload)
      mode = Abgleich.openai.mode
      reject_review_request(mode.delete_prefix('diagnostic ')) if mode&.start_with?('diagnostic ')
      mode
    end

    def reject_review_request(problem)
      status, code, type = REVIEW_FAILURES.fetch(problem)
      request_id = problem == 'unrecognized diagnostics' ? 'req_sk-provider-secret' : 'req_0123456789abcdef0123456789abcdef'
      headers 'X-Request-Id' => request_id
      message = 'provider-private-detail nonnative-summary-key func Worker() bool @everyone <script> https://untrusted.invalid'
      halt status, { error: { code:, type:, message:, param: 'sk-provider-secret' } }.to_json
    end

    def response_output(content, mode)
      messages = [{ id: 'msg_summary', type: 'message', role: 'assistant', status: 'completed', content: }]

      if mode == 'review a request outside the approved capabilities'
        messages << { id: 'call_review', type: 'function_call', call_id: 'call_review', name: 'mutate', arguments: '{}' }
      end

      messages
    end

    def response_content(output, mode)
      content = [{ type: 'output_text', text: output, annotations: [] }]
      refusal = { type: 'refusal', refusal: 'I cannot provide this answer.' }
      return [refusal] if mode == 'refusal'

      content << refusal if mode == 'review a refusal alongside an answer'
      content
    end

    # Usage fixtures distinguish ordinary input, cache reads/writes, and absent metadata.
    def token_usage(mode)
      return nil if mode == 'missing usage'

      usage = { input_tokens: 100, input_tokens_details: CACHE_USAGE.fetch(mode, { cached_tokens: 0, cache_write_tokens: 0 }),
                output_tokens: mode == 'negative usage' ? -1 : 40, output_tokens_details: { reasoning_tokens: 10 }, total_tokens: 140 }
      usage.delete(:input_tokens_details) if mode == 'missing cache breakdown'
      usage
    end
  end
end
