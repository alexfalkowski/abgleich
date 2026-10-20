# frozen_string_literal: true

module Abgleich
  class GitHubRepository
    # Guarded rebases and concurrent branch changes.
    module Rebasing
      def update(input)
        @mutex.synchronize do
          pull = @state[:pulls].values.find { |item| item[:node_id] == input['pullRequestId'] }
          return { errors: [{ message: 'Invalid or premature rebase request' }] } unless valid_rebase?(input, pull)

          race(pull) if @state[:failure] == 'concurrent push'
          return { errors: [{ message: 'Head OID changed' }] } unless input['expectedHeadOid'] == pull.dig(:head, :sha)

          mutation_result(input, pull)
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

      def valid_rebase?(input, pull)
        pull && input['updateMethod'] == 'REBASE' && discovered?(pull)
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
  end
end
