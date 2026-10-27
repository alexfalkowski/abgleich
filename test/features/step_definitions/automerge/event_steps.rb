# frozen_string_literal: true

Given('GitHub event checks are configured between hourly sweeps') do
  Abgleich.prepare_github_webhook
  config = YAML.safe_load_file('.config/server.yaml')
  config['scheduler']['interval'] = '1h'
  config['automerge'] = { 'completion_check' => 'ci/workflow' }
  File.write('.config/server.yaml', YAML.dump(config))
end

Given('the repository notification has {string}') do |problem|
  @repository_event = Abgleich.github.repository_notification
  case problem
  when 'an invalid signature' then @event_signature = 'sha256=invalid'
  when 'a missing signature' then @event_signature = ''
  when 'another installation' then @repository_event[:installation][:id] += 1
  when 'no installation' then @repository_event.delete(:installation)
  when 'another owner' then @repository_event[:repository][:owner][:login] = 'someone'
  when 'no repository' then @repository_event.delete(:repository)
  when 'inconsistent repository' then @repository_event[:repository][:full_name] = 'alexfalkowski/other'
  when 'an invalid JSON body' then @repository_event = '{'
  end
end

Given('the repository notification uses uppercase names') do
  @repository_event = Abgleich.github.repository_notification
  repository = @repository_event[:repository]
  repository[:owner][:login].upcase!
  repository[:name].upcase!
  repository[:full_name].upcase!
end

Given('the event repository is now {string}') do |condition|
  repository = Abgleich::GitHubRepository::DEFAULT_REPOSITORY
  case condition
  when 'on hold' then Abgleich.github.label_pull('feature', 'hold')
  when 'opted out' then Abgleich.github.add_repository(repository, auto_merge: false)
  when 'archived' then Abgleich.github.add_repository(repository, archived: true)
  when 'disabled' then Abgleich.github.add_repository(repository, disabled: true)
  when 'no longer installed' then Abgleich.github.clear_repositories
  when 'suspended' then Abgleich.github.fail_with('suspended installation')
  end
end

Given('a scheduled sweep is waiting for its repository snapshot') do
  scans = Abgleich.github.progress.fetch(:scans)
  Abgleich.restart { Abgleich.github.pause_pull_listing }
  wait_for { Abgleich.github.progress.fetch(:scans) }.to be > scans, 'expected the startup sweep to be waiting for GitHub'
end

Given('an event check is waiting for its repository snapshot') do
  scans = Abgleich.github.progress.fetch(:scans)
  Abgleich.github.pause_pull_listing
  @event_response = Abgleich.deliver_event('push', Abgleich.github.repository_notification)
  wait_for { Abgleich.github.progress.fetch(:scans) }.to be > scans, 'expected the event check to be waiting for GitHub'
end

Given('event checks and sweeps are both enabled on the regular schedule') do
  completed = Abgleich.job_logs('automerge').length
  Abgleich.restart do
    config = YAML.safe_load_file('.config/server.yaml')
    config['scheduler']['interval'] = '1s'
    File.write('.config/server.yaml', YAML.dump(config))
  end
  wait_for { Abgleich.job_logs('automerge').length }.to be > completed, 'expected the regular schedule to finish its startup sweep'
  expect_successful_check('automerge')
end

Given('the event check for branch {string} failed') do |branch|
  step 'event checks and sweeps are both enabled on the regular schedule'
  Abgleich.github.add_pull(branch)
  Abgleich.github.fail_with('list')
  @event_response = Abgleich.deliver_event('push', Abgleich.github.repository_notification)
  wait_for { Abgleich.logs }.to include(a_hash_including('msg' => 'automerge event finished', 'level' => 'error'))
end

Given('an event check for branch {string} is waiting for GitHub') do |branch|
  Abgleich.github.add_pull(branch)
  Abgleich.github.fail_with('timeout')
  scans = Abgleich.github.progress.fetch(:scans)
  @event_response = Abgleich.deliver_event('push', Abgleich.github.repository_notification)
  wait_for { Abgleich.github.progress.fetch(:scans) }.to be > scans, 'expected an active event check before restarting'
  Abgleich.github.fail_with(nil)
end

