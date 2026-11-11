# frozen_string_literal: true

module Abgleich
  class GitHubRepository
    def record_gem_request(path, authorization)
      @mutex.synchronize do
        requests = @state[:gem_requests] ||= []
        requests << { path:, authorization: }
        requests.length
      end
    end

    def gem_requests
      @mutex.synchronize { @state.fetch(:gem_requests, []).dup }
    end

    def source_project_patch(path)
      return summary_patch(path) unless path == 'lib/worker.rb'

      "@@ -1,2 +1,2 @@\n # ruby-repository-context\n-def worker = true\n+def worker = false\n"
    end
  end

  class GitHubService
    RUBY_GEM_FIXTURES = { 'Ruby invalid archive' => 'invalid.gem', 'Ruby symlink' => 'symlink.gem',
                          'Ruby hardlink' => 'hardlink.gem', 'Ruby duplicate archive entry' => 'duplicate.gem',
                          'Ruby highly compressed archive' => 'expanded.gem', 'Ruby expansion during listing' => 'expanded.gem',
                          'Ruby oversized requested file' => 'oversized_source.gem', 'Ruby excess data after archive end' => 'trailing.gem',
                          'Ruby corrupt compression trailer' => 'corrupt.gem', 'Ruby repeated archive lookup' => 'repeated_expansion.gem',
                          'Ruby large unrelated file' => 'large.gem' }.freeze
    RUBY_LOCK_CHECKSUMS = RUBY_GEM_FIXTURES.transform_values { |filename| Digest::SHA256.file("fixtures/#{filename}").hexdigest }
                                           .merge('Ruby lock checksum mismatch' => '0' * 64, 'Ruby invalid checksum' => 'invalid').freeze
    RUBY_REGISTRY_CHECKSUM_ONLY = ['Ruby registry checksum only', 'Ruby oversized download',
                                   'Ruby download without a size', 'Ruby oversized metadata'].freeze

    private

    def source_project_paths
      go = %w[main.go worker.go test.go helpers helpers/guard.go go.mod go.sum]
      ruby = %w[lib/worker.rb Gemfile Gemfile.lock features/worker.feature features/step_definitions/worker_steps.rb]
      return ruby if Abgleich.github.file_mode == 'Ruby'

      Abgleich.github.file_mode == 'mixed' ? go + ruby : go
    end

    def source_project_content(filename)
      case filename
      when 'lib/worker.rb', 'features/step_definitions/worker_steps.rb' then "# ruby-repository-context\ndef worker = false\n"
      when 'features/worker.feature' then "Feature: Worker behavior\n  Scenario: Successful work\n    Then work succeeds\n"
      when 'Gemfile' then "source 'https://rubygems.org'\ngem 'example'\n"
      when 'Gemfile.lock' then ruby_lockfile
      else "package worker\n// complete-file-context for #{filename}\nfunc Worker() bool { return false }\n"
      end
    end

    def ruby_lockfile
      lock = File.read('fixtures/Gemfile.lock')
      checksum = RUBY_LOCK_CHECKSUMS[Abgleich.github.failure]
      return lock.gsub(/sha256=[a-f0-9]+/, "sha256=#{checksum}") if checksum
      return lock.split('CHECKSUMS').first if RUBY_REGISTRY_CHECKSUM_ONLY.include?(Abgleich.github.failure)

      case Abgleich.github.failure
      when 'Ruby private registry' then lock.sub('https://rubygems.org/', 'https://private.invalid/')
      when 'Ruby Git dependency' then lock.sub('GEM', 'GIT')
      when 'Ruby duplicate dependency' then lock.sub('    example (1.2.3)', "    example (1.2.3)\n    example (1.2.3)")
      else lock
      end
    end
  end
end
