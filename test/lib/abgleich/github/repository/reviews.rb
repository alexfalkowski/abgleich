# frozen_string_literal: true

module Abgleich
  class GitHubRepository
    # Concurrent authorization changes and result history for code review requests.
    module Reviews
      PATCH = "@@ -1,2 +1,2 @@\n package worker\n-func Worker() bool { return true }\n+func Worker() bool { return false }\n"

      def withdraw_review_reply
        @mutex.synchronize do
          reply = @state[:replies].pop
          @state[:comments].delete(reply.fetch(:id))
        end
      end

      def change_review_state(mode)
        return change_summary_state(mode) if ['head changed', 'base changed', 'permission revoked', 'request edited'].include?(mode)

        @mutex.synchronize do
          pull = @state[:pulls].values.find { |item| item.dig(:head, :ref) == 'contribution' }
          change_review_eligibility(pull, mode)
        end
      end

      def change_review_eligibility(pull, mode)
        case mode
        when 'closed' then pull[:state] = 'closed'
        when 'draft' then pull[:draft] = true
        when 'retargeted' then pull[:base][:ref] = 'another-target'
        when 'request deleted' then @state[:failure] = 'review deleted comment'
        when 'missing base revision' then pull[:base].delete(:sha)
        end
      end
    end
  end
end
