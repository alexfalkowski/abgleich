# frozen_string_literal: true

Given('GitHub temporarily cannot list pull requests') do
  Abgleich.github.fail_with('temporary list')
end

Given('the worker could not rebase branch {string} because {string}') do |branch, reason|
  failure = {
    'pull requests could not be listed' => 'list',
    'the pull request could not be read' => 'pull request',
    'the pull request could not be rechecked' => 'pull request recheck',
    'the token was rejected' => 'authentication',
    'the rate limit was reached' => 'rate limit',
    'the pull request response was malformed' => 'malformed response',
    'the branches could not be compared' => 'comparison',
    'the comparison was incomplete' => 'missing comparison',
    'the pull request identity was missing' => 'missing node ID',
    'the branch update request failed' => 'mutation HTTP',
    'GitHub rejected the rebase' => 'mutation',
    'GitHub reported a partial rebase failure' => 'partial mutation',
    'the rebase response was empty' => 'empty mutation',
    'the rebase response was malformed' => 'malformed mutation'
  }.fetch(reason)
  Abgleich.github.add_pull(branch)
  Abgleich.github.fail_with(failure)
  Nonnative.start
  wait_for { Abgleich.job_logs.first }.to include('level' => 'error', 'error' => a_string_matching(/.+/))
  expect(Abgleich.github.unchanged?).to be(true)
  expect_healthy_worker
end

Given('GitHub rebased branch {string} without confirming it to the worker') do |branch|
  Abgleich.github.add_pull(branch)
  Abgleich.github.fail_with('lost mutation response')
  Nonnative.start
  expect_failed_check
  wait_for { Abgleich.github.rebased?(branch) }.to be(true)
end

When('GitHub becomes available again') do
  Abgleich.github.fail_with(nil)
end

When('the worker restarts after GitHub recovers') do
  Abgleich.restart { Abgleich.github.fail_with(nil) }
end

When('the worker retries the changed branch') do
  Abgleich.github.fail_with(nil)
end

Then('only one rebase request should have been sent') do
  expect(Abgleich::GitHubService.request_counts.fetch(:mutation_requests)).to eq(1), 'GitHub received repeated rebase requests'
end
