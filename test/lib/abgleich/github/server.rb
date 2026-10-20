# frozen_string_literal: true

module Abgleich
  class GitHubServer < Nonnative::HTTPServer
    def initialize(service)
      Abgleich.github
      super(GitHubService.new, service)
    end
  end
end
