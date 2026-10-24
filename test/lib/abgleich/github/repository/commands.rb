# frozen_string_literal: true

module Abgleich
  class GitHubRepository
    # App installation credentials and current comment state at the fake API boundary.
    module Commands
      def token_lifetime=(seconds)
        @mutex.synchronize { @state[:token_lifetime] = seconds }
      end

      def installation_token
        @mutex.synchronize { "nonnative-installation-token-#{@state[:token_requests]}" }
      end

      def create_installation_token
        @mutex.synchronize do
          @state[:token_requests] += 1
          expires_at = (Time.now.utc + @state[:token_lifetime]).iso8601
          @state[:token_renews_at] = Time.iso8601(expires_at) - 60
          { token: "nonnative-installation-token-#{@state[:token_requests]}", expires_at: }
        end
      end

      def token_requests
        @mutex.synchronize { @state[:token_requests] }
      end

      def token_renews_at
        @mutex.synchronize { @state.fetch(:token_renews_at) }
      end

      def label_requests
        @mutex.synchronize { @state[:label_requests] }
      end

      def add_comment(branch, permission: 'write', body: '/ab automerge', created_at: '2026-01-01T00:00:05Z')
        @mutex.synchronize do
          pull = @state[:pulls].values.find { |item| item.dig(:head, :ref) == branch }
          repository = pull.dig(:base, :repo, :full_name)
          comment = command_comment(pull, body, created_at)
          @state[:comments][comment[:id]] = { comment:, permission:, reactions: {} }
          { action: 'created', installation: { id: INSTALLATION_ID }, repository: @state[:repositories].fetch(repository).dup,
            issue: { number: pull[:number], pull_request: {} },
            comment: JSON.parse(comment.to_json) }
        end
      end

      def comment(id)
        @mutex.synchronize { JSON.parse(@state[:comments].fetch(id).fetch(:comment).to_json) }
      end

      def comment_permission
        @mutex.synchronize do
          { permission: @state[:comments].values.last.fetch(:permission), user: { id: 77, login: 'maintainer' } }
        end
      end

      def change_comment(id, field, value)
        @mutex.synchronize { @state[:comments].fetch(id).fetch(:comment)[field] = value }
      end

      def comment_reactions(id)
        @mutex.synchronize { @state[:comments].fetch(id).fetch(:reactions).keys }
      end

      def react_to_comment(id, content)
        @mutex.synchronize do
          reactions = @state[:comments].fetch(id).fetch(:reactions)
          created = !reactions.key?(content)
          reactions[content] ||= { id: (id * 10) + reactions.length + 1, content:,
                                   user: { id: 9000, login: 'abgleich[bot]', type: 'Bot' } }
          [reactions.fetch(content), created]
        end
      end

      def reply_requests
        @mutex.synchronize { @state[:reply_requests] }
      end

      def record_reply_request
        @mutex.synchronize { @state[:reply_requests] += 1 }
      end

      def replies
        @mutex.synchronize { JSON.parse(@state[:replies].to_json, symbolize_names: true) }
      end

      def publish_reply(repository, number, body)
        @mutex.synchronize do
          reply = { id: 10_000 + @state[:replies].length, body:,
                    issue_url: "http://localhost:11001/repos/#{repository}/issues/#{number}" }
          @state[:replies].push(reply).last
        end
      end

      def request_labels(repository, number, labels)
        @mutex.synchronize do
          @state[:label_requests] += 1
          pull = @state[:pulls].fetch([repository, number])
          labels.each do |label|
            next if pull[:labels].any? { |current| current[:name] == label }

            pull[:labels] << { name: label }
            pull[:automation_events] << { event: 'LabeledEvent', label: { name: label }, created_at: Time.now.utc.iso8601 }
          end
          pull[:labels]
        end
      end

      def revoke_comment_request(branch, event = 'UnlabeledEvent', occurred_at: '2026-01-01T00:00:06Z')
        @mutex.synchronize do
          pull = @state[:pulls].values.find { |item| item.dig(:head, :ref) == branch }
          pull[:auto_merge] = nil
          pull[:labels].reject! { |label| label[:name] == 'automerge' }
          pull[:automation_events] << { event:, label: { name: 'automerge' }, created_at: occurred_at }
        end
      end

      private

      def command_comment(pull, body, created_at)
        repository = pull.dig(:base, :repo, :full_name)
        { id: @state[:comments].length + 1, body:, created_at:, updated_at: created_at,
          issue_url: "http://localhost:11001/repos/#{repository}/issues/#{pull[:number]}",
          user: { id: 77, login: 'maintainer', type: 'User' } }
      end
    end
  end
end
