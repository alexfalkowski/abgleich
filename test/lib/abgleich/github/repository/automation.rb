# frozen_string_literal: true

module Abgleich
  class GitHubRepository
    # Remote label and auto-merge history, including pagination and revocation.
    module Automation
      def label_pull(branch, label)
        @mutex.synchronize do
          pull = @state[:pulls].values.find { |item| item.dig(:head, :ref) == branch }
          pull[:labels] << { name: label }
          pull[:automation_events] << automation_event('LabeledEvent', 1, label)
        end
      end

      def paginate_automation_events
        @mutex.synchronize { @state[:automation_pagination] = true }
      end

      def automation_events(input)
        @mutex.synchronize do
          pull = @state[:pulls].values.find { |item| item[:node_id] == input['id'] }
          return { errors: [{ message: 'Unknown pull request' }] } unless pull

          automation_response(pull, input)
        end
      end

      def revoke_labeled_auto_merge(branch, condition)
        @mutex.synchronize do
          pull = @state[:pulls].values.find { |item| item.dig(:head, :ref) == branch }
          pull[:auto_merge] = nil
          pull[:automation_events] << automation_event('AutoMergeDisabledEvent', 2)
          pull[:automation_events].concat(renewal_events(condition))
          pull[:head][:sha] = Digest::SHA1.hexdigest('updated fork head') if condition == 'after the contributor pushes'
        end
      end

      private

      def automation_response(pull, input)
        events = JSON.parse(pull.fetch(:automation_events).to_json, symbolize_names: true)
        result = { data: { node: { id: pull[:node_id], timelineItems: automation_page(events, input) } } }
        corrupt_automation_history(result, @state[:failure], input['after'])
        result
      end

      def automation_page(events, input)
        offset = input['after'].to_i
        more = @state[:automation_pagination] && offset + 1 < events.length
        nodes = @state[:automation_pagination] ? events.slice(offset, 1).to_a : events
        { nodes:, pageInfo: { hasNextPage: more, endCursor: (offset + 1).to_s } }
      end

      def renewal_events(condition)
        events = {
          'after the contributor pushes' => [], 'after the maintainer disables it' => [],
          'and the label is reapplied' => [['UnlabeledEvent', 3, 'automerge'], ['LabeledEvent', 4, 'automerge']],
          'at the same time as the label is reapplied' => [['LabeledEvent', 2, 'automerge']],
          'and a different label is applied' => [['LabeledEvent', 4, 'ready']]
        }.fetch(condition)
        events.map { |event, second, label| automation_event(event, second, label) }
      end

      def automation_event(event, second, label = nil)
        result = { event:, created_at: "2026-01-01T00:00:0#{second}Z" }
        result[:label] = { name: label } if label
        result
      end

      def corrupt_automation_history(result, failure, cursor)
        node = result.dig(:data, :node)
        case failure
        when 'history GraphQL'
          result[:errors] = [{ message: 'Cannot read automation events' }]
        when 'history rate limit'
          result[:errors] = [{ type: 'RATE_LIMITED', message: 'Quota exhausted' }]
        when 'mismatched history' then node[:id] = 'another-pull-request'
        else corrupt_history_page(node[:timelineItems], failure, cursor)
        end
      end

      def corrupt_history_page(page, failure, cursor)
        case failure
        when 'incomplete history' then page.delete(:pageInfo)
        when 'missing history events' then page.delete(:nodes)
        when 'invalid history cursor' then page[:pageInfo][:endCursor] = cursor
        when 'empty history' then page[:nodes] = []
        else corrupt_history_event(page[:nodes].first, failure)
        end
      end

      def corrupt_history_event(event, failure)
        case failure
        when 'missing history timestamp' then event.delete(:created_at)
        when 'missing history label' then event.delete(:label)
        when 'unknown history event' then event[:event] = 'UnexpectedEvent'
        end
      end
    end
  end
end
