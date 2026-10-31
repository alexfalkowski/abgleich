# frozen_string_literal: true

require 'securerandom'
require 'json'
require 'yaml'
require 'base64'
require 'openssl'
require 'jwt'

require 'abgleich/assertions'
require 'abgleich/credentials'
require 'abgleich/github/repository'
require 'abgleich/github/service'
require 'abgleich/github/pull_request_service'
require 'abgleich/github/check_service'
require 'abgleich/github/app_service'
require 'abgleich/github/branch_service'
require 'abgleich/openai/responses'
require 'abgleich/openai/service'
require 'abgleich/openai/reviews'
require 'abgleich/api/server'

module Abgleich
  class << self
    attr_accessor :server_config_path

    # Select an existing worker fixture for the next startup or restart.
    def use_config(name)
      path = ".config/#{name}.yaml"
      raise ArgumentError, "Missing worker configuration fixture: #{path}" unless File.file?(path)

      self.server_config_path = path
    end

    # Send a signed delivery to the actual worker HTTP boundary.
    def deliver_comment(event, signature: nil)
      deliver_event('issue_comment', event, signature:)
    end

    # Repository events and comment commands share verification and HTTP limits.
    def deliver_event(kind, event, signature: nil)
      payload = event.is_a?(String) ? event : JSON.generate(event)
      signature ||= "sha256=#{OpenSSL::HMAC.hexdigest('SHA256', WEBHOOK_SECRET, payload)}"
      RestClient::Request.execute(method: :post, url: 'http://localhost:11000/webhooks/github', payload:,
                                  headers: { content_type: :json, user_agent: 'GitHub-Hookshot/nonnative',
                                             'X-GitHub-Event' => kind, 'X-Hub-Signature-256' => signature },
                                  timeout: 5)
    rescue RestClient::ExceptionWithResponse => e
      e.response
    end

    def github
      @github ||= GitHubRepository.new
    end

    def openai
      @openai ||= OpenAIResponses.new
    end

    # Ignore an incomplete final line while the service is writing a log record.
    def logs
      path = Nonnative.configuration.process_by_name('server').log
      Nonnative.log_lines(path, ->(line) { line.start_with?('{') && line.end_with?("\n") }).map do |line|
        JSON.parse(line)
      end
    end

    def job_logs(name)
      logs.select { |record| record['job'] == name }
    end

    def rebase_logs
      logs.select { |record| record['msg'] == 'requested pull request rebase' }
    end

    # Apply changes while stopped, then rebuild Nonnative runners before starting again.
    # Stopped HTTP fakes cannot be reused with their closed server and log handles.
    def restart
      Nonnative.stop
      yield if block_given?
      Nonnative.clear_pool
      Nonnative.start
    end

    def config
      Nonnative::ConfigurationFile.load(server_config_path)
    end

    def health_grpc
      @health_grpc ||= Nonnative.grpc_health(host: 'localhost', port: 12_000, service: 'abgleich.v1.Service')
    end

    def user_agent
      @user_agent ||= Nonnative::Header.grpc_user_agent('Abgleich-ruby-client/1.0 gRPC/1.0')
    end
  end

  module V1
  end
end
