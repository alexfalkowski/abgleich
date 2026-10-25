# frozen_string_literal: true

Around do |_scenario, run|
  server_config = File.read('.config/server.yaml')
  server = Nonnative.configuration.process_by_name('server')
  server_environment = server.environment
  Abgleich.github.reset
  Abgleich.openai.reset
  Abgleich::GitHubService.reset_request_counts
  run.call
ensure
  server.environment = server_environment
  File.write('.config/server.yaml', server_config)
  Abgleich.github.reset
  Abgleich.openai.reset
  Abgleich::GitHubService.reset_request_counts
end
