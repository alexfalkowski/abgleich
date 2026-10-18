# frozen_string_literal: true

Given('the GitHub token is {string}') do |state|
  Abgleich.prepare_github_token(state)
end

Given('the worker could not start with a {string} GitHub token') do |state|
  Abgleich.prepare_github_token(state)
  expect { Nonnative.start }.to raise_error(Nonnative::StartError)
end

Given('the worker cannot authenticate when checking branch {string}') do |branch|
  Abgleich.github.add_pull(branch)
  Abgleich.prepare_github_token('invalid')
  Nonnative.start
  expect_failed_check
  expect(Abgleich.github.unchanged?).to be(true)
  expect_healthy_worker
end

When('the worker restarts with valid credentials') do
  Abgleich.restart { Abgleich.prepare_github_token('valid') }
end
