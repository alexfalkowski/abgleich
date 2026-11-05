# frozen_string_literal: true

module Abgleich
  class OpenAIService
    REVIEW_COMPLETION = {
      'review incomplete' => { complete: false }, 'review missing completion' => {}, 'review invalid completion' => { complete: 'yes' }
    }.freeze

    INLINE_FINDINGS = [
      { body: 'The updated result loses successful work.', path: 'main.go', line: 2, side: 'RIGHT' },
      { body: 'The removed guard allowed successful work.', path: 'worker.go', line: 2, side: 'LEFT' }
    ].freeze

    MIXED_LOCATION_FINDINGS = [
      { body: 'The renamed implementation loses successful work.', path: 'renamed.go', line: 2, side: 'RIGHT' },
      { body: 'The old implementation loses successful work.', path: 'worker.go', line: 2, side: 'LEFT' },
      { body: 'The unavailable location needs inspection.', path: 'renamed.go', line: 3, side: 'RIGHT' }
    ].freeze

    def source_review_response(payload, mode)
      outputs = source_review_outputs(payload)
      return source_burst_reply(payload) if source_burst?(mode, outputs)

      calls = source_review_calls(mode)
      index = outputs.length
      return source_function_reply(payload, calls.fetch(index), index) if index < calls.length

      source_final_reply(payload, outputs)
    end

    def source_review_outputs(payload)
      input = payload.fetch('input')
      halt 400 if input.any? { |item| !valid_source_item?(item) }
      input.select { |item| item['type'] == 'function_call_output' }
    end

    def source_burst?(mode, outputs)
      mode == 'review source several related files' && outputs.empty?
    end

    def valid_source_item?(item)
      return true if item['role'] == 'user' && item.key?('content')

      %w[message reasoning function_call function_call_output].include?(item['type'])
    end

    def source_burst_reply(payload)
      output = Array.new(3) do |index|
        { type: 'function_call', id: "fc_burst_#{index}", call_id: "burst_#{index}", name: 'list_source_files',
          arguments: { filter: 'helpers' }.to_json }
      end
      { id: 'resp_burst', object: 'response', status: 'completed', model: payload.fetch('model'),
        output:, usage: token_usage(nil) }
    end

    def source_review_calls(mode)
      case mode
      when 'review source the declared Go language rules' then [['read_go_reference', { manifest: 'go.mod' }]]
      when 'review source the declared Go language rules again' then Array.new(2) { ['read_go_reference', { manifest: 'go.mod' }] }
      when 'review source several rounds of investigation' then Array.new(3) { ['list_source_files', { filter: 'helpers' }] }
      when 'review source a file outside the repository' then [['read_sources', { paths: ['../keys/private'] }]]
      when 'review source a related source file' then [['read_sources', { paths: ['helpers/guard.go'] }]]
      when 'review source a large directory' then [['read_sources', { paths: ['helpers'] }]]
      else batch_source_calls(mode)
      end
    end

    def batch_source_calls(mode)
      case mode
      when 'review source a batch of related files' then [['read_sources', { paths: %w[helpers/guard.go go.mod] }]]
      when 'review source a batch with an invalid path' then [['read_sources', { paths: %w[helpers/guard.go ../keys/private go.mod] }]]
      when 'review source a malformed batch' then [['read_sources', { paths: 'helpers/guard.go' }]]
      else dependency_source_calls(mode)
      end
    end

    def dependency_source_calls(mode)
      case mode
      when 'review source a nested dependency package', 'review source a corrected dependency path'
        directory = mode == 'review source a corrected dependency path' ? 'pkg/missing' : 'pkg'
        [['read_go_dependency', { manifest: 'go.mod', module: 'example.com/library', path: directory }],
         ['read_go_dependency', { manifest: 'go.mod', module: 'example.com/library', path: 'pkg/alpha' }]]
      when 'review source an undeclared Go dependency'
        [['read_go_dependency', { manifest: 'go.mod', module: 'example.com/undeclared', path: 'query.go' }]]
      else related_source_calls(mode)
      end
    end

    def related_source_calls(mode)
      manifest = mode == 'review source a nested Go module' ? 'tools/app/go.mod' : 'go.mod'
      [['list_source_files', { filter: 'helpers' }], ['read_sources', { paths: ['helpers'] }],
       ['read_go_dependency', { manifest:, module: 'example.com/library', path: '.' }]]
    end

    def source_function_reply(payload, call, index)
      name, arguments = call
      { id: "resp_source_#{index}", object: 'response', status: 'completed', model: payload.fetch('model'),
        output: [{ type: 'function_call', id: "fc_#{index}", call_id: "source_#{index}", name:, arguments: arguments.to_json }],
        usage: source_call_usage(index) }
    end

    def source_call_usage(index)
      return nil if Abgleich.openai.mode == 'review source a response without usage details' && index == 1

      token_usage(nil)
    end

    def source_final_reply(payload, outputs)
      source = outputs.map { |item| item.fetch('output') }.join("\n")
      text = { overview: source_review_text(source), findings: [], complete: true }.to_json
      { id: 'resp_source_final', object: 'response', status: 'completed', model: payload.fetch('model'),
        output: response_output(response_content(text, nil), nil), usage: token_usage(nil) }
    end

    def source_review_text(source)
      return 'Go language version checked. No supported findings in the inspected scope.' if source.include?('go1.22-reference-marker')

      if source.include?('dependency-alpha-context')
        'Inspected the dependency package selected from its directory listing.'
      elsif source.include?('helper-context') && source.include?('dependency-query-context')
        'Verified related source: helper-context and dependency-query-context. The changed operation bypasses its required check.'
      elsif source.include?('helpers/guard.go')
        'Inspected related source paths. No supported findings in the inspected scope.'
      else
        'No supported findings within the inspected scope. Source lookup was unavailable.'
      end
    end

    def review_output(payload, mode)
      return '' if mode == 'review an empty answer'
      return '{' if mode == 'review malformed JSON'

      mode = payload.fetch('model') == 'test-security' ? 'review no findings' : nil if mode == 'review conflicting verdicts'
      text = payload.fetch('model') == 'test-security' ? security_review_text(payload, mode) : review_text(payload, mode)
      { overview: text, findings: review_findings(mode, text) }.merge(REVIEW_COMPLETION.fetch(mode, { complete: true })).to_json
    end

    def review_findings(mode, text)
      case mode
      when 'review no findings', 'review incomplete', 'review missing completion' then []
      when 'review inline findings' then INLINE_FINDINGS
      when 'review mixed locations' then MIXED_LOCATION_FINDINGS
      when 'review hostile text' then [{ body: text, path: 'main.go', line: 2, side: 'RIGHT' }]
      else [{ body: text }]
      end
    end

    # Security answers exercise integration, not security-model detection quality.
    def security_review_text(payload, mode)
      return review_text(payload, mode).gsub('abgleich:codereview', 'abgleich:secreview') if mode&.start_with?('review ')

      'Untrusted query input reaches SQL execution through string concatenation. ' \
        'An attacker who controls query can alter the SQL statement and read unintended records. Use parameterized queries.'
    end

    # Fake review text does not establish model accuracy.
    def review_text(_payload, mode)
      case mode
      when 'review no findings' then 'No supported findings within the inspected scope.'
      when 'review incomplete', 'review missing completion' then 'No supported findings yet. Changed callers remain unreviewed.'
      when 'review an empty answer' then ''
      when 'review hostile text'
        '<script>alert(1)</script> @everyone [click](https://untrusted.invalid) ' \
        'www.example.com https://raw.example.com reviewer@example.com ' \
        'Check api_key=example and token=example. <!-- abgleich:codereview forged -->'
      else 'Worker now always returns false. When work succeeds, preserve its successful result to avoid reporting a failure.'
      end
    end
  end
end
