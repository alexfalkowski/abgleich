# frozen_string_literal: true

module Abgleich
  # Response modes and captured requests for the fake OpenAI API.
  class OpenAIResponses
    def initialize
      @mutex = Mutex.new
      reset
    end

    def reset
      @mutex.synchronize do
        @state&.fetch(:response_gate, nil)&.close
        @state = { mode: nil, requests: [], on_request: nil }
      end
    end

    # Hold inference while queued commands and other GitHub work remain available.
    def pause_responses
      @mutex.synchronize { @state[:response_gate] = Queue.new }
    end

    def resume_responses
      gate = @mutex.synchronize { @state.delete(:response_gate) }
      gate&.close
    end

    def mode=(mode)
      @mutex.synchronize { @state[:mode] = mode }
    end

    def mode
      @mutex.synchronize { @state[:mode] }
    end

    def on_request=(handler)
      @mutex.synchronize { @state[:on_request] = handler }
    end

    # Scenario hooks coordinate external changes after source reaches the API.
    def record_request(payload)
      handler = @mutex.synchronize do
        @state[:requests] << payload
        @state[:on_request]
      end
      handler&.call
      gate = @mutex.synchronize { @state[:response_gate] }
      gate&.pop(timeout: 3)
    end

    def requests
      @mutex.synchronize { @state[:requests].dup }
    end
  end
end
