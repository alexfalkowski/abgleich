# frozen_string_literal: true

module Abgleich
  class GitHubRepository
    # Concurrent authorization changes and result history for code review requests.
    module Reviews
      REVIEW_STATES = { 'COMMENT' => 'COMMENTED', 'APPROVE' => 'APPROVED', 'REQUEST_CHANGES' => 'CHANGES_REQUESTED' }.freeze

      def record_reference_request(version, authorization)
        @mutex.synchronize { (@state[:reference_requests] ||= []) << { version:, authorization: } }
      end

      def reference_requests
        @mutex.synchronize { @state.fetch(:reference_requests, []).dup }
      end

      def record_source_request(path, revision)
        @mutex.synchronize { (@state[:source_requests] ||= []) << { path:, revision: } }
      end

      def source_requests
        @mutex.synchronize { @state.fetch(:source_requests, []).dup }
      end

      def record_module_request(authorization)
        @mutex.synchronize { (@state[:module_requests] ||= []) << authorization }
      end

      def module_requests
        @mutex.synchronize { @state.fetch(:module_requests, []).dup }
      end

      PATCH = "@@ -1,2 +1,2 @@\n package worker\n-func Worker() bool { return true }\n+func Worker() bool { return false }\n"

      SECURITY_PATCH = <<~PATCH
        @@ -1,2 +1,2 @@
         package worker
        -func Lookup(query string) { db.Query("SELECT name FROM users WHERE name = ?", query) }
        +func Lookup(query string) { db.Query("SELECT name FROM users WHERE name = '" + query + "'") }
      PATCH

      SECURITY_SAFE_PATCH = <<~PATCH
        @@ -1,2 +1,2 @@
         package worker
        -func Lookup(query string) { db.Query("SELECT name FROM users WHERE name = '" + query + "'") }
        +func Lookup(query string) { db.Query("SELECT name FROM users WHERE name = ?", query) }
      PATCH

      def native_reviews
        @mutex.synchronize { JSON.parse(@state.fetch(:reviews, []).to_json, symbolize_names: true) }
      end

      def review_results
        replies + native_reviews
      end

      def review_cancellations(kind)
        replies.select { |reply| reply.fetch(:body).start_with?("⏹️ #{kind.capitalize} review canceled") }
      end

      def native_reviews_for(branch)
        number = pull_state(branch).fetch(:number)
        native_reviews.select { |review| review.fetch(:pull_request_url).end_with?("/pulls/#{number}") }
      end

      def publish_review(repository, number, payload)
        @mutex.synchronize do
          reviews = (@state[:reviews] ||= [])
          id = 20_000 + reviews.length
          review = { id:, node_id: "PRR_#{id}", body: payload.fetch('body'), commit_id: payload.fetch('commit_id'),
                     state: REVIEW_STATES.fetch(payload.fetch('event', 'COMMENT')), submitted_at: Time.now.utc.iso8601,
                     pull_request_url: "http://localhost:11001/github/repos/#{repository}/pulls/#{number}",
                     user: { id: 9000, login: 'abgleich[bot]', type: 'Bot' },
                     comments: payload.fetch('comments', []), viewer_did_author: true }
          reviews.push(review).last.dup
        end
      end

      def review_authors(ids)
        reviews = native_reviews
        nodes = ids.map do |id|
          review = reviews.find { |item| item[:node_id] == id }
          { id:, viewer_did_author: review.fetch(:viewer_did_author) }
        end
        { data: { nodes: nodes.reverse } }
      end

      def change_native_review_author(author)
        change_native_review(:viewer_did_author, false)
        change_native_review(:user, { id: 123, login: 'maintainer', type: 'User' }) if author == 'a human'
      end

      def change_native_review(field, value)
        @mutex.synchronize { @state.fetch(:reviews).last[field] = value }
      end

      def withdraw_review_reply
        @mutex.synchronize do
          return @state[:reviews].pop if @state.fetch(:reviews, []).any?

          reply = @state[:replies].pop
          @state[:comments].delete(reply.fetch(:id))
        end
      end

      def change_review_state(mode)
        if ['head changed', 'base changed', 'permission revoked', 'request edited', 'request identity changed'].include?(mode)
          return change_summary_state(mode)
        end

        @mutex.synchronize do
          pull = @state[:pulls].values.find { |item| item.dig(:head, :ref) == 'contribution' }
          change_review_eligibility(pull, mode)
        end
      end

      def change_review_eligibility(pull, mode)
        case mode
        when 'closed', 'merged' then pull.merge!(state: 'closed', merged: mode == 'merged')
        when 'draft' then pull[:draft] = true
        when 'retargeted' then pull[:base][:ref] = 'another-target'
        when 'request deleted' then @state[:failure] = 'review deleted comment'
        when 'missing base revision' then pull[:base].delete(:sha)
        when 'missing head revision' then pull[:head].delete(:sha)
        end
      end
    end
  end
end
