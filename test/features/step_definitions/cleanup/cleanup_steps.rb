# frozen_string_literal: true

Given('branch cleanup is configured') do
  Abgleich.use_config('cleanup')
end

Given('GitHub has cleanup branches:') do |table|
  table.hashes.each do |row|
    branch, state = row.values_at('branch', 'pull request')
    repository = row.fetch('repository', Abgleich::GitHubRepository::DEFAULT_REPOSITORY)
    Abgleich.github.add_repository(repository) if row.key?('repository')
    Abgleich.github.add_cleanup_branch(branch, days: Integer(row.fetch('commit age in days')), repository:)
    Abgleich.github.cleanup_pull(branch, state, repository:) unless state == 'none'
  end
end

Given('the operator uses the default cleanup settings') do
  Abgleich.use_config('cleanup_defaults')
end

Given('branch {string} is protected from cleanup') do |branch|
  Abgleich.github.add_cleanup_branch(branch, protected: true)
end

Given('cleanup excludes branch {string}') do |branch|
  Abgleich.use_config({ 'develop' => 'cleanup_exclude_develop' }.fetch(branch))
end

Given('an open pull request targets branch {string}') do |branch|
  Abgleich.github.cleanup_pull('feature', 'open', base: branch)
end

Given('branch {string} received new commits after merging') do |branch|
  Abgleich.github.advance_cleanup_branch(branch)
end

Given('branch cleanup is not configured') do
  Abgleich.use_config('server')
end

Given('cleanup waits for {int} inactive days') do |days|
  Abgleich.use_config({ 60 => 'cleanup_inactive_60_days' }.fetch(days))
end

Given('the cleanup repository is {string}') do |state|
  options = { 'opted out' => { auto_merge: false }, 'archived' => { archived: true }, 'disabled' => { disabled: true } }.fetch(state)
  Abgleich.github.add_repository(Abgleich::GitHubRepository::DEFAULT_REPOSITORY, **options)
end

Given('the repository has many branches and past pull requests') do
  Abgleich.github.paginate_branches
  Abgleich.github.paginate
end

Given('branch {string} was merged in a later pull request') do |branch|
  Abgleich.github.cleanup_pull(branch, 'merged')
end

Given('GitHub repeats a branch while listing the repository') do
  Abgleich.github.cleanup_mode = 'overlapping branches'
end

Given('GitHub cannot list branches in repository {string}') do |repository|
  Abgleich.github.fail_repository(repository, 'branch list')
end

Given('GitHub cannot finish discovering the cleanup repositories') do
  Abgleich.github.paginate_repositories
  Abgleich.github.fail_with('discovery page')
end

Given('GitHub cannot confirm {string} during cleanup') do |information|
  modes = { 'whether branches are protected' => 'missing protection', 'a branch\'s latest commit' => 'missing date',
            'a branch\'s pull request history' => 'null history', 'the repository\'s identity' => 'missing repository identity',
            'the complete branch listing' => 'null branches', 'the current branch\'s protection' => 'missing current protection',
            'the current branch\'s identity' => 'wrong current branch', 'current pull requests' => 'open PR unavailable',
            'new pull requests before deletion' => 'PR recheck unavailable',
            'a new contribution\'s revision' => 'missing PR recheck revision',
            'a contribution\'s history revision' => 'missing PR history revision',
            'a contribution\'s history branch' => 'wrong PR history branch',
            'an active contribution\'s revision' => 'missing PR revision',
            'an active contribution\'s repository' => 'wrong PR repository',
            'the recorded merge' => 'unconfirmed merge', 'the merged contribution' => 'merged PR unavailable' }
  Abgleich.github.cleanup_mode = modes.fetch(information)
end

Given('a maintainer {string} while cleanup is running') do |change|
  modes = { 'protects the branch' => 'protection recheck', 'makes the branch the default' => 'default recheck',
            'opts the repository out' => 'opt-out recheck', 'withdraws repository access' => 'withdrawn access' }
  Abgleich.github.cleanup_mode = modes.fetch(change)
end

Given('an author pushes new work while cleanup is running') do
  Abgleich.github.cleanup_mode = 'concurrent push'
end

Given('an author reopens their contribution while cleanup is running') do
  Abgleich.github.cleanup_mode = 'reopened PR'
end