Given('the event check for branch {string} timed out') do |branch|
  Abgleich.github.add_pull(branch)
  Abgleich.github.fail_with('timeout')
  Abgleich.deliver_event('push', Abgleich.github.repository_notification)
  wait_for { Abgleich.logs }.to include(
    a_hash_including('msg' => 'automerge event finished', 'level' => 'error', 'error' => a_string_including('context deadline exceeded'))
  )
  expect_healthy_worker
end

When('GitHub delivers a {string} notification with action {string}') do |kind, action|
  reference = kind == 'tag push' ? 'refs/tags/v1' : 'refs/heads/master'
  kind = 'push' if kind == 'tag push'
  event = Abgleich.github.repository_notification(action:, ref: reference)
  event[:state] = action if kind == 'status'
  @event_response = Abgleich.deliver_event(kind, event)
end

When('GitHub delivers the repository notification') do
  @event_response = Abgleich.deliver_event('push', @repository_event, signature: @event_signature)
end

When('GitHub delivers repeated notifications before the sweep finishes') do
  requests = Array.new(5) { Thread.new { Abgleich.deliver_event('push', Abgleich.github.repository_notification) } }
  @event_responses = requests.map(&:value)
  Abgleich.github.resume_pull_listing
end

When('a new branch {string} and repeated notifications arrive during the check') do |branch|
  Abgleich.github.add_pull(branch)
  requests = Array.new(5) { Thread.new { Abgleich.deliver_event('push', Abgleich.github.repository_notification) } }
  @event_responses = requests.map(&:value)
  Abgleich.github.resume_pull_listing
end

When('notifications for {int} distinct repositories and duplicates arrive before the sweep finishes') do |count|
  events = Array.new(count) do |index|
    Abgleich.github.repository_notification(repository: "alexfalkowski/queued-#{index}")
  end
  events.push(events.first, events.last)
  @event_responses = events.map { |event| Abgleich.deliver_event('push', event) }
  Abgleich.github.resume_pull_listing
end

When('the worker resumes its regular schedule after GitHub recovers') do
  Abgleich.github.fail_with(nil)
end

When('GitHub recovers and sends another repository notification') do
  Abgleich.github.fail_with(nil)
  @event_response = Abgleich.deliver_event('push', Abgleich.github.repository_notification)
end

Then('the event delivery should return HTTP {int}') do |status|
  expect(@event_response.code).to eq(status), "expected event delivery HTTP #{status}, got #{@event_response.code}"
end

Then('every repository notification should return HTTP 204') do
  expect(@event_responses.map(&:code)).to all(eq(204)), 'expected every notification to be queued without waiting for GitHub'
end

Then('exactly {int} event checks should finish') do |count|
  @expected_event_checks = count
  wait_for { Abgleich.logs.count { |record| record['msg'] == 'automerge event finished' } }.to eq(count)
end

Then('no extra event checks should follow') do
  expect_event_barrier
  completed = Abgleich.logs.count { |record| record['msg'] == 'automerge event finished' }
  expect(completed).to eq(@expected_event_checks + 1), 'expected duplicates to combine before the later control notification'
end

Then('no further scheduled sweep should have completed') do
  expect(Abgleich.job_logs('automerge').length).to eq(1), 'expected the webhook to act before the next hourly sweep'
end

Then('the notification should cause no repository check') do
  expect_event_barrier
  expect(Abgleich.github.progress.fetch(:scans)).to eq(2), 'expected only the startup sweep and the later control notification to run'
  expect_no_automation_mutations
end

Then('branch {string} should await its next scheduled sweep') do |branch|
  expect(Abgleich.github.rebased?(branch)).to be(false), "expected #{branch} to remain untouched by another repository's notification"
end

Then('no event mutations should reach GitHub') do
  expect_no_automation_mutations
end

Then('the event check should fail') do
  wait_for { Abgleich.logs }.to include(a_hash_including('msg' => 'automerge event finished', 'level' => 'error'))
  expect_healthy_worker
end

Then('event and scheduled checks should not overlap') do
  expect(Abgleich::GitHubService.request_counts.fetch(:peak_scans)).to eq(1), 'expected event checks and sweeps to share execution'
end

Then('the interrupted event check should be cancelled') do
  expect(Abgleich.logs).to include(
    a_hash_including('msg' => 'automerge event finished', 'level' => 'error', 'error' => a_string_including('context canceled'))
  )
end
