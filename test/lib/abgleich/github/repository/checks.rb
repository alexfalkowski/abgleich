# frozen_string_literal: true

module Abgleich
  class GitHubRepository
    # Current-head CI observations, independent of the worker's event payloads.
    module Checks
      def ci_phase(branch, phase)
        @mutex.synchronize do
          sha = @state[:pulls].values.find { |pull| pull.dig(:head, :ref) == branch }.dig(:head, :sha)
          @state[:statuses][sha], @state[:checks][sha] = ci_observations(sha, phase)
        end
      end

      def ci_pages
        @mutex.synchronize { @state[:ci_pages] = true }
      end

      def ci_response(sha, kind, page)
        @mutex.synchronize do
          entries = @state[kind].fetch(sha) { default_ci_observations(sha, kind) }
          entries = JSON.parse(entries.to_json, symbolize_names: true)
          count = entries.length
          more = @state[:ci_pages] && page < count
          entries = entries.slice(page - 1, 1).to_a if @state[:ci_pages]
          result = kind == :statuses ? { sha:, total_count: count, statuses: entries } : { total_count: count, check_runs: entries }
          [result, more]
        end
      end

      def ci_push(branch, current: true)
        @mutex.synchronize do
          pull = @state[:pulls].values.find { |item| item.dig(:head, :ref) == branch }
          sha = Digest::SHA1.hexdigest("ci:#{pull.dig(:head, :sha)}")
          pull[:head][:sha] = sha
          @raced_sha = sha
          @state[:behind][sha] = current ? 0 : 1
          @state[:statuses][sha] = []
        end
      end

      def mark_branch_behind(branch)
        @mutex.synchronize do
          pull = @state[:pulls].values.find { |item| item.dig(:head, :ref) == branch }
          @state[:behind][pull.dig(:head, :sha)] = 1
        end
      end

      private

      # Existing PR fixtures model completed CI; explicit phases override that
      # evidence, and a newly pushed revision has no inherited completion signal.
      def default_ci_observations(sha, kind)
        return [] unless kind == :statuses && @state[:original].value?(sha)

        [{ context: 'ci/workflow', state: 'success' }]
      end

      def ci_observations(sha, phase)
        final = { context: 'ci/workflow', state: 'success' }
        case phase
        when 'not started' then [[], []]
        when 'building' then [[{ context: 'ci/build', state: 'pending' }], []]
        when 'between validation and sync' then [[{ context: 'ci/build', state: 'success' }], []]
        when 'syncing' then [[{ context: 'ci/build', state: 'success' }, { context: 'ci/sync', state: 'pending' }], []]
        else ci_final_status_observations(sha, phase, final)
        end
      end

      def ci_final_status_observations(sha, phase, final)
        case phase
        when 'failed before sync' then [[{ context: 'ci/build', state: 'failure' }], []]
        when 'finished' then [[{ context: 'ci/build', state: 'success' }, { context: 'ci/sync', state: 'success' }, final], []]
        when 'finished with failure' then [[final.merge(state: 'failure')], []]
        when 'finished with error' then [[final.merge(state: 'error')], []]
        when 'final status pending' then [[final.merge(state: 'pending')], []]
        when 'unknown status' then [[final.merge(state: 'unknown')], []]
        else ci_check_observations(sha, phase, final)
        end
      end

      def ci_check_observations(sha, phase, final)
        check = { id: 1, name: 'ci/workflow', head_sha: sha, status: 'completed', conclusion: 'success' }
        case phase
        when 'check queued' then [[], [check.merge(status: 'queued', conclusion: nil)]]
        when 'check running' then [[], [check.merge(status: 'in_progress', conclusion: nil)]]
        when 'other check running' then [[final], [check.merge(name: 'analysis', status: 'in_progress', conclusion: nil)]]
        when 'check missing conclusion' then [[], [check.merge(conclusion: nil)]]
        else ci_completed_check_observations(check, phase, final)
        end
      end

      def ci_completed_check_observations(check, phase, final)
        case phase
        when 'check finished' then [[], [check]]
        when 'check failed' then [[], [check.merge(conclusion: 'failure')]]
        when 'check cancelled' then [[], [check.merge(conclusion: 'cancelled')]]
        when 'multiple checks finished'
          [[{ context: 'ci/build', state: 'success' }, { context: 'ci/sync', state: 'success' }, final],
           [check, check.merge(id: 2, name: 'analysis')]]
        else raise "unknown CI phase: #{phase}"
        end
      end
    end
  end
end
