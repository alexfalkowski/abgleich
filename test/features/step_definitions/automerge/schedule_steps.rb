# frozen_string_literal: true

Given('GitHub responds after the schedule interval') do
  Abgleich.github.fail_with('slow list')
end

Given('a check for branch {string} timed out while {string}') do |branch, operation|
  failure = { 'listing pull requests' => 'timeout', 'rebasing the branch' => 'mutation timeout' }.fetch(operation)
  Abgleich.github.add_pull(branch)
  Abgleich.github.fail_with(failure)
  Nonnative.start
  wait_for { Abgleich.job_logs.last }.to include(
    'level' => 'error', 'error' => a_string_including('context deadline exceeded'),
    'overrun' => true, 'interval' => '1s', 'duration' => a_kind_of(String)
  )
  expect_healthy_worker
end

Given('GitHub failures have interrupted successful branch checks') do
  Nonnative.start
  wait_for { Abgleich.job_logs.last }.to include('level' => 'info')
  Abgleich.github.fail_with('list')
  expect_failed_check
  expect_healthy_worker
  failures = Abgleich.job_logs.count { |record| record['level'] == 'error' }
  wait_for { Abgleich.job_logs.count { |record| record['level'] == 'error' } }.to be > failures
end

Given('the worker is waiting for GitHub to check branch {string}') do |branch|
  Abgleich.github.add_pull(branch)
  Abgleich.github.fail_with('timeout')
  Nonnative.start
  wait_for { Abgleich.github.progress.fetch(:scans) }.to be > 0
  Abgleich.github.fail_with(nil)
end

When('the worker restarts') do
  Abgleich.restart
end

Then('the interrupted check should be cancelled') do
  expect(Abgleich.job_logs).to include(
    a_hash_including('level' => 'error', 'error' => a_string_including('context canceled'))
  )
end

Then('the worker should warn about the slow check') do
  wait_for { Abgleich.job_logs.last }.to include(
    'msg' => 'job finished', 'level' => 'warn', 'overrun' => true, 'interval' => '1s', 'duration' => a_kind_of(String)
  )
end

Then('branch checks should not overlap') do
  target = Abgleich.github.progress.fetch(:scans) + 2
  wait_for { Abgleich.github.progress.fetch(:scans) }.to be >= target
  expect(Abgleich::GitHubService.request_counts.fetch(:peak_scans)).to eq(1), 'GitHub received overlapping branch checks'
end
