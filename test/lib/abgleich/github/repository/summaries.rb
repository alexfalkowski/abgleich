# frozen_string_literal: true

module Abgleich
  class GitHubRepository
    # Comment history, changed files, and concurrent GitHub changes for summaries.
    module Summaries
      def publish_reply(repository, number, body)
        @mutex.synchronize do
          id = 10_000 + @state[:replies].length
          reply = { id:, body:,
                    issue_url: "http://localhost:11001/github/repos/#{repository}/issues/#{number}",
                    html_url: "https://github.com/#{repository}/pull/#{number}#issuecomment-#{id}",
                    user: { id: 9000, login: 'abgleich[bot]', type: 'Bot' }, performed_via_github_app: { id: APP_ID },
                    created_at: Time.now.utc.iso8601, updated_at: Time.now.utc.iso8601 }
          @state[:comments][reply[:id]] = { comment: reply, reactions: {} }
          @state[:replies].push(reply).last
        end
      end

      def issue_comments(repository, number)
        @mutex.synchronize do
          url = "http://localhost:11001/github/repos/#{repository}/issues/#{number}"
          @state[:comments].values.map { |entry| entry[:comment] }.select { |comment| comment[:issue_url] == url }
        end
      end

      def file_mode=(mode)
        @mutex.synchronize do
          @state[:file_mode] = mode
          @state[:pulls].each_value { |pull| pull[:changed_files] = 21 } if mode == 'file budget'
          @state[:pulls].each_value { |pull| pull.delete(:changed_files) } if mode == 'missing file count'
        end
      end

      def file_mode
        @mutex.synchronize { @state[:file_mode] }
      end

      def summary_files
        paths = %w[main.go worker.go test.go]
        paths += (4..21).map { |number| "file#{number}.go" } if file_mode == 'file budget'
        paths.pop if file_mode == 'incomplete file list'
        # An empty list must not substitute for an absent file count.
        paths.clear if file_mode == 'missing file count'
        paths.map do |path|
          file = { filename: path, status: 'modified', additions: 1, deletions: 1, changes: 2,
                   sha: Digest::SHA1.hexdigest(path), patch: summary_patch(path) }
          apply_summary_file_change(file)
        end
      end

      def apply_summary_file_change(file)
        if file[:filename] == 'test.go'
          file.delete(:filename) if file_mode == 'missing file name'
          file[:filename] = 'main.go' if file_mode == 'duplicate file'
        end
        file
      end

      def summary_patch(path)
        case file_mode
        when 'missing patch' then nil
        when 'patch budget' then path == 'test.go' ? 'withheld-source' * 400 : 'x' * 6000
        when 'oversized patch' then 'withheld-source' * 1000
        else "@@ -1,3 +1,3 @@\n package worker\n-old\n+new\n"
        end
      end

      def collection_change=(mode)
        @mutex.synchronize { @state[:collection_change] = mode }
      end

      def apply_collection_change
        mode = @mutex.synchronize { @state.delete(:collection_change) }
        change_summary_state(mode) if mode
      end

      def change_summary_state(mode)
        @mutex.synchronize do
          if ['head changed', 'base changed'].include?(mode)
            pull = @state[:pulls].values.find { |item| item.dig(:head, :ref) == 'contribution' }
            pull.fetch(mode.split.first.to_sym)[:sha] = 'd' * 40
          else
            change_summary_authorization(mode)
          end
        end
      end

      def change_summary_authorization(mode)
        entry = @state[:comments].values.find { |comment| comment[:permission] }
        case mode
        when 'permission revoked' then entry[:permission] = 'read'
        when 'request edited' then entry[:comment][:updated_at] = '2026-01-01T00:00:06Z'
        when 'topic removed' then @state[:repositories].each_value { |repository| repository[:topics] = [] }
        end
      end
    end
  end
end
