# frozen_string_literal: true

Given('the App private key is {string}') do |state|
  Abgleich.prepare_github_key(state)
end

Given('the webhook secret comes from {string} with {string} content') do |source, content|
  Abgleich.prepare_github_webhook(content:, source:)
end

Given('the worker could not start with a {string} App private key') do |state|
  Abgleich.prepare_github_key(state)
  expect { Nonnative.start }.to raise_error(Nonnative::StartError)
end

Given('the App installation is not configured') do
  config = YAML.safe_load_file('.config/server.yaml')
  config['github'].delete('app')
  File.write('.config/server.yaml', YAML.dump(config))
end

Given('the installation token approaches expiry after one second') do
  Abgleich.github.token_lifetime = 61
end

Given('the worker cannot authenticate when checking branch {string}') do |branch|
  Abgleich.github.add_pull(branch)
  Abgleich.prepare_github_key('unrecognized')
  Nonnative.start
  expect_failed_check
  expect(Abgleich.github.unchanged?).to be(true)
  expect_healthy_worker
end

When('the worker restarts with valid App credentials') do
  Abgleich.restart { Abgleich.prepare_github_key('valid') }
end

Then('GitHub should receive one installation check per sweep and {int} for comment commands') do |command_checks|
  wait_for { Abgleich.job_logs.length }.to be >= 2, 'expected at least two completed sweeps to check installation freshness'
  wait_for do
    counts = Abgleich::GitHubService.request_counts
    counts.fetch(:installation_checks) - counts.fetch(:repository_discoveries)
  end.to eq(command_checks), "expected one installation check per sweep and #{command_checks} for comment commands"
end
