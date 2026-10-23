# frozen_string_literal: true

module Abgleich
  class GitHubService < Nonnative::HTTPService
    SECONDARY_LIMIT_URL = 'https://docs.github.com/rest/overview/resources-in-the-rest-api#secondary-rate-limits'

    class << self
      def reset_request_counts
        @request_mutex ||= Mutex.new
        @request_mutex.synchronize do
          @request_counts = { active_scans: 0, peak_scans: 0, mutation_requests: 0, installation_requests: 0,
                              installation_checks: 0, repository_discoveries: 0 }
        end
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

      def record_installation_request
        @request_mutex.synchronize { @request_counts[:installation_requests] += 1 }
      end

      def record_installation_check
        @request_mutex.synchronize { @request_counts[:installation_checks] += 1 }
      end

      def record_repository_discovery
        @request_mutex.synchronize { @request_counts[:repository_discoveries] += 1 }
      end
    end

    reset_request_counts

    before do
      @graphql_payload = JSON.parse(request.body.read) if request.post? && request.path_info == '/graphql'
      mutation = request.put? || (request.path_info.end_with?('/labels') && request.post?) ||
                 @graphql_payload&.fetch('query')&.start_with?('mutation ')
      GitHubService.record_mutation_request if mutation
      content_type :json
      halt 403 if request.user_agent.to_s.start_with?('GitHub-Hookshot/')
      authenticate_request
      if mutation && Abgleich.github.failure == 'one-shot mutation HTTP'
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

        halt 403, { message: 'Resource not accessible by integration' }.to_json if failure == 'permission'
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
      pull[:head][:repo] = nil if failure == 'missing head repository'
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
      payload = @graphql_payload
      if payload.fetch('query').include?('query PullRequestAutomationEvents')
        halt automation_events_response(payload.fetch('variables')).to_json
      end

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
      when :labels then []
      else "#{value}-changed"
      end
    end

    def automation_events_response(input)
      failure = Abgleich.github.failure
      halt 503 if failure == 'history HTTP' || (failure == 'history page' && input['after'])
      halt 200, '{' if failure == 'malformed history'

      Abgleich.github.automation_events(input)
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
end
