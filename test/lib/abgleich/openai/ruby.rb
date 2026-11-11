# frozen_string_literal: true

module Abgleich
  class OpenAIService
    RUBY_SOURCE_PATHS = {
      'review source Ruby missing path' => %w[missing lib/example.rb],
      'review source Ruby gems' => %w[. lib/example.rb lib/example.rb],
      'review source Ruby platform gem' => %w[. lib/example.rb lib/example.rb],
      'review source Ruby large unrelated file' => %w[. lib/example.rb],
      'review source Ruby expansion during listing' => ['.'],
      'review source Ruby repeated archive lookup' => %w[lib/example.rb lib/example.rb]
    }.freeze

    def ruby_source_calls(mode)
      return unless mode.start_with?('review source Ruby')

      gem = mode == 'review source Ruby undeclared gem' ? 'undeclared' : 'example'
      platform = mode == 'review source Ruby platform gem' ? 'x86_64-linux' : 'ruby'
      paths = RUBY_SOURCE_PATHS.fetch(mode, ['lib/example.rb'])
      paths.map do |path|
        ['read_ruby_dependency', { manifest: 'Gemfile.lock', gem:, platform:, path:, start_line: 2, end_line: 2 }]
      end
    end
  end
end
