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

    before do
      content_type :json
      halt 401 unless request.env['HTTP_AUTHORIZATION'] == 'Bearer nonnative-summary-key'
    end

    post '/v1/responses' do
      Abgleich.openai.record_request(JSON.parse(request.body.read))
      mode = Abgleich.openai.mode
      sleep 0.8 if mode == 'slow'
      # The 3s test job allows at most 1.5s for inference.
      sleep 2 if mode == 'timeout'
      halt 429, { error: { message: 'Rate limited', type: 'rate_limit_error' } }.to_json if mode == 'rate limit'
      halt 200, 'null' if mode == 'null response'

      output = SUMMARY
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
      content = [{ type: 'output_text', text: output, annotations: [] }]
      content = [{ type: 'refusal', refusal: 'I cannot provide this summary.' }] if mode == 'refusal'
      { id: 'resp_summary', object: 'response', status: mode == 'incomplete' ? 'incomplete' : 'completed',
        output: [{ id: 'msg_summary', type: 'message', role: 'assistant', status: 'completed', content: }],
        usage: { input_tokens: 100, output_tokens: 40, total_tokens: 140 } }.to_json
    end
  end
end
