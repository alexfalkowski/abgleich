# frozen_string_literal: true

module Abgleich
  GITHUB_TOKEN_PATH = 'secrets/github-token'

  # Keep the disposable fixture available while scenarios remove or empty its file.
  GITHUB_TOKEN = File.read(GITHUB_TOKEN_PATH).strip.freeze

  class << self
    # Prepare an unavailable, empty, invalid, or valid token. Valid restores the captured fixture.
    def prepare_github_token(state)
      if state == 'unavailable'
        File.delete(GITHUB_TOKEN_PATH)
      else
        token = { 'empty' => '', 'invalid' => 'invalid-nonnative-token', 'valid' => GITHUB_TOKEN }.fetch(state)
        File.write(GITHUB_TOKEN_PATH, token)
      end
    end
  end
end
