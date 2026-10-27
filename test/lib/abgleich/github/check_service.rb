# frozen_string_literal: true

module Abgleich
  # CI evidence is fetched over the same authenticated GitHub boundary.
  class GitHubService
    get '/repos/:owner/:repository/commits/:sha/status' do
      ci_result(:statuses)
    end

    get '/repos/:owner/:repository/commits/:sha/check-runs' do
      ci_result(:checks)
    end

    private

    def ci_result(kind)
      page = params.fetch('page', '1').to_i
      fail_ci_observation(kind, page)
      result, more = Abgleich.github.ci_response(params.fetch('sha'), kind, page)
      kind == :statuses ? corrupt_ci_statuses(result) : corrupt_ci_checks(result)
      link_ci_page(kind, page) if more
      change_ci_snapshot if kind == :checks
      result.to_json
    end

    def fail_ci_observation(kind, page)
      failure = Abgleich.github.failure
      halt 403 if failure == "#{kind} access"
      halt 503 if failure == "#{kind} page" && page > 1
    end

    def link_ci_page(kind, page)
      endpoint = kind == :statuses ? 'status' : 'check-runs'
      repository = "#{params.fetch('owner')}/#{params.fetch('repository')}"
      url = "http://localhost:11001/github/repos/#{repository}/commits/#{params.fetch('sha')}/#{endpoint}?page=#{page + 1}"
      headers 'Link' => "<#{url}>; rel=\"next\""
    end

    def change_ci_snapshot
      case Abgleich.github.failure
      when 'CI push during observation'
        Abgleich.github.ci_push('feature')
        Abgleich.github.fail_with(nil)
      when 'hold during CI observation'
        Abgleich.github.label_pull('feature', 'hold')
        Abgleich.github.fail_with(nil)
      end
    end

    def corrupt_ci_statuses(result)
      case Abgleich.github.failure
      when 'wrong status revision' then result[:sha] = 'old-sha'
      when 'missing status total' then result.delete(:total_count)
      when 'missing statuses' then result.delete(:statuses)
      when 'incomplete statuses' then result[:total_count] += 1
      end
    end

    def corrupt_ci_checks(result)
      checks = result.fetch(:check_runs)
      case Abgleich.github.failure
      when 'missing checks' then result.delete(:check_runs)
      when 'missing check total' then result.delete(:total_count)
      when 'incomplete checks' then result[:total_count] += 1
      when 'wrong check revision' then checks.first[:head_sha] = 'old-sha'
      when 'duplicate check identity' then checks.last[:id] = checks.first[:id]
      end
    end
  end
end
