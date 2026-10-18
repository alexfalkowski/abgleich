# frozen_string_literal: true

Around('@github') do |_scenario, run|
  Abgleich.github.reset
  Abgleich::GitHubService.reset_request_counts
  run.call
ensure
  Abgleich.github.reset
  Abgleich::GitHubService.reset_request_counts
end
