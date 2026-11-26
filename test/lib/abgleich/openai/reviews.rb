# frozen_string_literal: true

module Abgleich
  class OpenAIService
    SUBMODULE_SOURCE_PATHS = {
      'review source a submodule directory' => '.', 'review source a nested submodule' => 'nested/private.go',
      'review source an invalid submodule path' => '../private'
    }.freeze

    def source_review_response(payload, mode)
      outputs = source_review_outputs(payload)
      return source_burst_reply(payload) if source_burst?(mode, outputs)

      calls = source_calls(mode)
      index = outputs.length
      return source_function_reply(payload, calls.fetch(index), index) if index < calls.length

      source_final_reply(payload, outputs, mode)
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

    def source_calls(mode)
      return omitted_design_calls(mode) if mode.start_with?('review source the omitted ')

      if ['review source a new base file', 'review source the base after a fix'].include?(mode)
        paths = ['helpers/defaults.go']
        paths << 'worker.go' if mode == 'review source the base after a fix'
        return %w[read_sources read_base_sources].map { |tool| [tool, { paths: }] }
      end

      ruby_source_calls(mode) || source_review_calls(mode)
    end

    def omitted_design_calls(mode)
      return [['read_base_sources', { paths: ['docs/agents/accepted-design'] }]] if mode.end_with?('target-base design')

      [['read_sources', { paths: %w[docs/agents/accepted-design/jobs.md docs/agents/accepted-design helpers/guard.go] }]]
    end

    def source_review_calls(mode)
      case mode
      when 'review source the declared Go language rules' then [['read_go_reference', { manifest: 'go.mod' }]]
      when 'review source the declared Go language rules again' then Array.new(2) { ['read_go_reference', { manifest: 'go.mod' }] }
      when 'review source several rounds of investigation' then Array.new(3) { ['list_source_files', { filter: 'helpers' }] }
      when 'review source a file outside the repository' then [['read_sources', { paths: ['../keys/private'] }]]
      when 'review source a related source file' then [['read_sources', { paths: ['helpers/guard.go'] }]]
      when 'review source a large directory' then [['read_sources', { paths: ['helpers'] }]]
      else repository_source_calls(mode)
      end
    end

    def repository_source_calls(mode)
      case mode
      when 'review source shared tooling paths'
        [['read_sources', { paths: %w[bin bin/README.md bin/build bin/build/check helpers/guard.go] }]]
      when 'review source a corrected repository file path', 'review source a corrected repository directory path'
        missing = mode == 'review source a corrected repository file path' ? 'helpers/missing.go' : 'missing/helpers'
        [['read_sources', { paths: [missing] }], ['read_sources', { paths: ['helpers/guard.go'] }]]
      when 'review source a file beneath an excluded path' then [['read_sources', { paths: ['helpers/guard.go/private.go'] }]]
      when 'review source an empty source directory' then [['read_sources', { paths: ['helpers/empty'] }]]
      else submodule_source_calls(mode)
      end
    end

    def submodule_source_calls(mode)
      case mode
      when 'review source shared tooling'
        [['list_source_files', { filter: 'bin' }],
         *%w[. README.md build build/check].map { |path| ['read_submodule', { submodule: 'bin', path: }] }]
      when 'review source a corrected submodule path'
        %w[missing README.md].map { |path| ['read_submodule', { submodule: 'bin', path: }] }
      when 'review source a submodule directory', 'review source a nested submodule', 'review source an invalid submodule path'
        [['read_submodule', { submodule: 'bin', path: SUBMODULE_SOURCE_PATHS.fetch(mode) }]]
      else batch_source_calls(mode)
      end
    end

    def batch_source_calls(mode)
      case mode
      when 'review source an invalid excerpt'
        [['read_go_dependency', { manifest: 'go.mod', module: 'example.com/library', path: 'query.go', start_line: 3, end_line: 2 }]]
      when 'review source an empty excerpt'
        [['read_go_dependency', { manifest: 'go.mod', module: 'example.com/library', path: 'query.go', start_line: 100, end_line: 110 }]]
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
        paths = [directory, 'pkg/alpha', 'pkg/alpha/query.go']
        paths.unshift('.') if mode == 'review source a nested dependency package'
        paths.map { |path| ['read_go_dependency', { manifest: 'go.mod', module: 'example.com/library', path: }] }
      when 'review source an undeclared Go dependency'
        [['read_go_dependency', { manifest: 'go.mod', module: 'example.com/undeclared', path: 'query.go' }]]
      else related_source_calls(mode)
      end
    end

    def related_source_calls(mode)
      return [['read_sources', { paths: ['.'] }]] if mode == 'review source the repository root'

      manifest = mode == 'review source a nested Go module' ? 'tools/app/go.mod' : 'go.mod'
      if mode.end_with?('excerpts')
        return ['.', 'query.go'].map do |path|
          ['read_go_dependency', { manifest:, module: 'example.com/library', path: }]
        end
      end
      [['list_source_files', { filter: 'helpers' }], ['read_sources', { paths: ['helpers'] }],
       ['read_go_dependency', { manifest:, module: 'example.com/library', path: 'query.go' }]]
    end

    def source_function_reply(payload, call, index)
      name, arguments = call
      arguments = arguments.merge(start_line: 2, end_line: 2) if Abgleich.openai.mode.end_with?('excerpts') && name == 'read_go_dependency'
      { id: "resp_source_#{index}", object: 'response', status: 'completed', model: payload.fetch('model'),
        output: [{ type: 'function_call', id: "fc_#{index}", call_id: "source_#{index}", name:, arguments: arguments.to_json }],
        usage: source_call_usage(index) }
    end

    def source_call_usage(index)
      return nil if Abgleich.openai.mode == 'review source a response without usage details' && index == 1

      token_usage(nil)
    end

    def source_final_reply(payload, outputs, mode)
      reject_review_request('too much source') if mode == 'review source context limit'

      text = source_review_output(payload, outputs, mode)
      { id: 'resp_source_final', object: 'response', status: 'completed', model: payload.fetch('model'),
        output: response_output(response_content(text, nil), nil), usage: token_usage(nil) }
    end

    def source_review_output(payload, outputs, mode)
      return review_output(payload, 'review follow-up still open') if mode == 'review source a new base file'
      return review_output(payload, 'review follow-up cleared') if mode == 'review source the base after a fix'
      return review_output(payload, 'review follow-up unverified') if mode == 'review source the omitted target-base design'

      source = outputs.map { |item| item.fetch('output') }.join("\n")
      { overview: source_review_text(source), findings: [], complete: mode != 'review source incomplete' }.to_json
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
        'No supported findings within the inspected scope.'
      end
    end
  end
end
