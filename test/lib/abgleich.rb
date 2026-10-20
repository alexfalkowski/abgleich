# frozen_string_literal: true

require 'securerandom'
require 'json'
require 'yaml'
require 'base64'

require 'abgleich/assertions'
require 'abgleich/credentials'
require 'abgleich/github/repository'
require 'abgleich/github/service'
require 'abgleich/github/server'

module Abgleich
  class << self
    def github
      @github ||= GitHubRepository.new
    end

    # Ignore an incomplete final line while the service is writing a log record.
    def logs
      path = Nonnative.configuration.process_by_name('server').log
      Nonnative.log_lines(path, ->(line) { line.start_with?('{') && line.end_with?("\n") }).map do |line|
        JSON.parse(line)
      end
    end

    def job_logs
      logs.select { |record| record['job'] == 'automerge' }
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
      @config ||= Nonnative::ConfigurationFile.load('.config/server.yaml')
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
