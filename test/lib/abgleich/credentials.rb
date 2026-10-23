# frozen_string_literal: true

module Abgleich
  APP_ID = 42
  INSTALLATION_ID = 99
  APP_KEY_PATH = 'reports/github-app-private-key.pem'
  WEBHOOK_SECRET_PATH = 'reports/github-webhook-secret'
  WEBHOOK_SECRET = 'disposable-nonnative-webhook-secret'

  class << self
    # Every scenario uses disposable App credentials through the server config.
    def prepare_github_app
      @app_key ||= OpenSSL::PKey::RSA.generate(2048)
      File.write(APP_KEY_PATH, @app_key.to_pem, mode: 'w', perm: 0o600)
    end

    def prepare_github_webhook(secret: WEBHOOK_SECRET, source: 'file')
      if source == 'file'
        File.write(WEBHOOK_SECRET_PATH, secret, mode: 'w', perm: 0o600)
        secret_source = "file:#{WEBHOOK_SECRET_PATH}"
      else
        Nonnative.configuration.process_by_name('server').environment = { 'ABGLEICH_TEST_WEBHOOK_SECRET' => secret }
        secret_source = 'env:ABGLEICH_TEST_WEBHOOK_SECRET'
      end
      config = YAML.safe_load_file('.config/server.yaml')
      config['github']['webhook'] = { 'secret' => secret_source, 'timeout' => '2s' }
      File.write('.config/server.yaml', YAML.dump(config))
    end

    def app_public_key
      @app_key.public_key
    end

    # A different valid RSA key starts successfully but cannot authenticate.
    def prepare_github_key(state)
      return File.delete(APP_KEY_PATH) if state == 'unavailable'
      return prepare_github_app if state == 'valid'

      @unrecognized_key ||= OpenSSL::PKey::RSA.generate(2048)
      key = { 'empty' => '', 'invalid' => 'invalid-disposable-key', 'unrecognized' => @unrecognized_key.to_pem }.fetch(state)
      File.write(APP_KEY_PATH, key)
    end
  end
end
