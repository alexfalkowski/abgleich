# frozen_string_literal: true

module Abgleich
  # Pull request listing uses the same authenticated fake API and failure modes.
  class GitHubService
    get '/repos/:owner/:repository/pulls' do
      GitHubService.track_scan do
        repository = "#{params.fetch('owner')}/#{params.fetch('repository')}"
        page = params.fetch('page', '1').to_i
        pulls, more = list_pull_requests(repository, page)
        failure = Abgleich.github.failure(repository)
        if failure == 'temporary list'
          Abgleich.github.fail_with(nil)
          halt 503
        end
        halt 503 if failure == 'list'
        halt 503 if failure == 'pull request page' && page > 1
        if (limit = rate_limit_response(failure))
          code, payload, rate_headers = limit
          headers rate_headers
          halt code, payload.to_json
        end

        halt 403, { message: 'You have exceeded a secondary rate limit' }.to_json if failure == 'secondary rate limit without metadata'

        halt 403, { message: 'Resource not accessible by integration' }.to_json if failure == 'permission'
        halt 200, '{' if failure == 'malformed response'
        sleep 1.1 if failure == 'slow list'
        wait_for_token_renewal(failure)
        sleep 3 if failure == 'timeout'
        headers 'Link' => "<http://localhost:11001/github/repos/#{repository}/pulls?page=#{page + 1}>; rel=\"next\"" if more
        pulls.to_json
      end
    end

    private

    def list_pull_requests(repository, page)
      check_cleanup_pull_listing
      options = { sort: 'created', direction: 'desc', state: 'open' }
      options.merge!(params.to_h.slice('sort', 'direction', 'state', 'head').transform_keys(&:to_sym))
      pulls, more = Abgleich.github.list(page, repository, **options)
      Abgleich.github.wait_for_pull_listing
      [cleanup_pull_snapshots(pulls), more]
    end

    def check_cleanup_pull_listing
      mode = Abgleich.github.cleanup_mode
      case params['state']
      when 'open' then halt 503 if mode == 'open PR unavailable'
      when 'all' then check_cleanup_history(mode)
      end
    end

    def check_cleanup_history(mode)
      halt 200, 'null' if mode == 'null history'
      branch = params.fetch('head').partition(':').last
      halt 503 if mode == "history unavailable:#{branch}"
      Abgleich.github.reopen_cleanup_pull if mode == 'reopened PR'
    end

    def cleanup_pull_snapshots(pulls)
      mode = Abgleich.github.cleanup_mode
      return pulls unless !pulls.empty? && mode&.start_with?('missing PR', 'wrong PR')
      return pulls if mode == 'missing PR recheck revision' && pulls.first[:state] != 'open'

      snapshots = JSON.parse(pulls.to_json, symbolize_names: true)
      change_cleanup_pull_snapshot(snapshots.first, mode)
      snapshots
    end

    def change_cleanup_pull_snapshot(pull, mode)
      pull[:head].delete(:sha) if mode.start_with?('missing PR')
      pull[:base][:repo][:full_name] = 'someone/other' if mode == 'wrong PR repository'
      pull[:head][:ref] = 'different-branch' if mode == 'wrong PR history branch'
    end
  end
end
