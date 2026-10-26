# frozen_string_literal: true

module Abgleich
  class OpenAIService < Nonnative::HTTPService
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

      output = "The worker now processes requests predictably.\n\nWhere to look:\ntest.go: tests cover the reported behavior."
      if mode == 'hostile prose'
        output = '<script>alert(1)</script> @everyone [click](https://untrusted.invalid) ![track](https://untrusted.invalid/image)'
      end
      content = [{ type: 'output_text', text: output, annotations: [] }]
      content = [{ type: 'refusal', refusal: 'I cannot provide this summary.' }] if mode == 'refusal'
      { id: 'resp_summary', object: 'response', status: mode == 'incomplete' ? 'incomplete' : 'completed',
        output: [{ id: 'msg_summary', type: 'message', role: 'assistant', status: 'completed', content: }],
        usage: { input_tokens: 100, output_tokens: 40, total_tokens: 140 } }.to_json
    end
  end
end
