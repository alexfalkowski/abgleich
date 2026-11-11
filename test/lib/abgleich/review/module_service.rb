# frozen_string_literal: true

module Abgleich
  # Independent public Go module and reference endpoint on the shared listener.
  class ModuleService < Nonnative::HTTPService
    get '/docs/go:version' do
      Abgleich.github.record_reference_request(params.fetch('version'), request.env['HTTP_AUTHORIZATION'])
      halt 403 if request.env['HTTP_AUTHORIZATION']
      halt 503 if Abgleich.github.failure == 'source Go reference unavailable'
      halt 404 unless params.fetch('version') == '1.22'
      content_type 'text/html'
      '<html><body>Go 1.22 official reference fixture: go1.22-reference-marker</body></html>'
    end

    get '/example.com/library/@v/v1.0.0.zip' do
      Abgleich.github.record_module_request(request.env['HTTP_AUTHORIZATION'])
      halt 403 if request.env['HTTP_AUTHORIZATION']
      halt 503 if Abgleich.github.failure == 'source public dependency unavailable'
      halt 429 if Abgleich.github.failure == 'source public dependency rate limited'
      if Abgleich.github.failure == 'source redirected to an unrelated origin' && request.host == 'localhost'
        redirect 'http://127.0.0.1:11001/modules/example.com/library/@v/v1.0.0.zip', 302
      end
      content_type 'application/zip'
      File.binread('fixtures/library.zip')
    end
  end
end
