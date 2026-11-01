# frozen_string_literal: true

module Abgleich
  class GitHubService
    get '/repos/:owner/:repository/contents/*' do
      filename = params.fetch('splat').first
      revision = params.fetch('ref')
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
      paths = %w[main.go worker.go test.go helpers/guard.go go.mod go.sum]
      paths.concat((1..3).map { |index| "helpers/part#{index}.go" }) if Abgleich.github.failure == 'source spread across several files'
      paths << 'go.work' if Abgleich.github.failure == 'source managed by a Go workspace'
      paths << 'vendor/modules.txt' if Abgleich.github.failure == 'source vendored in the repository'
      workspace = {
        'source workspace beside its manifest' => 'tools/app/go.work',
        'source workspace in its parent directory' => 'tools/go.work',
        'source workspace in an unrelated directory' => 'tools/other/go.work'
      }[Abgleich.github.failure]
      paths.concat(%w[tools/app/go.mod tools/app/go.sum] << workspace) if workspace
      { sha: 'source-tree', truncated: Abgleich.github.failure == 'source missing part of the repository listing',
        tree: paths.map do |path|
          mode = Abgleich.github.failure == 'source behind a symbolic link' && path == 'helpers/guard.go' ? '120000' : '100644'
          { path:, mode:, type: 'blob', sha: Digest::SHA1.hexdigest(path) }
        end }.to_json
    end

    private

    def source_content(filename)
      if Abgleich.github.failure == 'source spread across several files' && filename.start_with?('helpers/part')
        return directory_source(filename)
      end

      case filename
      when 'go.mod', 'tools/app/go.mod' then source_manifest
      when 'go.sum', 'tools/app/go.sum' then source_checksum
      when 'helpers/guard.go' then "package worker\n// helper-context\nfunc Allowed() bool { return true }\n"
      else "package worker\n// complete-file-context for #{filename}\nfunc Worker() bool { return false }\n"
      end
    end

    def directory_source(filename)
      marker = filename == 'helpers/part3.go' ? 'directory-tail-context' : 'directory-context'
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
