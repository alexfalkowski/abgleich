# frozen_string_literal: true

require 'json'
require 'digest'

module Abgleich
  class << self
    def github
      @github ||= GitHubRepository.new
    end
  end

  # Fake REST and GraphQL boundary; mutations reject stale expected head OIDs.
  class GitHubRepository
    attr_reader :raced_sha

    def initialize
      @mutex = Mutex.new
      reset
    end

    def reset
      @mutex.synchronize do
        @generation = (@generation || 0) + 1
        @state = { failure: nil, scans: 0, pagination: false, pulls: {}, behind: {}, original: {}, updates: [], details: Hash.new(0) }
        @raced_sha = nil
        Abgleich.prepare_github_token('valid')
      end
    end

    def add_pull(branch, **options)
      @mutex.synchronize do
        sha = Digest::SHA1.hexdigest("#{@generation}:#{branch}")
        number = @state[:pulls].length + 1
        @state[:original][branch] = sha
        @state[:behind][sha] = options[:current] ? 0 : 1
        @state[:pulls][number] = pull_data(branch, number, sha, options)
      end
    end

    def fail_with(failure)
      @mutex.synchronize { @state[:failure] = failure }
    end

    def failure
      @mutex.synchronize { @state[:failure] }
    end

    def paginate
      @mutex.synchronize { @state[:pagination] = true }
    end

    def progress
      @mutex.synchronize { { scans: @state[:scans], updates: @state[:updates].length } }
    end

    def list(page)
      @mutex.synchronize do
        @state[:scans] += 1 if page == 1
        pulls = @state[:pulls].values
        [@state[:pagination] ? pulls.slice(page - 1, 1).to_a : pulls, @state[:pagination] && page < pulls.length]
      end
    end

    def pull(number)
      @mutex.synchronize do
        @state[:details][number] += 1
        result = JSON.parse(@state[:pulls].fetch(number).to_json, symbolize_names: true)
        result[:auto_merge] = nil if disabled?(number)
        result[:node_id] = nil if @state[:failure] == 'missing node ID'
        result
      end
    end

    def behind(sha)
      @mutex.synchronize { @state[:behind].fetch(sha) }
    end

    def update(input)
      @mutex.synchronize do
        pull = @state[:pulls].values.find { |item| item[:node_id] == input['pullRequestId'] }
        return { errors: [{ message: 'Invalid rebase request' }] } unless pull && input['updateMethod'] == 'REBASE'

        race(pull) if @state[:failure] == 'concurrent push'
        return { errors: [{ message: 'Head OID changed' }] } unless input['expectedHeadOid'] == pull.dig(:head, :sha)

        mutation_result(input, pull)
      end
    end

    def unchanged?
      @mutex.synchronize do
        @state[:pulls].values.all? { |pull| @state[:original][pull.dig(:head, :ref)] == pull.dig(:head, :sha) }
      end
    end

    def rebased?(branch)
      @mutex.synchronize { @state[:updates].any? { |input| input[:branch] == branch && input['updateMethod'] == 'REBASE' } }
    end

    def concurrent_head_used?
      @mutex.synchronize { @state[:updates].any? { |input| input['expectedHeadOid'] == raced_sha } }
    end

    private

    def pull_data(branch, number, sha, options)
      { number:, node_id: "PR_#{@generation}_#{number}", state: options.fetch(:state, 'open'), draft: options.fetch(:draft, false),
        auto_merge: options.fetch(:auto_merge, true) ? { merge_method: 'squash' } : nil,
        head: { ref: branch, sha:, repo: { full_name: options[:fork] ? 'someone/infraops' : 'alexfalkowski/infraops' } },
        base: { ref: 'master', sha: 'b' * 40, repo: { full_name: 'alexfalkowski/infraops' } } }
    end

    def disabled?(number)
      @state[:failure] == 'disabled auto-merge' || (@state[:failure] == 'disabled before rebase' && @state[:details][number] > 1)
    end

    def race(pull)
      @raced_sha = Digest::SHA1.hexdigest('concurrent author change')
      pull[:head][:sha] = raced_sha
      @state[:behind][raced_sha] = 1
      @state[:failure] = 'mutation'
    end

    def mutation_result(input, pull)
      failure = @state[:failure]
      if failure == 'partial mutation'
        return { data: { updatePullRequestBranch: { pullRequest: { id: pull[:node_id] } } }, errors: [{ message: 'Rebase failed' }] }
      end
      return { errors: [{ message: 'Cannot rebase branch' }] } if ['mutation', "conflict:#{pull.dig(:head, :ref)}"].include?(failure)

      pull[:head][:sha] = Digest::SHA1.hexdigest("rebased:#{input['expectedHeadOid']}")
      @state[:behind][pull.dig(:head, :sha)] = 0
      @state[:updates] << input.merge(branch: pull.dig(:head, :ref))
      { data: { updatePullRequestBranch: { pullRequest: { id: pull[:node_id] } } } }
    end
  end

  class GitHubService < Nonnative::HTTPService
    class << self
      def reset_request_counts
        @request_mutex ||= Mutex.new
        @request_mutex.synchronize { @request_counts = { active_scans: 0, peak_scans: 0, mutation_requests: 0 } }
      end

      def request_counts
        @request_mutex.synchronize { @request_counts.dup }
      end

      # Retain each scenario's counters while cancelled requests finish after a reset.
      def track_scan
        counts = @request_mutex.synchronize do
          @request_counts[:active_scans] += 1
          @request_counts[:peak_scans] = [@request_counts[:peak_scans], @request_counts[:active_scans]].max
          @request_counts
        end
        yield
      ensure
        @request_mutex.synchronize { counts[:active_scans] -= 1 }
      end

      def record_mutation_request
        @request_mutex.synchronize { @request_counts[:mutation_requests] += 1 }
      end
    end

    reset_request_counts

    before do
      GitHubService.record_mutation_request if request.post? && request.path_info == '/graphql'
      content_type :json
      expected = "Bearer #{GITHUB_TOKEN}"
      halt 401, { message: 'Bad credentials' }.to_json unless request.env['HTTP_AUTHORIZATION'] == expected
      case Abgleich.github.failure
      when 'authentication'
        halt 401, { message: 'Bad credentials' }.to_json
      when 'rate limit'
        headers 'Retry-After' => '1'
        halt 429, { message: 'Rate limit exceeded' }.to_json
      end
    end

    get '/repos/alexfalkowski/infraops/pulls' do
      GitHubService.track_scan do
        pulls, more = Abgleich.github.list(params.fetch('page', '1').to_i)
        failure = Abgleich.github.failure
        if failure == 'temporary list'
          Abgleich.github.fail_with(nil)
          halt 503
        end
        halt 503 if failure == 'list'
        halt 200, '{' if failure == 'malformed response'
        sleep 1.1 if failure == 'slow list'
        sleep 3 if failure == 'timeout'
        headers 'Link' => '<http://localhost:11001/repos/alexfalkowski/infraops/pulls?page=2>; rel="next"' if more
        pulls.to_json
      end
    end

    get '/repos/alexfalkowski/infraops/pulls/:number' do
      failure = Abgleich.github.failure
      halt 503 if failure == 'pull request'
      pull = Abgleich.github.pull(params.fetch('number').to_i)
      if failure&.start_with?('changed:')
        *path, key = failure.delete_prefix('changed:').split('.').map(&:to_sym)
        parent = path.reduce(pull) { |snapshot, field| snapshot.fetch(field) }
        parent[key] = key == :sha ? Digest::SHA1.hexdigest("changed:#{parent.fetch(key)}") : "#{parent.fetch(key)}-changed"
        Abgleich.github.fail_with('list')
      end
      pull.to_json
    end

    get '/repos/alexfalkowski/infraops/compare/:range' do
      failure = Abgleich.github.failure
      halt 503 if failure == 'comparison'
      halt 200, '{}' if failure == 'missing comparison'
      if failure == 'pull request recheck'
        Abgleich.github.fail_with('pull request')
      elsif failure&.start_with?('change:')
        Abgleich.github.fail_with(failure.sub('change:', 'changed:'))
      end
      { behind_by: Abgleich.github.behind(params.fetch('range').split('...').last) }.to_json
    end

    post '/graphql' do
      payload = JSON.parse(request.body.read)
      halt 400 unless payload.fetch('query').include?('updatePullRequestBranch(input: $input)')
      halt 503 if Abgleich.github.failure == 'mutation HTTP'
      halt 200, '{' if Abgleich.github.failure == 'malformed mutation'
      halt 200, { data: { updatePullRequestBranch: nil } }.to_json if Abgleich.github.failure == 'empty mutation'
      sleep 3 if Abgleich.github.failure == 'mutation timeout'
      result = Abgleich.github.update(payload.fetch('variables').fetch('input'))
      if Abgleich.github.failure == 'lost mutation response'
        Abgleich.github.fail_with('list')
        halt 503
      end
      result.to_json
    end
  end

  class GitHubServer < Nonnative::HTTPServer
    def initialize(service)
      Abgleich.github
      super(GitHubService.new, service)
    end
  end
end
