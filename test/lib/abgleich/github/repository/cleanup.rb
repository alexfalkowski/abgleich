# frozen_string_literal: true

module Abgleich
  class GitHubRepository
    # Cleanup history, race simulation, and atomic guarded deletions.
    module Cleanup
      def cleanup_mode=(mode)
        @mutex.synchronize { @state[:cleanup_mode] = mode }
      end

      def cleanup_mode
        @mutex.synchronize { @state[:cleanup_mode] }
      end

      def cleanup_pull(branch, state, base: 'master', repository: DEFAULT_REPOSITORY)
        add_pull(branch, state: state == 'open' ? 'open' : 'closed', auto_merge: false, repository:)
        @mutex.synchronize do
          pull = @state[:pulls].values.last
          pull[:base][:ref] = base
          if state == 'merged'
            pull[:merged] = true
            pull[:merged_at] = Time.now.utc.iso8601
          end
        end
      end

      def reopen_cleanup_pull
        @mutex.synchronize { @state[:pulls].values.last[:state] = 'open' }
      end

      def deletions
        @mutex.synchronize { @state[:deletions].dup }
      end

      def delete_branch(input)
        @mutex.synchronize do
          repository = @state[:repositories].values.find { |repo| repo[:node_id] == input['repositoryId'] }
          update = input.fetch('refUpdates').first
          key = [repository.fetch(:full_name), update.fetch('name').delete_prefix('refs/heads/')]
          failure = cleanup_deletion_rejection(repository, update, @state[:branches][key])
          return { errors: [{ message: failure }] } if failure

          finish_cleanup_deletion(input, key)
        end
      end

      private

      def finish_cleanup_deletion(input, key)
        @state[:branches].delete(key)
        @state[:deletions] << key.last
        mutation_id = @state[:cleanup_mode] == 'unconfirmed deletion' ? 'unconfirmed' : input['clientMutationId']
        { data: { updateRefs: { clientMutationId: mutation_id } } }
      end

      def cleanup_deletion_rejection(repository, update, current)
        return 'Invalid discovery or deletion' unless valid_cleanup_discovery?(repository, update)
        return 'Protected branch' unless current && !current[:protected] && current[:name] != repository[:default_branch]

        cleanup_head_rejection(update, current)
      end

      def cleanup_head_rejection(update, current)
        current[:commit][:sha] = 'e' * 40 if @state[:cleanup_mode] == 'concurrent push'
        return 'Head OID changed' unless update['beforeOid'] == current.dig(:commit, :sha)

        'Deletion rejected' if @state[:cleanup_mode] == "reject:#{current[:name]}"
      end

      def valid_cleanup_discovery?(repository, update)
        @state[:discovery_complete] && @state[:branch_listed][repository[:full_name]] && repository[:allow_auto_merge] &&
          update['afterOid'] == '0' * 40 && update['name'].start_with?('refs/heads/')
      end
    end
  end
end
