# frozen_string_literal: true

module Abgleich
  class GitHubService
    SOURCE_BLOB_CHANGES = {
      'source from another revision' => { sha: 'bad-blob' },
      'source binary' => { binary: true, content: nil },
      'source truncated' => { truncated: true },
      'source missing binary metadata' => { binary: nil },
      'source missing truncation metadata' => { truncated: nil },
      'source missing text' => { content: nil },
      'source missing byte size' => { size: nil },
      'source incorrect byte size' => { size: -1 },
      'source not a blob' => { type: 'Commit' }
    }.freeze

    class << self
      def record_source_read
        @request_mutex.synchronize { @request_counts[:source_reads] += 1 }
      end

      def record_source_tree_read
        @request_mutex.synchronize { @request_counts[:source_tree_reads] += 1 }
      end
    end

    get '/repos/:owner/:repository/contents/*' do
      GitHubService.record_source_read
      filename = params.fetch('splat').first
      revision = params.fetch('ref')
      shared_submodule = filename == 'bin' && Abgleich.github.submodule
      excluded_submodule = filename == 'helpers/guard.go' && Abgleich.github.failure == 'source a submodule'
      if shared_submodule || excluded_submodule
        halt 404 unless revision == Abgleich.github.pull_state('contribution').dig(:head, :sha)
        halt submodule_metadata(filename).to_json
      end
      Abgleich.github.record_source_request(filename, revision)
      halt 404 if Abgleich.github.failure == 'source unavailable'
      content = source_content(filename)
      halt 404 if content.nil?
      content = "\x00binary" if Abgleich.github.failure == 'source binary'
      file = { type: 'file', path: filename, encoding: 'base64', content: Base64.strict_encode64(content),
               size: content.bytesize, sha: Digest::SHA1.hexdigest("blob #{content.bytesize}\x00#{content}") }
      file[:sha] = 'bad-blob' if Abgleich.github.failure == 'source from another revision'
      file.to_json
    end

    get '/repos/:owner/:repository/git/trees/:revision' do
      GitHubService.record_source_tree_read
      halt submodule_tree.to_json if params.fetch('repository') == 'bin' && Abgleich.github.submodule
      paths = source_project_paths
      paths << 'bin' if Abgleich.github.submodule
      paths << 'image.png' if Abgleich.github.summary_paths.include?('image.png')
      paths << 'helpers/favicon.png' if Abgleich.github.failure == 'source a binary asset'
      paths.concat((1..source_directory_size).map { |index| "helpers/part#{index}.go" })
      paths << 'go.work' if Abgleich.github.failure == 'source managed by a Go workspace'
      paths << 'vendor/modules.txt' if Abgleich.github.failure == 'source vendored in the repository'
      workspace = {
        'source workspace beside its manifest' => 'tools/app/go.work',
        'source workspace in its parent directory' => 'tools/go.work',
        'source workspace in an unrelated directory' => 'tools/other/go.work'
      }[Abgleich.github.failure]
      paths.concat(%w[tools/app/go.mod tools/app/go.sum] << workspace) if workspace
      paths << 'helpers/empty' if Abgleich.github.failure == 'source an empty directory'
      { sha: 'source-tree', truncated: Abgleich.github.failure == 'source missing part of the repository listing',
        tree: paths.map do |path|
          mode = Abgleich.github.failure == 'source behind a symbolic link' && path == 'helpers/guard.go' ? '120000' : '100644'
          mode = '160000' if path == 'bin' || (Abgleich.github.failure == 'source a submodule' && path == 'helpers/guard.go')
          mode = '040000' if %w[helpers helpers/empty].include?(path)
          type = { '160000' => 'commit', '040000' => 'tree' }.fetch(mode, 'blob')
          content = source_content(path)
          { path:, mode:, type:, sha: path == 'bin' ? 'b' * 40 : source_digest(content), size: content.bytesize }
        end }.to_json
    end

    private

    def submodule_changes(files)
      return files unless ['submodule update', 'submodule update missing patch'].include?(Abgleich.github.file_mode)

      patch = "@@ -1 +1 @@\n-Subproject commit #{'a' * 40}\n+Subproject commit #{'b' * 40}\n"
      patch = nil if Abgleich.github.file_mode == 'submodule update missing patch'
      files.first.merge!(filename: 'bin', sha: 'b' * 40, patch:)
      files
    end

    def submodule_metadata(filename)
      options = Abgleich.github.submodule || { sha: source_digest(source_content(filename)), url: 'https://example.invalid/tooling.git' }
      { type: 'submodule', path: options.fetch(:path, filename), sha: options.fetch(:sha, 'b' * 40),
        submodule_git_url: options.fetch(:url) }
    end

    def submodule_tree
      halt 404 if Abgleich.github.submodule[:unavailable] || params.fetch('revision') != 'b' * 40

      { sha: 'submodule-tree', truncated: false,
        tree: %w[README.md build build/check nested].map do |path|
          next { path:, mode: '040000', type: 'tree', sha: 'build-tree' } if path == 'build'
          next { path:, mode: '160000', type: 'commit', sha: 'c' * 40 } if path == 'nested'

          content = "# shared-tooling-pinned-context for #{path}\n"
          { path:, mode: '100644', type: 'blob', sha: source_digest(content), size: content.bytesize }
        end }
    end

    def source_files_response(variables)
      GitHubService.record_source_read
      failure = Abgleich.github.failure
      halt 503 if failure == 'source unavailable'
      halt 200, '{' if failure == 'source malformed response'
      halt 200, { data: nil }.to_json if failure == 'source missing response data'

      repository = Abgleich.github.repository("#{variables.fetch('owner')}/#{variables.fetch('repository')}")
      data = source_objects(variables, repository)
      return { data:, errors: [{ message: 'Source query failed' }] } if failure == 'source query errors'

      { data: }
    end

    def source_objects(variables, repository)
      variables.filter_map do |name, expression|
        next unless name.start_with?('file')

        revision, filename = expression.split(':', 2)
        Abgleich.github.record_source_request(filename, revision, repository:)
        object = { node_id: repository.fetch(:node_id), full_name: repository.fetch(:full_name), object: source_blob(filename, repository) }
        object[:node_id] = 'R_different' if Abgleich.github.failure == 'source from another repository'
        object = nil if Abgleich.github.failure == 'source missing repository'
        [name, object]
      end.to_h
    end

    def source_blob(filename, repository)
      return nil if Abgleich.github.failure == 'source missing related file' && filename == 'helpers/guard.go'
      return nil if Abgleich.github.failure == 'source missing checksum file' && filename == 'go.sum'

      content = repository.fetch(:name) == 'bin' ? "# shared-tooling-pinned-context for #{filename}\n" : source_content(filename)
      binary = content.include?("\x00")
      { type: 'Blob', content: binary ? nil : content, size: content.bytesize, sha: source_digest(content), binary:, truncated: false }
        .merge(source_blob_changes(content))
    end

    def source_blob_changes(content)
      return { content: "x#{content[1..]}" } if Abgleich.github.failure == 'source digest mismatch'
      return { content: nil } if Abgleich.github.failure == 'source missing empty file text' && content.empty?

      SOURCE_BLOB_CHANGES.fetch(Abgleich.github.failure, {})
    end

    def source_digest(content)
      Digest::SHA1.hexdigest("blob #{content.bytesize}\x00#{content}")
    end

    def source_directory_size
      case Abgleich.github.failure
      when 'source spread across several files' then 3
      when 'source spread across many files' then 105
      else 0
      end
    end

    def source_content(filename)
      return directory_source(filename) if source_directory_size.positive? && filename.start_with?('helpers/part')

      case filename
      when 'image.png', 'helpers/favicon.png' then "\x00binary-asset-context"
      when 'go.mod', 'tools/app/go.mod' then source_manifest
      when 'go.sum', 'tools/app/go.sum' then source_checksum
      when 'helpers/guard.go' then source_helper
      else source_project_content(filename)
      end
    end

    def source_helper
      return '' if ['source empty related file', 'source missing empty file text'].include?(Abgleich.github.failure)
      if Abgleich.github.failure == 'source changed helper'
        return "package worker\n// updated-helper-context\nfunc Allowed() bool { return false }\n"
      end

      content = "package worker\n// helper-context\nfunc Allowed() bool { return true }\n"
      content += "// unrelated-source-padding\n" * 200_000 if Abgleich.github.failure == 'source a large related file'
      content
    end

    def directory_source(filename)
      marker = filename == "helpers/part#{source_directory_size}.go" ? 'directory-tail-context' : 'directory-context'
      "package worker\n// directory-context for #{filename}\n// #{marker}\n"
    end

    def source_checksum
      return "example.com/library v1.0.0 h1:invalid\n" if Abgleich.github.failure == 'source unverified against the dependency checksum'

      File.read('fixtures/library.sum')
    end

    def source_manifest
      manifest = "module example.com/worker\n\ngo 1.22\n\nrequire example.com/library v1.0.0\n"
      return manifest unless Abgleich.github.failure == 'source selected by a version-specific replacement'

      manifest + "\nreplace example.com/library => ../local\n" \
                 "replace example.com/library v1.0.0 => example.com/library v1.0.0\n"
    end
  end
end
