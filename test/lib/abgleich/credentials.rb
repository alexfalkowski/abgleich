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
      if source == 'environment'
        path = WEBHOOK_SECRET_PATHS.fetch(content)
        Nonnative.configuration.process_by_name('server').environment = { 'ABGLEICH_TEST_WEBHOOK_SECRET' => File.read(path) }
      end

      profiles = { %w[file valid] => 'webhook',
                   %w[file empty] => 'webhook_empty',
                   %w[file whitespace] => 'webhook_whitespace',
                   %w[environment empty] => 'webhook_environment_empty' }

      use_config(profiles.fetch([source, content]))
    end

    def app_public_key
      @app_public_key ||= OpenSSL::PKey.read(File.read(APP_KEY_PATH)).public_key
    end

    # Select fixtures without modifying them; the missing-key path has no file.
    def prepare_github_key(state)
      profiles = {
        'valid' => 'server',
        'unavailable' => 'app_key_unavailable',
        'empty' => 'app_key_empty',
        'invalid' => 'app_key_invalid',
        'unrecognized' => 'app_key_unrecognized'
      }

      use_config(profiles.fetch(state))
    end
  end
end
