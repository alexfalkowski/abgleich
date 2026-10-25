# frozen_string_literal: true

Given('a pull request from {string} for branch {string} is {string}') do |source, branch, condition|
  options = {
    'waiting for checks' => {},
    'ready to merge' => { mergeable_state: 'clean' },
    'behind its base' => { current: false },
    'waiting with auto-merge enabled' => { auto_merge: true },
    'behind with auto-merge enabled' => { auto_merge: true, current: false },
    'a draft' => { draft: true },
    'closed' => { state: 'closed' },
    'of unknown readiness' => { mergeable_state: 'unknown' },
    'missing mergeability' => { mergeable: nil, mergeable_state: 'clean' },
    'conflicting' => { mergeable: false, mergeable_state: 'dirty' },
    'failing checks' => { mergeable_state: 'unstable' }
  }.fetch(condition)
  fork = { 'a fork' => true, 'the repository' => false }.fetch(source)
  Abgleich.github.add_pull(branch, fork:, auto_merge: false, current: true, **options)
end

Given('branch {string} has the {string} label') do |branch, label|
  Abgleich.github.label_pull(branch, label)
end

Given('the labeled operation fails with {string}') do |failure|
  Abgleich.github.fail_with(failure)
end

Given('GitHub rejects automation on branch {string}') do |branch|
  Abgleich.github.fail_with("conflict:#{branch}")
end

Given('its {string} changes before labeled automation') do |field|
  path = {
    'identity' => 'node_id', 'number' => 'number', 'head commit' => 'head.sha', 'head branch' => 'head.ref',
    'base branch' => 'base.ref', 'head repository' => 'head.repo.full_name', 'base repository' => 'base.repo.full_name',
    'state' => 'state', 'draft' => 'draft', 'labels' => 'labels', 'readiness' => 'mergeable_state'
  }.fetch(field)
  Abgleich.github.fail_with("before mutation:#{path}")
end

Given('GitHub disabled auto-merge for branch {string} {string}') do |branch, condition|
  Abgleich.github.revoke_labeled_auto_merge(branch, condition)
end

Given('the automation events span multiple pages') do
  Abgleich.github.paginate_automation_events
end

Given('the worker previously enabled labeled auto-merge before the contributor updated branch {string}') do |branch|
  Abgleich.github.add_pull(branch, fork: true, auto_merge: false, current: true, labels: ['automerge'])
  Nonnative.start
  wait_for { Abgleich.github.pull_state(branch).dig(:auto_merge, :merge_method) }.to eq('squash')
  Nonnative.stop
  Nonnative.clear_pool
  Abgleich.github.revoke_labeled_auto_merge(branch, 'after the contributor pushes')
end

Then('only one merge action should occur for branch {string}') do |branch|
  expected_scans = Abgleich.github.progress.fetch(:scans) + 2
  wait_for { Abgleich.github.progress.fetch(:scans) }.to be >= expected_scans, "expected two more checks after automating #{branch}"
  expect(Abgleich.github.pull_state(branch).fetch(:actions).length).to eq(1), "expected only one merge action for #{branch}"
end

Then('no rebase should be requested') do
  expect(Abgleich.github.progress.fetch(:updates)).to eq(0), 'expected fork branches to remain untouched'
  expect_no_accepted_rebases
end

Then('no merge actions should be accepted') do
  expect(Abgleich.github.progress.fetch(:actions)).to eq(0), 'expected no accepted merge actions'
  expect_no_accepted_merge_actions
end

Then('auto-merge should remain disabled for branch {string}') do |branch|
  wait_for { Abgleich.job_logs('automerge').length }.to be >= 2, 'expected at least two completed checks after auto-merge was disabled'
  expect(Abgleich.github.pull_state(branch).fetch(:auto_merge)).to be_nil, "expected auto-merge to stay disabled for #{branch}"
  expect(Abgleich.github.pull_state(branch).fetch(:merged)).to be(false), "expected #{branch} to remain open"
end
