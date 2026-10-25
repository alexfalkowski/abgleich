# frozen_string_literal: true

require 'time'

module Abgleich
  class GitHubRepository
    # Pull request snapshots, pagination, and branch comparisons.
    module PullRequests
      def paginate(overlap: false)
        @mutex.synchronize do
          @state[:pagination] = true
          @state[:overlapping_pages] = overlap
        end
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

      def list(page, repository = DEFAULT_REPOSITORY, **options)
        @mutex.synchronize do
          @state[:scans] += 1 if page == 1
          pulls = filtered_pulls(repository, options)
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

      private

      def filtered_pulls(repository, options)
        pulls = listed_pulls(repository, options.fetch(:sort, 'created'), options.fetch(:direction, 'desc'))
        state, head = options.values_at(:state, :head)
        state ||= 'open'
        pulls.select! { |pull| pull[:state] == state } unless state == 'all'
        pulls.select! { |pull| "#{pull.dig(:head, :repo, :owner, :login)}:#{pull.dig(:head, :ref)}" == head } if head
        pulls
      end

      def listed_pulls(repository, sort, direction)
        pulls = @state[:pulls].select { |(name, _number), _pull| name == repository }.values
        pulls.sort_by! { |pull| pull.fetch(:"#{sort}_at") }
        pulls.reverse! if direction == 'desc'
        pulls.insert(1, pulls.first) if @state[:overlapping_pages] && !pulls.empty?
        pulls
      end

      def disabled?(number)
        @state[:failure] == 'disabled auto-merge' || (@state[:failure] == 'disabled before rebase' && @state[:details][number] > 1)
      end

      def pull_timestamps(number, options)
        created_at = options[:created_at] || Time.at(number).utc.iso8601
        { created_at:, updated_at: options[:updated_at] || created_at }
      end

      def pull_branches(repository, branch, sha, options)
        { head: { ref: branch, sha:, repo: repository_data(options[:fork] ? 'someone/infraops' : repository) },
          base: { ref: 'master', sha: 'b' * 40, repo: repository_data(repository) } }
      end

      def pull_data(repository, branch, number, sha, options)
        { number:, node_id: "PR_#{@generation}_#{repository}_#{number}",
          changed_files: 3, title: 'Improve the worker', body: 'Keep processing predictable.',
          state: options.fetch(:state, 'open'), draft: options.fetch(:draft, false), merged: false, actions: [],
          user: { login: options.fetch(:author, 'author') }, mergeable: options.fetch(:mergeable, true),
          mergeable_state: options.fetch(:mergeable_state, 'blocked'),
          labels: options.fetch(:labels, []).map { |name| { name: } },
          automation_events: options.fetch(:labels, []).map { |name| automation_event('LabeledEvent', 1, name) },
          auto_merge: options.fetch(:auto_merge, true) ? { merge_method: 'squash' } : nil,
          **pull_branches(repository, branch, sha, options) }.merge(pull_timestamps(number, options))
      end
    end
  end
end
