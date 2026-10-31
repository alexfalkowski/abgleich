# frozen_string_literal: true

module Abgleich
  # Independent fake APIs share one listener and the existing Nonnative lifecycle.
  class APIServer < Nonnative::HTTPServer
    def initialize(service)
      super({ '/github' => GitHubService.new, '/openai' => OpenAIService.new, '/modules' => ModuleService.new }, service)
    end
  end
end
