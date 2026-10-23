# frozen_string_literal: true

module Abgleich
  APP_ID = 42
  INSTALLATION_ID = 99
  APP_KEY_PATH = 'secrets/github-app-private-key.pem'
  WEBHOOK_SECRET_PATH = 'secrets/github-webhook-secret'
  WEBHOOK_SECRET_PATHS = {
    'valid' => WEBHOOK_SECRET_PATH,
    'empty' => 'secrets/github-webhook-empty-secret',
    'whitespace' => 'secrets/github-webhook-whitespace-secret'
  }.freeze
  WEBHOOK_SECRET = File.read(WEBHOOK_SECRET_PATH).strip

  class << self
    # Both source types use the same committed webhook fixtures.
    def prepare_github_webhook(content: 'valid', source: 'file')
      path = WEBHOOK_SECRET_PATHS.fetch(content)
      if source == 'file'
        secret_source = "file:#{path}"
      else
        Nonnative.configuration.process_by_name('server').environment = { 'ABGLEICH_TEST_WEBHOOK_SECRET' => File.read(path) }
        secret_source = 'env:ABGLEICH_TEST_WEBHOOK_SECRET'
      end
      config = YAML.safe_load_file('.config/server.yaml')
      config['github']['webhook'] = { 'secret' => secret_source, 'timeout' => '2s' }
      File.write('.config/server.yaml', YAML.dump(config))
    end

    def app_public_key
      @app_public_key ||= OpenSSL::PKey.read(File.read(APP_KEY_PATH)).public_key
    end

    # Select fixtures without modifying them; the missing-key path has no file.
    def prepare_github_key(state)
      path = {
        'valid' => APP_KEY_PATH,
        'unavailable' => 'secrets/github-app-missing-private-key.pem',
        'empty' => 'secrets/github-app-empty-private-key.pem',
        'invalid' => 'secrets/github-app-invalid-private-key.pem',
        'unrecognized' => 'secrets/github-app-unrecognized-private-key.pem'
      }.fetch(state)
      config = YAML.safe_load_file('.config/server.yaml')
      config['github']['app']['private_key'] = "file:#{path}"
      File.write('.config/server.yaml', YAML.dump(config))
    end
  end
end
