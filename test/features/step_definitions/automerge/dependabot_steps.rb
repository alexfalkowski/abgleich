# frozen_string_literal: true

Given('Dependabot automation excludes repositories:') do |table|
  profiles = {
    ['alexfalkowski/infraops'] => 'excluded',
    %w[alexfalkowski/unrelated alexfalkowski/infraops] => 'excluded_with_unrelated',
    %w[alexfalkowski/unrelated ALEXFALKOWSKI/INFRAOPS] => 'excluded_uppercase_with_unrelated',
    ['alexfalkowski/docker'] => 'docker',
    ['someone/infraops'] => 'other_owner',
    ['alexfalkowski/infraops-extra'] => 'longer_name',
    ['alexfalkowski/infra'] => 'shorter_name',
    [] => 'empty',
    [''] => 'invalid'
  }

  context = {
    'server' => 'dependabot',
    'webhook' => 'webhook_dependabot',
    'events_hourly' => 'events_dependabot'
  }

  prefix = context.fetch(File.basename(Abgleich.server_config_path, '.yaml'))
  profile = profiles.fetch(table.hashes.map { |row| row.fetch('repository') })

  Abgleich.use_config("#{prefix}_#{profile}")
end

Given('repository {string} has a ready Dependabot pull request for branch {string}') do |repository, branch|
  Abgleich.github.add_repository(repository)
  Abgleich.github.add_pull(branch, repository:, author: 'dependabot[bot]', auto_merge: false, current: true, mergeable_state: 'clean')
end

Given('the Dependabot pull request for branch {string} is {string}') do |branch, condition|
  options = {
    'waiting for checks' => {},
    'ready to merge' => { mergeable_state: 'clean' },
    'ready with auto-merge enabled' => { mergeable_state: 'clean', auto_merge: true },
    'failing optional checks' => { mergeable_state: 'unstable' },
    'failing optional checks with auto-merge enabled' => { auto_merge: true, mergeable_state: 'unstable' },
    'behind its base' => { current: false },
    'behind with auto-merge enabled' => { current: false, auto_merge: true },
    'waiting with auto-merge enabled' => { auto_merge: true },
    'of unknown readiness with auto-merge' => { auto_merge: true, mergeable_state: 'unknown' },
    'conflicting with auto-merge enabled' => { auto_merge: true, mergeable: false, mergeable_state: 'dirty' },
    'failing required checks with auto-merge enabled' => { auto_merge: true, mergeable_state: 'blocked' },
    'missing mergeability with auto-merge' => { auto_merge: true, mergeable: nil, mergeable_state: 'clean' },
    'a draft' => { draft: true, mergeable_state: 'clean' },
    'closed' => { state: 'closed', mergeable_state: 'clean' },
    'from a fork' => { fork: true, mergeable_state: 'clean' }
  }.fetch(condition)
  Abgleich.github.add_pull(branch, author: 'dependabot[bot]', auto_merge: false, current: true, **options)
end

Given('a ready pull request for branch {string} was opened by {string}') do |branch, author|
  Abgleich.github.add_pull(branch, author:, auto_merge: false, current: true, mergeable_state: 'clean')
end

Given('a ready pull request for branch {string} was opened by {string} with auto-merge enabled') do |branch, author|
  Abgleich.github.add_pull(branch, author:, current: true, mergeable_state: 'clean')
end

Given('the repository has auto-merge disabled') do
  Abgleich.github.add_repository(Abgleich::GitHubRepository::DEFAULT_REPOSITORY, auto_merge: false)
end

Given('the repository has squash merging disabled') do
  Abgleich.github.add_repository(Abgleich::GitHubRepository::DEFAULT_REPOSITORY, squash_merge: false)
end

Given('its {string} changes before Dependabot automation') do |field|
  path = {
    'identity' => 'node_id', 'number' => 'number', 'head commit' => 'head.sha', 'head branch' => 'head.ref',
    'base branch' => 'base.ref', 'head repository' => 'head.repo.full_name', 'base repository' => 'base.repo.full_name',
    'author' => 'user.login', 'state' => 'state', 'draft' => 'draft', 'readiness' => 'mergeable_state'
  }.fetch(field)
  Abgleich.github.fail_with("before mutation:#{path}")
end

Given('the Dependabot operation fails with {string}') do |failure|
  Abgleich.github.fail_with(failure)
end

Given('GitHub rejects Dependabot changes on branch {string}') do |branch|
  Abgleich.github.fail_with("conflict:#{branch}")
