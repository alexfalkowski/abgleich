# frozen_string_literal: true

Around do |_scenario, run|
  server = Nonnative.configuration.process_by_name('server')
  command = server.command
  environment = server.environment
  path = Abgleich.server_config_path

  Abgleich.use_config('server')
  server.command = lambda do
    argv = command.call.dup
    argv[argv.index('-config') + 1] = "file:#{Abgleich.server_config_path}"
    argv
  end

  [Abgleich.github, Abgleich.openai].each(&:reset)
  Abgleich::GitHubService.reset_request_counts

  run.call
ensure
  Abgleich.github.resume_pull_listing
  server.command = command
  server.environment = environment
  Abgleich.server_config_path = path

  [Abgleich.github, Abgleich.openai].each(&:reset)
  Abgleich::GitHubService.reset_request_counts
end
