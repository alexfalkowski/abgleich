# frozen_string_literal: true

module Abgleich
  class GitHubRepository
    # Squash auto-merge and immediate merge outcomes.
    module Dependabot
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
          advance_base(repository, pull) if @state[:failure] == 'stale after merge'
          [200, { merged: true, sha: pull.dig(:head, :sha) }]
        end
      end

      private

      def advance_base(repository, pull)
        @state[:pulls].each do |(name, _number), other|
          next unless name == repository && other[:state] == 'open' && other.dig(:base, :ref) == pull.dig(:base, :ref)

          other[:base][:sha] = pull.dig(:head, :sha)
          other[:mergeable_state] = 'blocked'
          @state[:behind][other.dig(:head, :sha)] = 1
        end
      end

      def merge_rejection(input, pull)
        race(pull) if @state[:failure] == 'concurrent push'
        return [409, { message: 'Head OID changed' }] unless input['sha'] == pull.dig(:head, :sha)
        return [422, { message: 'Invalid or premature merge request' }] unless input['merge_method'] == 'squash' && discovered?(pull)

        [405, { message: 'Merge requirements not met' }] if mutation_failure(pull) || pull[:mergeable_state] != 'clean'
      end
    end
  end
end