end

Given('merging a pull request makes other branches stale') do
  Abgleich.github.fail_with('stale after merge')
end

Given('the worker lost confirmation of a Dependabot action on branch {string} that was {string}') do |branch, condition|
  options = { 'waiting for checks' => {}, 'ready to merge' => { mergeable_state: 'clean' } }.fetch(condition)
  Abgleich.github.add_pull(branch, author: 'dependabot[bot]', auto_merge: false, current: true, **options)
  Abgleich.github.fail_with('lost mutation response')
  Nonnative.start
  expect_failed_check('automerge')
  expect(Abgleich.github.pull_state(branch).fetch(:actions).length).to eq(1), "expected GitHub to accept one action on #{branch}"
  expect_no_accepted_merge_actions
end

Then('branch {string} should have squash auto-merge enabled') do |branch|
  wait_for { Abgleich.github.pull_state(branch).dig(:auto_merge, :merge_method) }.to eq('squash'), "expected squash auto-merge on #{branch}"
end

Then('branch {string} should be squash-merged') do |branch|
  wait_for { Abgleich.github.pull_state(branch).fetch(:merged) }.to be(true), "expected a squash merge for #{branch}"
end

Then('branch {string} should wait for checks before merging') do |branch|
  expected_checks = Abgleich.job_logs('automerge').length + 2
  wait_for do
    Abgleich.job_logs('automerge').length
  end.to be >= expected_checks, "expected two more checks while #{branch} waits for checks after rebasing"
  expect(Abgleich.github.pull_state(branch).fetch(:merged)).to be(false), "expected #{branch} to wait for checks after rebasing"
  expect(Abgleich.logs).not_to include(a_hash_including('msg' => 'merged pull request', 'branch' => branch))
  expect(Abgleich::GitHubService.request_counts.fetch(:mutation_requests)).to eq(2), 'expected only the first merge and second rebase'
end

Then('the worker should report {string} for branch {string}') do |message, branch|
  expected = { 'msg' => message, 'level' => 'info', 'repository' => Abgleich::GitHubRepository::DEFAULT_REPOSITORY,
               'pull_request' => 1, 'branch' => branch }
  wait_for { Abgleich.logs }.to include(a_hash_including(expected))
end

Then('only one Dependabot action should occur for branch {string}') do |branch|
  expected_scans = Abgleich.github.progress.fetch(:scans) + 2
  wait_for { Abgleich.github.progress.fetch(:scans) }.to be >= expected_scans, 'expected two more checks after the Dependabot action'
  expect(Abgleich.github.pull_state(branch).fetch(:actions).length).to eq(1), "expected only one Dependabot action for #{branch}"
end

Then('no pull request mutations should be requested') do
  wait_for { Abgleich.job_logs('automerge').length }.to be >= 2, 'expected at least two completed checks before asserting no mutations'
  expect(Abgleich::GitHubService.request_counts.fetch(:mutation_requests)).to eq(0), 'expected no PR mutation requests'
  expect(Abgleich.github.progress.fetch(:actions)).to eq(0), 'expected no accepted Dependabot actions'
  expect_no_accepted_rebases
end

Then('no Dependabot actions should be accepted') do
  expect(Abgleich.github.progress.fetch(:actions)).to eq(0), 'expected no accepted Dependabot actions'
  expect_no_accepted_merge_actions
end

Then('only one mutation request should have been sent') do
  expect(Abgleich::GitHubService.request_counts.fetch(:mutation_requests)).to eq(1), 'expected no automatic mutation retries'
end

Then('the first check should report only one failure for pull request {int}') do |number|
  wait_for { Abgleich.job_logs('automerge').first }.to include('level' => 'error')
  error = Abgleich.job_logs('automerge').first.fetch('error')
  expect(error.scan("pull request ##{number}:").length).to eq(1), "expected one failed attempt for pull request ##{number}: #{error}"
end

Then('enabling auto-merge and rebasing should occur in separate checks') do
  records = Abgleich.logs
  enabled = records.index { |record| record['msg'] == 'enabled pull request auto-merge' }
  rebased = records.index { |record| record['msg'] == 'requested pull request rebase' }
  checks_between_actions = records[(enabled + 1)...rebased].select { |record| record['job'] == 'automerge' }
  expect(checks_between_actions).not_to be_empty, 'expected a completed check between enabling auto-merge and rebasing'
end
