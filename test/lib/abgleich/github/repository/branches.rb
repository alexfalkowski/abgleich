# frozen_string_literal: true

module Abgleich
  class GitHubRepository
    # Branch and commit snapshots for the fake GitHub API.
    module Branches
      def add_cleanup_branch(branch, days: 31, protected: false, repository: DEFAULT_REPOSITORY)
        @mutex.synchronize do
          sha = Digest::SHA1.hexdigest("#{@generation}:#{repository}:#{branch}")
          @state[:branches][[repository, branch]] = { name: branch, protected:, commit: { sha: } }
          @state[:commits][[repository, sha]] = { sha:, commit: { committer: { date: (Time.now.utc - (days * 86_400)).iso8601 } } }
        end
      end

      def paginate_branches
        @mutex.synchronize { @state[:branch_pages] = true }
      end

      def branches(repository, page)
        @mutex.synchronize do
          result, more = branch_page(repository, page)
          result = JSON.parse(result.to_json, symbolize_names: true)
          result.first&.delete(:protected) if @state[:cleanup_mode] == 'missing protection'
          result = nil if @state[:cleanup_mode] == 'null branches'
          [result, more]
        end
      end

      def cleanup_branch(repository, branch)
        @mutex.synchronize do
          current = @state[:branches].fetch([repository, branch])
          change_cleanup_branch(current)
          result = JSON.parse(current.to_json, symbolize_names: true)
          result.delete(:protected) if @state[:cleanup_mode] == 'missing current protection'
          result[:name] = 'different-branch' if @state[:cleanup_mode] == 'wrong current branch'
          result
        end
      end

      def cleanup_commit(repository, sha)
        @mutex.synchronize do
          result = JSON.parse(@state[:commits].fetch([repository, sha]).to_json, symbolize_names: true)
          change_cleanup_commit(repository, result)
          result
        end
      end

      def advance_cleanup_branch(branch)
        @mutex.synchronize do
          key = [DEFAULT_REPOSITORY, branch]
          sha = 'd' * 40
          @state[:branches].fetch(key)[:commit][:sha] = sha
          @state[:commits][[DEFAULT_REPOSITORY, sha]] = { sha:, commit: { committer: { date: Time.now.utc.iso8601 } } }
        end
      end

      def remaining_cleanup_branches
        @mutex.synchronize { @state[:branches].values.map { |branch| branch[:name] }.sort }
      end

      private

      def branch_page(repository, page)
        branches = @state[:branches].select { |(name, _branch), _snapshot| name == repository }.values
        branches.sort_by! { |branch| branch[:name] }
        change_cleanup_listing(branches, page)
        more = @state[:branch_pages] && page < branches.length
        @state[:branch_listed][repository] = !more
        [@state[:branch_pages] ? branches.slice(page - 1, 1).to_a : branches, more]
      end

      def change_cleanup_listing(branches, page)
        mode = @state[:cleanup_mode]
        return if branches.empty? || !['overlapping branches', 'listing push'].include?(mode)

        branches.insert(1, branches.first)
        branches.first[:commit][:sha] = 'e' * 40 if mode == 'listing push' && page == 2
      end

      def change_cleanup_branch(current)
        case @state[:cleanup_mode]
        when 'protection recheck' then current[:protected] = true
        when 'branch push' then current[:commit][:sha] = 'e' * 40
        when 'new open PR', 'missing PR recheck revision' then @state[:pulls].values.last[:state] = 'open'
        when 'PR recheck unavailable' then @state[:failure] = 'list'
        end
      end

      def change_cleanup_commit(repository, result)
        case @state[:cleanup_mode]
        when 'missing date' then result[:commit][:committer].delete(:date)
        when 'default recheck' then @state[:repositories][repository][:default_branch] = 'old'
        when 'opt-out recheck' then @state[:repositories][repository][:allow_auto_merge] = false
        when 'withdrawn access' then @state[:repository_failures][repository] = 'repository access'
        end
      end
    end
  end
end
