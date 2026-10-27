# frozen_string_literal: true

Given('CI for branch {string} is {string}') do |branch, phase|
  @ci_phase = phase
  Abgleich.github.ci_phase(branch, phase)
end

Given('the CI observations span multiple pages') do
  Abgleich.github.ci_pages
end

Given('an event check left branch {string} waiting for CI') do |branch|
  Abgleich.github.add_pull(branch)
  Abgleich.github.ci_phase(branch, 'building')
  expect(Abgleich.deliver_event('push', Abgleich.github.repository_notification).code).to eq(204)
  wait_for { Abgleich.logs.count { |record| record['msg'] == 'automerge event finished' } }.to eq(1)
  expect_no_automation_mutations
end

Given('an event check left branch {string} waiting after CI failed before sync') do |branch|
  Abgleich.github.add_pull(branch)
  Abgleich.github.ci_phase(branch, 'failed before sync')
  event = Abgleich.github.repository_notification
  event[:state] = 'failure'
  expect(Abgleich.deliver_event('status', event).code).to eq(204)
  wait_for { Abgleich.logs.count { |record| record['msg'] == 'automerge event finished' } }.to eq(1)
  expect_no_automation_mutations
end

Given('CI finished syncing branch {string} before its base advanced') do |branch|
  Abgleich.github.add_pull(branch, current: true)
  Abgleich.github.ci_phase(branch, 'finished')
  Abgleich.github.mark_branch_behind(branch)
end

Given('CI pushed a new stale head for branch {string} without starting checks') do |branch|
  @completed_ci_head = Abgleich.github.pull_state(branch).dig(:head, :sha)
  Abgleich.github.ci_push(branch, current: false)
end

Given('the event completion check is {string}') do |configuration|
  completed = Abgleich.job_logs('automerge').length
  Abgleich.restart do
    config = YAML.safe_load_file('.config/server.yaml')
    case configuration
    when 'omitted' then config.delete('automerge')
    when 'empty' then config['automerge']['completion_check'] = ''
    when 'unreported' then config['automerge']['completion_check'] = 'another/workflow'
    when 'differently cased' then config['automerge']['completion_check'] = 'CI/workflow'
    end
    File.write('.config/server.yaml', YAML.dump(config))
  end
  wait_for { Abgleich.job_logs('automerge').length }.to be > completed, 'expected the startup sweep to finish before adding a new branch'
end

When('GitHub reports the final CI result for branch {string}') do |branch|
  event = Abgleich.github.repository_notification
  event[:sha] = @completed_ci_head || Abgleich.github.pull_state(branch).dig(:head, :sha)
  event[:context] = 'ci/workflow'
  event[:state] = { 'finished with failure' => 'failure', 'finished with error' => 'error' }.fetch(@ci_phase, 'success')
  event[:action] = 'completed'
  kind = @ci_phase&.start_with?('check ') ? 'check_run' : 'status'
  @event_response = Abgleich.deliver_event(kind, event)
end

When('GitHub reports a successful validation job for branch {string}') do |branch|
  event = Abgleich.github.repository_notification
  event.merge!(sha: Abgleich.github.pull_state(branch).dig(:head, :sha), context: 'ci/build', state: 'success')
  @event_response = Abgleich.deliver_event('status', event)
end

When('CI finishes its sync stage for branch {string} and reports completion') do |branch|
  Abgleich.github.ci_phase(branch, 'finished')
  event = Abgleich.github.repository_notification
  event.merge!(sha: Abgleich.github.pull_state(branch).dig(:head, :sha), context: 'ci/workflow', state: 'success')
  @event_response = Abgleich.deliver_event('status', event)
end

When('CI synchronizes and pushes branch {string}') do |branch|
  Abgleich.github.ci_push(branch)
  @event_response = Abgleich.deliver_event('push', Abgleich.github.repository_notification(ref: "refs/heads/#{branch}"))
end

Then("CI's pushed head for branch {string} should remain untouched") do |branch|
  head = Abgleich.github.pull_state(branch).dig(:head, :sha)
  expect(head).to eq(Abgleich.github.raced_sha), "expected the worker to preserve CI's pushed head for #{branch}"
end
