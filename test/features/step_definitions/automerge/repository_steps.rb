# frozen_string_literal: true

Given('GitHub has repositories with stale auto-merge branches:') do |table|
  Abgleich.github.clear_repositories

  table.hashes.each do |row|
    repository = row.fetch('repository')
    options = { archived: row['state'] == 'archived', disabled: row['state'] == 'disabled', private: row['state'] == 'private',
                auto_merge: row['state'] != 'auto-merge disabled' }

    Abgleich.github.add_repository(repository, **options)
    Abgleich.github.add_pull(row.fetch('branch'), repository:)
  end
end

Given('the repositories span multiple pages') do
  Abgleich.github.paginate_repositories
end

Given('GitHub has no repositories') do
  Abgleich.github.clear_repositories
end

Given('the token belongs to account {string}') do |owner|
  Abgleich.github.authenticated_owner = owner
end

Given('a later repository page is unavailable') do
  Abgleich.github.fail_with('discovery page')
end

Given('repository {string} fails with {string}') do |repository, failure|
  Abgleich.github.fail_repository(repository, failure)
end

Given('the worker has successfully checked the existing repositories') do
  Nonnative.start
  expect_successful_check
end

Given('a repository timeout has interrupted a sweep') do
  Nonnative.start
  wait_for { Abgleich.job_logs.last }.to include('error' => a_string_including('context deadline exceeded'))
  expect_healthy_worker
end

When('repository {string} recovers') do |repository|
  Abgleich.github.fail_repository(repository, nil)
end

When('GitHub adds repository {string} with stale auto-merge branch {string}') do |repository, branch|
  Abgleich.github.add_repository(repository)
  Abgleich.github.add_pull(branch, repository:)
end

Then('branch {string} in repository {string} should be rebased') do |branch, repository|
  wait_for { Abgleich.github.rebased?(branch, repository) }.to be(true), "expected #{repository} branch #{branch} to be rebased"
end

Then('the repositories should have these branch outcomes:') do |table|
  wait_for { Abgleich.job_logs }.not_to be_empty

  table.hashes.each do |row|
    repository, branch, outcome = row.values_at('repository', 'branch', 'outcome')
    expected = outcome == 'rebased'
    wait_for { Abgleich.github.rebased?(branch, repository) }.to be(expected), "expected #{repository} branch #{branch} to be #{outcome}"
  end
end

Then('no repositories should have been checked') do
  expect(Abgleich.github.progress.fetch(:scans)).to eq(0), 'expected discovery failure to prevent repository checks'
  expect(Abgleich::GitHubService.request_counts.fetch(:mutation_requests)).to eq(0), 'expected no rebases after discovery failure'
  expect_no_accepted_rebases
end

Then('the failure should identify repository {string}') do |repository|
  wait_for { Abgleich.job_logs }.to include(a_hash_including('error' => a_string_including(repository)))
end

Then('the first check should stop at repository {string} before repository {string}') do |limited, later|
  wait_for { Abgleich.job_logs.first }.to include('error' => a_string_including(limited))
  expect(Abgleich.job_logs.first.fetch('error')).not_to include(later), 'expected throttling to stop the sweep before the later failure'
end

Then('the first check should stop before pull request {int} in repository {string}') do |number, repository|
  wait_for { Abgleich.job_logs.first }.to include('error' => a_string_including(repository))
  error = Abgleich.job_logs.first.fetch('error')
  expect(error).not_to include("pull request ##{number}"), "expected throttling to skip #{repository} pull request ##{number}"
end

Then('accepted rebases should identify their repositories') do
  wait_for { Abgleich.rebase_logs }.to contain_exactly(
    a_hash_including('repository' => 'alexfalkowski/infraops', 'pull_request' => 1, 'branch' => 'feature'),
    a_hash_including('repository' => 'alexfalkowski/private', 'pull_request' => 1, 'branch' => 'feature')
  )
end
