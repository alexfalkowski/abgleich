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
    DEFAULT_REPOSITORY = 'alexfalkowski/infraops'

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
                   login: 'alexfalkowski', listed: {} }
        @state[:repositories][DEFAULT_REPOSITORY] = repository_data(DEFAULT_REPOSITORY)
        @raced_sha = nil
        Abgleich.prepare_github_token('valid')
      end
    end

    def add_repository(name, **)
      @mutex.synchronize { @state[:repositories][name] = repository_data(name, **) }
    end

    def clear_repositories
      @mutex.synchronize { @state[:repositories].clear }
    end

    def repository(name)
      @mutex.synchronize { @state[:repositories].fetch(name).dup }
    end

    def authenticated_owner=(owner)
      @mutex.synchronize { @state[:login] = owner }
    end

    def authenticated_owner
      @mutex.synchronize { @state[:login] }
    end

    def paginate_repositories
      @mutex.synchronize { @state[:repository_pagination] = true }
    end

    def paginate
      @mutex.synchronize { @state[:pagination] = true }
    end

    def progress
      @mutex.synchronize do
        { scans: @state[:scans], updates: @state[:updates].length, actions: @state[:pulls].values.sum { |pull| pull[:actions].length } }
      end
    end

    def repositories(page)
      @mutex.synchronize do
        repositories = @state[:repositories].values.sort_by { |repo| repo[:full_name] }
        more = @state[:repository_pagination] && page < repositories.length
        @state[:discovery_complete] = !more
        page_repositories = @state[:repository_pagination] ? repositories.slice(page - 1, 1).to_a : repositories
        [page_repositories.map { |repo| repo.except(:allow_auto_merge, :allow_squash_merge) }, more]
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

    def add_pull(branch, **options)
      @mutex.synchronize do
        repository = options.fetch(:repository, DEFAULT_REPOSITORY)
        sha = Digest::SHA1.hexdigest("#{@generation}:#{repository}:#{branch}")
        number = @state[:pulls].keys.count { |name, _number| name == repository } + 1
        @state[:original][[repository, branch]] = sha
        @state[:behind][sha] = options[:current] ? 0 : 1
        @state[:pulls][[repository, number]] = pull_data(repository, branch, number, sha, options)
      end
    end

    def list(page, repository = DEFAULT_REPOSITORY)
      @mutex.synchronize do
        @state[:scans] += 1 if page == 1
        pulls = @state[:pulls].select { |(name, _number), _pull| name == repository }.values
        more = @state[:pagination] && page < pulls.length
        @state[:listed][repository] = !more
        [@state[:pagination] ? pulls.slice(page - 1, 1).to_a : pulls, more]
      end
    end

    def pull(number, repository = DEFAULT_REPOSITORY)
      @mutex.synchronize do
        key = [repository, number]
        @state[:details][key] += 1
        result = JSON.parse(@state[:pulls].fetch(key).to_json, symbolize_names: true)
        result[:auto_merge] = nil if disabled?(key)
        result[:node_id] = nil if @state[:failure] == 'missing node ID'
        [result, @state[:details][key] > 1]
      end
    end

    def behind(sha)
      @mutex.synchronize { @state[:behind].fetch(sha) }
    end

    def update(input)
      @mutex.synchronize do
        pull = @state[:pulls].values.find { |item| item[:node_id] == input['pullRequestId'] }
        return { errors: [{ message: 'Invalid or premature rebase request' }] } unless valid_rebase?(input, pull)

        race(pull) if @state[:failure] == 'concurrent push'
        return { errors: [{ message: 'Head OID changed' }] } unless input['expectedHeadOid'] == pull.dig(:head, :sha)

        mutation_result(input, pull)
      end
    end

    def enable_auto_merge(input)
      @mutex.synchronize do
        pull = @state[:pulls].values.find { |item| item[:node_id] == input['pullRequestId'] }
        valid_request = pull && input['mergeMethod'] == 'SQUASH' && discovered?(pull)
        return { errors: [{ message: 'Invalid or premature auto-merge request' }] } unless valid_request

        failure = mutation_failure(pull, :enablePullRequestAutoMerge)
        return failure if failure

        pull[:auto_merge] = { merge_method: 'squash' }
        pull[:actions] << 'auto-merge'
        { data: { enablePullRequestAutoMerge: { pullRequest: { id: pull[:node_id] } } } }
      end
    end

    def merge(repository, number, input)
      @mutex.synchronize do
        pull = @state[:pulls].fetch([repository, number])
        failure = merge_rejection(input, pull)
        return failure if failure

        pull.merge!(state: 'closed', merged: true)
        pull[:actions] << 'merge'
        [200, { merged: true, sha: pull.dig(:head, :sha) }]
      end
    end

    def pull_state(branch)
      @mutex.synchronize { @state[:pulls].values.find { |pull| pull.dig(:head, :ref) == branch }.dup }
    end

    def unchanged?
      @mutex.synchronize do
        @state[:pulls].all? do |(repository, _number), pull|
          @state[:original][[repository, pull.dig(:head, :ref)]] == pull.dig(:head, :sha)
        end
      end
    end

    def rebased?(branch, repository = DEFAULT_REPOSITORY)
      @mutex.synchronize do
        @state[:updates].any? do |input|
          input[:branch] == branch && input[:repository] == repository && input['updateMethod'] == 'REBASE'
        end
      end
    end

    def concurrent_head_used?
      @mutex.synchronize { @state[:updates].any? { |input| input['expectedHeadOid'] == raced_sha } }
    end

    private

    def merge_rejection(input, pull)
      race(pull) if @state[:failure] == 'concurrent push'
      return [409, { message: 'Head OID changed' }] unless input['sha'] == pull.dig(:head, :sha)
      return [422, { message: 'Invalid or premature merge request' }] unless input['merge_method'] == 'squash' && discovered?(pull)

      [405, { message: 'Merge requirements not met' }] if mutation_failure(pull) || pull[:mergeable_state] != 'clean'
    end

    def disabled?(number)
      @state[:failure] == 'disabled auto-merge' || (@state[:failure] == 'disabled before rebase' && @state[:details][number] > 1)
    end

    def repository_data(full_name, **options)
      owner, name = full_name.split('/')
      { name:, full_name:, owner: { login: owner }, archived: options.fetch(:archived, false),
        disabled: options.fetch(:disabled, false), private: options.fetch(:private, false),
        allow_auto_merge: options.fetch(:auto_merge, true), allow_squash_merge: options.fetch(:squash_merge, true) }
    end

    def valid_rebase?(input, pull)
      pull && input['updateMethod'] == 'REBASE' && discovered?(pull)
    end

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

    def pull_data(repository, branch, number, sha, options)
      { number:, node_id: "PR_#{@generation}_#{repository}_#{number}",
        state: options.fetch(:state, 'open'), draft: options.fetch(:draft, false), merged: false, actions: [],
        user: { login: options.fetch(:author, 'author') }, mergeable: options.fetch(:mergeable, true),
        mergeable_state: options.fetch(:mergeable_state, 'blocked'),
        auto_merge: options.fetch(:auto_merge, true) ? { merge_method: 'squash' } : nil,
        head: { ref: branch, sha:, repo: { full_name: options[:fork] ? 'someone/infraops' : repository } },
        base: { ref: 'master', sha: 'b' * 40, repo: { full_name: repository } } }
    end

    def race(pull)
      @raced_sha = Digest::SHA1.hexdigest('concurrent author change')
      pull[:head][:sha] = raced_sha
      @state[:behind][raced_sha] = 1
      @state[:failure] = 'mutation'
    end

    def mutation_result(input, pull)
      failure = mutation_failure(pull)
      return failure if failure

      pull[:head][:sha] = Digest::SHA1.hexdigest("rebased:#{input['expectedHeadOid']}")
      @state[:behind][pull.dig(:head, :sha)] = 0
      @state[:updates] << input.merge(branch: pull.dig(:head, :ref), repository: pull.dig(:base, :repo, :full_name))
      { data: { updatePullRequestBranch: { pullRequest: { id: pull[:node_id] } } } }
    end
  end

  class GitHubService < Nonnative::HTTPService
    SECONDARY_LIMIT_URL = 'https://docs.github.com/rest/overview/resources-in-the-rest-api#secondary-rate-limits'

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
      GitHubService.record_mutation_request if (request.post? && request.path_info == '/graphql') || request.put?
      content_type :json
      expected = "Bearer #{GITHUB_TOKEN}"
      halt 401, { message: 'Bad credentials' }.to_json unless request.env['HTTP_AUTHORIZATION'] == expected
      if (request.post? || request.put?) && Abgleich.github.failure == 'one-shot mutation HTTP'
        Abgleich.github.fail_with('list')
        halt 503
      end
      case Abgleich.github.failure
      when 'authentication'
        halt 401, { message: 'Bad credentials' }.to_json
      when 'rate limit'
        headers 'Retry-After' => '1'
        halt 429, { message: 'Rate limit exceeded' }.to_json
      end
    end

    get '/user' do
      { login: Abgleich.github.authenticated_owner }.to_json
    end

    get '/user/repos' do
      halt 400 unless params['affiliation'] == 'owner' && params['visibility'] == 'all'
      page = params.fetch('page', '1').to_i
      halt 503 if Abgleich.github.failure == 'discovery' || (page > 1 && Abgleich.github.failure == 'discovery page')

      repositories, more = Abgleich.github.repositories(page)
      headers 'Link' => "<http://localhost:11001/user/repos?page=#{page + 1}&affiliation=owner&visibility=all>; rel=\"next\"" if more
      repositories.to_json
    end

    get '/repos/:owner/:repository' do
      repository = "#{params.fetch('owner')}/#{params.fetch('repository')}"
      halt 503 if Abgleich.github.failure(repository) == 'repository settings'
      result = Abgleich.github.repository(repository)
      result[:full_name] = 'someone/other' if Abgleich.github.failure(repository) == 'mismatched repository'
      result.to_json
    end

    get '/repos/:owner/:repository/pulls' do
      GitHubService.track_scan do
        repository = "#{params.fetch('owner')}/#{params.fetch('repository')}"
        page = params.fetch('page', '1').to_i
        pulls, more = Abgleich.github.list(page, repository)
        failure = Abgleich.github.failure(repository)
        if failure == 'temporary list'
          Abgleich.github.fail_with(nil)
          halt 503
        end
        halt 503 if failure == 'list'
        if (limit = rate_limit_response(failure))
          code, payload, rate_headers = limit
          headers rate_headers
          halt code, payload.to_json
        end

        halt 403, { message: 'You have exceeded a secondary rate limit' }.to_json if failure == 'secondary rate limit without metadata'

        halt 403, { message: 'Resource not accessible by personal access token' }.to_json if failure == 'permission'
        halt 200, '{' if failure == 'malformed response'
        sleep 1.1 if failure == 'slow list'
        sleep 3 if failure == 'timeout'
        headers 'Link' => "<http://localhost:11001/repos/#{repository}/pulls?page=#{page + 1}>; rel=\"next\"" if more
        pulls.to_json
      end
    end

    get '/repos/:owner/:repository/pulls/:number' do
      failure = Abgleich.github.failure
      halt 503 if failure == 'pull request'
      repository = "#{params.fetch('owner')}/#{params.fetch('repository')}"
      pull, rechecked = Abgleich.github.pull(params.fetch('number').to_i, repository)
      if failure&.start_with?('changed:')
        change_snapshot(pull, failure.delete_prefix('changed:'))
      elsif rechecked && failure&.start_with?('before mutation:')
        change_snapshot(pull, failure.delete_prefix('before mutation:'))
      end
      pull.to_json
    end

    get '/repos/:owner/:repository/compare/:range' do
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
      operation = if payload.fetch('query').include?('enablePullRequestAutoMerge(input: $input)')
                    :enable_auto_merge
                  elsif payload.fetch('query').include?('updatePullRequestBranch(input: $input)')
                    :update
                  else
                    halt 400
                  end
      halt 503 if Abgleich.github.failure == 'mutation HTTP'
      halt 200, '{' if Abgleich.github.failure == 'malformed mutation'
      halt 200, { data: { updatePullRequestBranch: nil } }.to_json if Abgleich.github.failure == 'empty mutation'
      sleep 3 if Abgleich.github.failure == 'mutation timeout'
      result = Abgleich.github.public_send(operation, payload.fetch('variables').fetch('input'))
      if Abgleich.github.failure == 'lost mutation response'
        Abgleich.github.fail_with('list')
        halt 503
      end
      result.to_json
    end

    put '/repos/:owner/:repository/pulls/:number/merge' do
      halt 503 if Abgleich.github.failure == 'mutation HTTP'
      halt 200, '{' if Abgleich.github.failure == 'malformed mutation'
      halt 200, '{}' if Abgleich.github.failure == 'empty mutation'
      halt 200, { merged: false }.to_json if Abgleich.github.failure == 'unconfirmed merge'
      repository = "#{params.fetch('owner')}/#{params.fetch('repository')}"
      code, result = Abgleich.github.merge(repository, params.fetch('number').to_i, JSON.parse(request.body.read))
      if Abgleich.github.failure == 'lost mutation response'
        Abgleich.github.fail_with('list')
        halt 503
      end
      status code
      result.to_json
    end

    private

    def change_snapshot(pull, field_path)
      *path, field = field_path.split('.').map(&:to_sym)
      parent = path.reduce(pull) { |snapshot, name| snapshot.fetch(name) }
      parent[field] = changed_value(field, parent.fetch(field))
      Abgleich.github.fail_with('list')
    end

    def changed_value(field, value)
      case field
      when :draft then true
      when :state then 'closed'
      when :number then value + 1
      when :mergeable_state then 'blocked'
      else "#{value}-changed"
      end
    end

    def rate_limit_response(failure)
      case failure
      when 'rate limit'
        [429, { message: 'Rate limit exceeded' }, { 'Retry-After' => '1' }]
      when 'primary rate limit'
        [403, { message: 'API rate limit exceeded' }, primary_limit_headers]
      when 'secondary rate limit'
        [403, { message: 'Secondary rate limit', documentation_url: SECONDARY_LIMIT_URL }, { 'Retry-After' => '1' }]
      end
    end

    def primary_limit_headers
      { 'X-RateLimit-Limit' => '5000', 'X-RateLimit-Remaining' => '0', 'X-RateLimit-Reset' => (Time.now.to_i + 2).to_s }
    end
  end

  class GitHubServer < Nonnative::HTTPServer
    def initialize(service)
      Abgleich.github
      super(GitHubService.new, service)
    end
  end
end
