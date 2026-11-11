# frozen_string_literal: true

module Abgleich
  # Independent public RubyGems endpoint on the existing shared listener.
  class GemService < Nonnative::HTTPService
    before '/api/v2/*' do
      count = Abgleich.github.record_gem_request(request.path_info, request.env['HTTP_AUTHORIZATION'])
      halt 403 if request.env['HTTP_AUTHORIZATION']
      if Abgleich.github.failure == 'Ruby persistent registry failure' ||
         (Abgleich.github.failure == 'Ruby temporary registry failure' && count == 1)
        halt 503
      end
      if Abgleich.github.failure == 'Ruby persistent rate limit' ||
         (Abgleich.github.failure&.start_with?('Ruby rate limit') && count == 1)
        case Abgleich.github.failure
        when 'Ruby rate limit beyond deadline' then headers 'Retry-After' => '60'
        when 'Ruby rate limit date' then headers 'Retry-After' => (Time.now + 2).httpdate
        when 'Ruby rate limit without hint' then nil
        else headers 'Retry-After' => '1'
        end
        halt 429
      end
    end

    get '/api/v2/rubygems/:gem/versions/:version.json' do
      sha = if Abgleich.github.failure == 'Ruby registry checksum mismatch'
              '0' * 64
            else
              Digest::SHA256.hexdigest(gem_content)
            end

      metadata = { name: params.fetch('gem'), version: params.fetch('version'), platform: params.fetch('platform'), sha:,
                   authors: 'Example author', metadata: { 'source_code_uri' => 'https://github.com/example/library' },
                   dependencies: { development: [], runtime: [] } }.to_json
      metadata = (' ' * (33 << 20)) + metadata if Abgleich.github.failure == 'Ruby oversized metadata'
      metadata
    end

    get '/gems/:name.gem' do
      Abgleich.github.record_gem_request(request.path_info, request.env['HTTP_AUTHORIZATION'])
      halt 403 if request.env['HTTP_AUTHORIZATION']
      content_type 'application/octet-stream'
      sleep 3 if Abgleich.github.failure == 'Ruby slow download'
      content = gem_content
      if Abgleich.github.failure == 'Ruby download without a size'
        body(Enumerator.new { |chunks| (0...content.bytesize).step(16_384) { |offset| chunks << content.byteslice(offset, 16_384) } })
      else
        content
      end
    end

    private

    def gem_content
      filename = GitHubService::RUBY_GEM_FIXTURES.fetch(Abgleich.github.failure, 'library.gem')
      content = File.binread("fixtures/#{filename}")
      if ['Ruby oversized download', 'Ruby download without a size'].include?(Abgleich.github.failure)
        content = content.ljust(33 << 20, "\0")
      end
      content
    end
  end
end
