# frozen_string_literal: true

module Abgleich
  class GitHubRepository
    # Existing native review conversations maintained independently of inference.
    module Threads
      # Thread state persists across worker restarts, like the remote GitHub discussion.
      def seed_review_thread(name, kind: 'code', state: 'outdated')
        @mutex.synchronize do
          pull = @state[:pulls].values.find { |item| item.dig(:head, :ref) == 'contribution' }
          thread = review_thread_data(name, kind, state, thread_pull_identity(pull))
          (@state[:review_threads] ||= []) << thread
        end
      end

      def review_thread_data(name, kind, state, identity)
        command = { 'code' => 'codereview', 'security' => 'secreview' }.fetch(kind)
        body = state == 'no result marker' ? 'Ordinary review' : "Review overview\n<!-- abgleich:#{command}:v2:#{'a' * 64} -->"
        review = { id: "REVIEW_#{name}", body:, viewerDidAuthor: state != 'another review author',
                   author: { __typename: 'Bot' }, pullRequest: identity }
        root = { id: "COMMENT_#{name}", viewerDidAuthor: state != 'another root author',
                 author: { __typename: state == 'another root author' ? 'User' : 'Bot' }, pullRequestReview: review }
        { id: name, isOutdated: state != 'current', isResolved: state == 'resolved',
          viewerCanResolve: state != 'cannot resolve', pullRequest: identity, comments: { nodes: [root] } }
      end

      def review_threads
        @mutex.synchronize { JSON.parse(@state.fetch(:review_threads, []).to_json, symbolize_names: true) }
      end

      def change_review_thread(name, **changes)
        @mutex.synchronize { @state.fetch(:review_threads).find { |thread| thread[:id] == name }.merge!(changes) }
      end

      def thread_recheck=(change)
        @mutex.synchronize { @state[:thread_recheck] = change }
      end

      def thread_failure=(failure)
        @mutex.synchronize { @state[:thread_failure] = failure }
      end

      def thread_failure
        @mutex.synchronize { @state[:thread_failure] }
      end

      def thread_resolution_requests
        @mutex.synchronize { @state.fetch(:thread_resolution_requests, []).dup }
      end

      def review_thread_page(id, after)
        @mutex.synchronize do
          pull = @state[:pulls].values.find { |item| item[:node_id] == id }
          threads = @state.fetch(:review_threads, []).select { |thread| thread.dig(:pullRequest, :id) == id }
          { data: { node: thread_pull_identity(pull).merge(reviewThreads: thread_connection(threads, after)) } }
        end
      end

      def thread_connection(threads, after)
        offset = after.to_i
        page = { endCursor: (offset + 100).to_s, hasNextPage: offset + 100 < threads.length }
        { nodes: threads.slice(offset, 100), pageInfo: page }
      end

      def review_thread_node(id)
        @mutex.synchronize do
          thread = @state.fetch(:review_threads).find { |item| item[:id] == id }
          if (change = @state[:thread_recheck])
            thread.merge!(change)
          end
          pull = @state[:pulls].values.find { |item| item[:node_id] == thread.dig(:pullRequest, :id) }
          thread[:pullRequest] = thread_pull_identity(pull)
          { data: { node: thread } }
        end
      end

      def resolve_review_thread(id)
        @mutex.synchronize do
          (@state[:thread_resolution_requests] ||= []) << id
          return { errors: [{ message: 'Resolution unavailable' }] } if @state[:thread_failure] == id

          thread = @state.fetch(:review_threads).find { |item| item[:id] == id }
          thread[:isResolved] = true
          { data: { resolveReviewThread: { thread: thread.slice(:id, :isResolved) } } }
        end
      end

      def thread_pull_identity(pull)
        { id: pull.fetch(:node_id), number: pull.fetch(:number), state: pull.fetch(:state).upcase,
          repository: { nameWithOwner: pull.dig(:base, :repo, :full_name) } }
      end
    end
  end
end