Given('an author pushes new work before the branch is rechecked') do
  Abgleich.github.cleanup_mode = 'branch push'
end

Given('an author pushes while GitHub is listing the repository') do
  Abgleich.github.cleanup_mode = 'listing push'
end

Given('the operator\'s cleanup deadline expires while GitHub is slow') do
  Abgleich.use_config('cleanup_deadline')
  Abgleich.github.cleanup_mode = 'slow commit'
end

Given('GitHub takes longer than the request timeout to recheck a branch') do
  Abgleich.use_config('cleanup_request_timeout')
  Abgleich.github.cleanup_mode = 'slow branch'
end

Given('a new pull request opens while cleanup is running') do
  Abgleich.github.cleanup_mode = 'new open PR'
end

Given('GitHub cannot delete branch {string} because {string}') do |branch, reason|
  prefix = { 'the deletion is refused' => 'reject', 'GitHub is unavailable' => 'delete HTTP' }.fetch(reason)
  Abgleich.github.cleanup_mode = "#{prefix}:#{branch}"
end

Given('GitHub cannot retrieve the contribution history for branch {string}') do |branch|
  Abgleich.github.cleanup_mode = "history unavailable:#{branch}"
end

Given('GitHub deleted branch {string} without confirming the deletion') do |branch|
  Abgleich.github.cleanup_mode = 'unconfirmed deletion'
  Nonnative.start
  expect_failed_check('cleanup')
  expect(Abgleich.github.remaining_cleanup_branches).not_to include(branch), "expected #{branch} to have been deleted"
  expect(Abgleich.logs).not_to include(a_hash_including('msg' => 'deleted branch'))
end

Given('cleanup has failed to delete branch {string} repeatedly') do |branch|
  Abgleich.use_config('cleanup_recovery')
  Abgleich.github.cleanup_mode = "reject:#{branch}"
  Nonnative.start
  wait_for do
    Abgleich.job_logs('cleanup').count { |record| record['level'] == 'error' }
  end.to be >= 2, 'expected at least two failed cleanup checks'
  expect_healthy_worker
end

When('the worker runs branch cleanup') do
  Nonnative.start
end

When('GitHub accepts branch deletion again') do
  Abgleich.github.cleanup_mode = nil
end

When('the worker restarts after cleanup recovers') do
  Abgleich.restart { Abgleich.github.cleanup_mode = nil }
end

Then('cleanup should delete exactly {string}') do |branches|
  expected = branches.split(', ')
  wait_for { Abgleich.github.deletions }.to match_array(expected), "expected cleanup to delete #{branches}"
  wait_for do
    Abgleich.logs.select { |record| record['msg'] == 'deleted branch' }.map { |record| record['branch'] }
  end.to match_array(expected)
end

Then('cleanup should preserve exactly {string}') do |branches|
  wait_for { Abgleich.job_logs('cleanup') }.not_to be_empty if File.basename(Abgleich.server_config_path).start_with?('cleanup')
  expect(Abgleich.github.remaining_cleanup_branches).to match_array(branches.split(', ')), "expected cleanup to preserve #{branches}"
end

Then('the cleanup check should succeed') do
  expect_successful_check('cleanup')
end

Then('the cleanup check should report a branch request timeout') do
  expect_failed_check('cleanup')
  expect(Abgleich.job_logs('cleanup').last).to include('error' => a_string_including('github: get branch', 'context deadline exceeded'))
end

Then('cleanup should make no deletion request') do
  expect(Abgleich::GitHubService.request_counts.fetch(:mutation_requests)).to eq(0), 'expected no branch deletion requests'
  expect(Abgleich.logs).not_to include(a_hash_including('msg' => 'deleted branch'))
end

Then('no cleanup job should run') do
  expect_successful_check('automerge')
  expect(Abgleich.job_logs('cleanup')).to be_empty, 'expected cleanup to remain disabled'
end

Then('branch {string} should stay deleted without another deletion request') do |branch|
  expect_successful_check('cleanup')
  expect(Abgleich.github.remaining_cleanup_branches).not_to include(branch), "expected #{branch} to remain deleted"
  expect(Abgleich::GitHubService.request_counts.fetch(:mutation_requests)).to eq(1), "expected no second deletion request for #{branch}"
  expect(Abgleich.logs).not_to include(a_hash_including('msg' => 'deleted branch'))
end
