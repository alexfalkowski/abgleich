# frozen_string_literal: true

module Abgleich
  # Source context assertions share the scenario RSpec context.
  module Assertions
    # Normalize published Markdown only for assertions about its displayed text.
    def review_text(kind)
      wait_for_review(kind).fetch(:body).delete('\\').delete("\u200b")
    end

    # Return every source-tool result exactly as supplied to the model.
    def review_source_results
      Abgleich.openai.requests.last.fetch('input').filter_map do |item|
        item.fetch('output') if item['type'] == 'function_call_output'
      end
    end

    def review_source_markers(scope)
      {
        'repository' => %w[complete-file-context helper-context],
        'helper' => ['helper-context'],
        'dependency' => ['dependency-query-context']
      }.fetch(scope)
    end
  end
end
