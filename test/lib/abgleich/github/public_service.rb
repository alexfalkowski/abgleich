# frozen_string_literal: true

module Abgleich
  # Public repository endpoints use the existing fake listener without App auth.
  class GitHubService
    PUBLIC_REVISION = 'a' * 40
    PUBLIC_TREE = 'b' * 40
    PUBLIC_TEXT = "external-chart-context\nnumTrustedHops: 1\n"

    get '/repos/upstream/chart/git/blobs/:sha' do
      content = public_source_content
      sha = Abgleich.github.failure == 'public inconsistent blob' ? 'c' * 40 : public_blob_sha
      { encoding: 'base64', content: Base64.strict_encode64(content), size: content.bytesize, sha: }.to_json
    end

    private

    def authenticate_request
      return authenticate_app if request.path_info.start_with?('/app/')
      return authenticate_public_source if request.path_info.start_with?('/repos/upstream/')

      expected = "token #{Abgleich.github.installation_token}"
      halt 401, { message: 'Bad credentials' }.to_json unless request.env['HTTP_AUTHORIZATION'] == expected
    end

    def authenticate_public_source
      Abgleich.github.record_public_request(path: request.path_info, authorization: request.env['HTTP_AUTHORIZATION'])
      halt 403 if request.env['HTTP_AUTHORIZATION']
    end

    def public_repository_response
      result = { full_name: 'upstream/chart', private: false }
      case Abgleich.github.failure
      when 'public null repository response' then return 'null'
      when 'public empty repository response' then return ''
      when 'public whitespace repository response' then return " \n\t"
      when 'public private repository' then result[:private] = true
      when 'public missing visibility' then result.delete(:private)
      when 'public mismatched repository' then result[:full_name] = 'someone/other'
      end
      result.to_json
    end

    def public_commit_response
      revision = Abgleich.github.failure == 'public moved tag' && params.fetch('sha') == 'v1.0.0' ? 'c' * 40 : PUBLIC_REVISION
      { sha: revision, commit: { tree: { sha: PUBLIC_TREE } } }.to_json
    end

    def public_tree_response
      sha = Abgleich.github.failure == 'public inconsistent tree' ? 'c' * 40 : PUBLIC_TREE
      { sha:, truncated: Abgleich.github.failure == 'public truncated tree', tree: public_tree_entries }.to_json
    end

    def public_tree_entries
      modes = { 'public symlink' => '120000', 'public file beneath symlink' => '120000',
                'public submodule' => '160000', 'public file beneath submodule' => '160000' }
      mode = modes.fetch(Abgleich.github.failure, '100644')
      type = mode == '160000' ? 'commit' : 'blob'
      entry = { path: 'chart/values.yaml', mode:, type:, sha: public_blob_sha }
      if ['public file beneath symlink', 'public file beneath submodule'].include?(Abgleich.github.failure)
        return [entry.merge(path: 'chart')]
      end

      [{ path: 'chart', mode: '040000', type: 'tree', sha: PUBLIC_TREE }, entry]
    end

    def public_source_content
      Abgleich.github.failure == 'public binary' ? "\x00binary" : PUBLIC_TEXT
    end

    def public_blob_sha
      content = public_source_content
      Digest::SHA1.hexdigest("blob #{content.bytesize}\x00#{content}")
    end
  end
end
