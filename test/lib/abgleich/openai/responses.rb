# frozen_string_literal: true

module Abgleich
  # Response modes and captured requests for the fake OpenAI API.
  class OpenAIResponses
    def initialize
      @mutex = Mutex.new
      reset
    end

    def reset
      @mutex.synchronize { @state = { mode: nil, requests: [], on_request: nil } }
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
    end

    def requests
      @mutex.synchronize { @state[:requests].dup }
    end
  end
end
