# frozen_string_literal: true

Given('GitHub has a stale auto-merge branch {string}') do |branch|
  Abgleich.github.add_pull(branch)
end

Given('GitHub has stale auto-merge branches:') do |table|
  table.hashes.each do |row|
    Abgleich.github.add_pull(row['branch'])
  end
end

Given('the pull request for branch {string} is {string}') do |branch, condition|
  options = {
    'not set to auto-merge' => { auto_merge: false },
    'a draft' => { draft: true },
    'from a fork' => { fork: true },
    'closed' => { state: 'closed' },
    'up to date' => { current: true }
  }.fetch(condition)
  Abgleich.github.add_pull(branch, **options)
end

Given('auto-merge is disabled before {string}') do |operation|
  failure_mode = { 'comparison' => 'disabled auto-merge', 'rebase' => 'disabled before rebase' }.fetch(operation)
  Abgleich.github.fail_with(failure_mode)
end

Given("the pull request's {string} changes before rebase") do |field|
  field_path = {
    'identity' => 'node_id',
    'head branch' => 'head.ref',
    'head commit' => 'head.sha',
    'base branch' => 'base.ref'
  }.fetch(field)
  Abgleich.github.fail_with("change:#{field_path}")
end

Given('the pull requests span multiple pages') do
  Abgleich.github.paginate
end

Given('GitHub cannot rebase branch {string}') do |branch|
  Abgleich.github.fail_with("conflict:#{branch}")
end

Given("an author's commit interrupted the rebase of branch {string}") do |branch|
  Abgleich.github.add_pull(branch)
  Abgleich.github.fail_with('concurrent push')
  Nonnative.start
  expect_failed_check
  expect(Abgleich.github.rebased?(branch)).to be(false), "expected the concurrent author change to prevent rebasing #{branch}"
end

When('the worker checks auto-merge branches') do
  Nonnative.start
end

Then('branch {string} should be rebased') do |branch|
  wait_for { Abgleich.github.rebased?(branch) }.to be(true), "expected branch #{branch} to be rebased"
end

Then('the branches should have these outcomes:') do |table|
  table.hashes.each do |row|
    branch, outcome = row.values_at('branch', 'outcome')
    should_be_rebased = { 'rebased' => true, 'not rebased' => false }.fetch(outcome)
    wait_for { Abgleich.github.rebased?(branch) }.to be(should_be_rebased), "expected branch #{branch} to be #{outcome}"
  end
end

Then('the worker should report these accepted rebases:') do |table|
  expected = table.hashes.map do |row|
    a_hash_including('level' => 'info', 'pull_request' => Integer(row['pull request']), 'branch' => row['branch'])
  end
  wait_for { Abgleich.rebase_logs }.to contain_exactly(*expected)
end

Then('branch {string} should remain unchanged') do |branch|
  expected_scans = Abgleich.github.progress.fetch(:scans) + 2
  wait_for { Abgleich.github.progress.fetch(:scans) }.to be >= expected_scans, 'expected two more checks to observe unchanged branches'
  expect(Abgleich.github.unchanged?).to be(true), 'expected all branch commits to remain unchanged'
  expect(Abgleich.github.rebased?(branch)).to be(false), "expected branch #{branch} to remain unchanged"
  expect_no_accepted_rebases
end

Then('branch {string} should be rebased only once') do |branch|
  wait_for { Abgleich.github.rebased?(branch) }.to be(true), "expected branch #{branch} to be rebased"
  expected_scans = Abgleich.github.progress.fetch(:scans) + 2
  wait_for { Abgleich.github.progress.fetch(:scans) }.to be >= expected_scans, 'expected two more checks after the rebase'
  expect(Abgleich.github.progress.fetch(:updates)).to eq(1), 'expected later checks to leave the rebased branch alone'
end

Then('the branch check should succeed') do
  expect_successful_check
end

Then('the branch check should fail') do
  expect_failed_check
end

Then('the branch check should fail without affecting worker health') do
  expect_failed_check
  expect_healthy_worker
end

Then('the worker should defer rebasing branch {string}') do |branch|
  wait_for { Abgleich.job_logs.first }.to include(
    'level' => 'error', 'error' => a_string_including('pull request changed before rebase; retry on the next run')
  )
  expect(Abgleich.github.rebased?(branch)).to be(false), "branch #{branch} was rebased despite the changed pull request"
  expect(Abgleich::GitHubService.request_counts.fetch(:mutation_requests)).to eq(0), 'GitHub received a rebase after the failed recheck'
  expect_no_accepted_rebases
end

Then('the worker should remain healthy') do
  expect_healthy_worker
end

Then("the rebase should use the author's latest commit") do
  expect(Abgleich.github.raced_sha).not_to be_nil
  expect(Abgleich.github.concurrent_head_used?).to be(true), "expected the rebase to use the author's latest commit"
end

Then('no branch checks should have failed') do
  expect(Abgleich.job_logs).not_to include(a_hash_including('level' => 'error'))
end
