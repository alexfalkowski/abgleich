# frozen_string_literal: true

require 'json'
require 'digest'

require 'abgleich/github/repository/discovery'
require 'abgleich/github/repository/pull_requests'
require 'abgleich/github/repository/rebasing'
require 'abgleich/github/repository/dependabot'
require 'abgleich/github/repository/automation'
require 'abgleich/github/repository/commands'

module Abgleich
  # Shared state for the fake GitHub REST and GraphQL API.
  class GitHubRepository
    DEFAULT_REPOSITORY = 'alexfalkowski/infraops'

    include Discovery
    include PullRequests
    include Rebasing
    include Dependabot
    include Automation
    include Commands

    attr_reader :raced_sha

    def initialize
      @mutex = Mutex.new
      reset
    end

    def reset
      @mutex.synchronize do
        @generation = (@generation || 0) + 1
        @state = { failure: nil, scans: 0, pagination: false, pulls: {}, behind: {}, original: {}, updates: [], details: Hash.new(0),
                   repositories: {}, repository_failures: {}, repository_pagination: false, discovery_complete: false,
                   login: 'alexfalkowski', listed: {}, automation_pagination: false,
                   token_requests: 0, token_lifetime: 3600, comments: {}, replies: [], reply_requests: 0, label_requests: 0 }
        @state[:repositories][DEFAULT_REPOSITORY] = repository_data(DEFAULT_REPOSITORY)
        @raced_sha = nil
      end
    end

    def progress
      @mutex.synchronize do
        { scans: @state[:scans], updates: @state[:updates].length, actions: @state[:pulls].values.sum { |pull| pull[:actions].length } }
      end
    end

    def fail_repository(repository, failure)
      @mutex.synchronize { @state[:repository_failures][repository] = failure }
    end

    def fail_with(failure)
      @mutex.synchronize { @state[:failure] = failure }
    end

    def failure(repository = nil)
      @mutex.synchronize { @state[:repository_failures][repository] || @state[:failure] }
    end

    private

    def discovered?(pull)
      @state[:discovery_complete] && @state[:listed][pull.dig(:base, :repo, :full_name)]
    end

    def mutation_failure(pull, operation = :updatePullRequestBranch)
      failure = @state[:repository_failures][pull.dig(:base, :repo, :full_name)] || @state[:failure]

      case failure
      when 'graphql rate limit' then { errors: [{ type: 'RATE_LIMITED', message: 'Quota exhausted' }] }
      when 'graphql secondary limit' then { errors: [{ message: 'You have exceeded a secondary rate limit' }] }
      when 'partial mutation'
        { data: { operation => { pullRequest: { id: pull[:node_id] } } }, errors: [{ message: 'Mutation failed' }] }
      when 'mismatched mutation' then { data: { operation => { pullRequest: { id: 'another-pull-request' } } } }
      when 'mutation', "conflict:#{pull.dig(:head, :ref)}" then { errors: [{ message: 'Cannot mutate pull request' }] }
      end
    end
  end
end
