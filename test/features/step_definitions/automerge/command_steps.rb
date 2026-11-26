# frozen_string_literal: true

Given('GitHub App comment commands are configured') do
  Abgleich.prepare_github_webhook
end

Given('an earlier command completed before its installation token approached expiry') do
  Abgleich.github.token_lifetime = 61
  Abgleich.github.add_pull('earlier-contribution', auto_merge: false, current: true)
  Nonnative.start
  response = Abgleich.deliver_comment(Abgleich.github.add_comment('earlier-contribution', body: '/ab automerge'))
  expect(response.code).to eq(204), 'expected the earlier App command to complete'
  wait_for { Time.now.utc >= Abgleich.github.token_renews_at }.to be(true), 'expected installation token renewal to become due'
end

Given('comment deliveries are disabled') do
  Abgleich.use_config('server')
end

Given('installation token acquisition exceeds the command deadline') do
  Abgleich.use_config('webhook_short_deadline')
  Abgleich.github.fail_with('installation token timeout')
end

Given('installation token acquisition is still running when a command arrives') do
  Abgleich.use_config('webhook_token_wait')
  Abgleich.github.fail_with('installation token timeout')
  Nonnative.start
  wait_for { Abgleich::GitHubService.request_counts.fetch(:installation_requests) }.to be >= 1, 'expected token acquisition to be running'
end

Given('its comment delivery has {string}') do |problem|
  @comment_event = Abgleich.github.add_comment('contribution', body: '/ab automerge')
  case problem
  when 'an invalid signature' then @signature = 'sha256=invalid'
  when 'a missing signature' then @signature = ''
  when 'another installation' then @comment_event[:installation][:id] += 1
  when 'another owner'
    @comment_event[:repository][:owner][:login] = 'someone'
    @comment_event[:repository][:full_name] = 'someone/infraops'
  when 'missing PR identity' then @comment_event[:issue] = nil
  when 'an invalid JSON body' then @comment_event = '{'
  when 'an oversized body' then @comment_event[:comment]['body'] = 'a' * (1024 * 1024)
  end
end

Given('its comment delivery is {string}') do |event|
  @comment_event = Abgleich.github.add_comment('contribution', body: '/ab automerge')
  if event == 'an edited comment'
    @comment_event[:action] = 'edited'
  else
    @comment_event[:issue].delete(:pull_request)
  end
end

Given('its original command comment now has {string}') do |change|
  @comment_event = Abgleich.github.add_comment('contribution', body: '/ab automerge')
  field, value = {
    'different text' => [:body, 'please wait'],
    'an unknown command' => [:body, '/ab unknown'],
    'extra command text' => [:body, '/ab automerge please'],
    'an edit timestamp' => [:updated_at, '2026-01-01T00:00:06Z'],
    'no edit timestamp' => [:updated_at, nil],
    'no author login' => [:user, { id: 77, login: '', type: 'User' }],
    'a bot author' => [:user, { id: 77, login: 'maintainer', type: 'Bot' }],
    'another pull request' => [:issue_url, 'http://localhost:11001/github/repos/alexfalkowski/infraops/issues/999'],
    'another author' => [:user, { id: 88, login: 'someone', type: 'User' }]
  }.fetch(change)
  Abgleich.github.change_comment(@comment_event.fetch(:comment).fetch('id'), field, value)
end

Given('its command authorization changes before labeling through {string}') do |change|
  Abgleich.github.fail_with("status authorization:#{change}")
end

Given('the comment for branch {string} was created {string} {string}') do |branch, timing, event|
  @comment_event = Abgleich.github.add_comment(branch, body: '/ab automerge')
  Abgleich.github.label_pull(branch, 'bug')
  occurred_at = {
    'before' => '2026-01-01T00:00:06Z',
    'at the same time' => '2026-01-01T00:00:05Z',
    'after' => '2026-01-01T00:00:04Z'
  }.fetch(timing)
  events = { 'the automerge label was removed' => 'UnlabeledEvent', 'auto-merge was disabled' => 'AutoMergeDisabledEvent' }
  Abgleich.github.revoke_comment_request(branch, events.fetch(event), occurred_at:)
end

Given('an earlier comment labeled branch {string} before its label was removed') do |branch|
  Abgleich.github.add_pull(branch, auto_merge: false, current: true)
  @comment_event = Abgleich.github.add_comment(branch, body: '/ab automerge')
  Nonnative.start
  expect(Abgleich.deliver_comment(@comment_event).code).to eq(204)
  Abgleich.github.revoke_comment_request(branch)
  Nonnative.stop
  Nonnative.clear_pool
end

When('a maintainer with {string} access comments {string}') do |permission, body|
  @comment_event = Abgleich.github.add_comment('contribution', permission:, body:)
  Nonnative.start
  @comment_response = Abgleich.deliver_comment(@comment_event)
end

When('GitHub delivers that comment') do
  Nonnative.start
  @comment_response = Abgleich.deliver_comment(@comment_event, signature: @signature)
end

When('GitHub redelivers that comment after the worker restarts') do
  Nonnative.start
  @comment_response = Abgleich.deliver_comment(@comment_event)
end

When('GitHub delivers the same automerge comment twice') do
  @comment_event = Abgleich.github.add_comment('contribution', body: '/ab automerge')
  Nonnative.start
  expect(Abgleich.deliver_comment(@comment_event).code).to eq(204)
  @comment_response = Abgleich.deliver_comment(@comment_event)
end

When("GitHub delivers maintainer commands and a reader's status request during labeling") do
  automerge = Abgleich.github.add_comment('contribution', body: '/ab automerge')
  writer = { id: 78, login: 'status-writer', type: 'User' }
  reader = { id: 79, login: 'reader', type: 'User' }
  status = Abgleich.github.add_comment('contribution', body: '/ab status', author: writer)
  denied = Abgleich.github.add_comment('contribution', body: '/ab status', permission: 'read', author: reader)
  Abgleich.github.pause_label_updates
  Nonnative.start
  original = Thread.new { Abgleich.deliver_comment(automerge) }
  wait_for { Abgleich.github.label_update_waiting? }.to be(true), 'expected GitHub to be holding the label update'

  @reader_response = Abgleich.deliver_comment(denied)
  @label_pending = Abgleich.github.label_update_waiting?
  previous_checks = Abgleich.github.permission_checks('maintainer')
  duplicate = Thread.new { Abgleich.deliver_comment(automerge) }
  status_delivery = Thread.new { Abgleich.deliver_comment(status) }
  wait_for { Abgleich.github.permission_checks('maintainer') }.to be > previous_checks, 'expected overlap with the redelivery'
  wait_for { Abgleich.github.permission_checks('status-writer') }.to be > 0, 'expected the status delivery while the label was pending'
  Abgleich.github.resume_label_updates
  @comment_responses = [original.value, duplicate.value, status_delivery.value]
ensure
  Abgleich.github.resume_label_updates
  original&.join
  duplicate&.join
  status_delivery&.join
end

When('GitHub redelivers an automerge comment whose first response was lost') do
  @comment_event = Abgleich.github.add_comment('contribution', body: '/ab automerge')
  Nonnative.start
  expect(Abgleich.deliver_comment(@comment_event).code).to eq(503)
  @comment_response = Abgleich.deliver_comment(@comment_event)
end

When('GitHub delivers overlapping and then repeated reader requests during labeling') do
  automerge = Abgleich.github.add_comment('contribution', body: '/ab automerge')
  reader = { id: 79, login: 'reader', type: 'User' }
  denied = Abgleich.github.add_comment('contribution', body: '/ab status', permission: 'read', author: reader)
  Abgleich.github.pause_label_updates
  Abgleich.github.pause_permission_checks('reader')
  Nonnative.start
  original = Thread.new { Abgleich.deliver_comment(automerge) }
  wait_for { Abgleich.github.label_update_waiting? }.to be(true), 'expected GitHub to be holding the label update'

  checking = Array.new(2) { Thread.new { Abgleich.deliver_comment(denied) } }
  wait_for { Abgleich.github.permission_checks('reader') }.to eq(2), 'expected both permission lookups to reach GitHub'
  @concurrent_excess = Abgleich.deliver_comment(denied)
  Abgleich.github.resume_permission_checks('reader')
  @checked_readers = checking.map(&:value) + Array.new(2) { Abgleich.deliver_comment(denied) }
  @rate_excess = Abgleich.deliver_comment(denied)

  Abgleich.github.resume_label_updates
  @label_response = original.value
ensure
  Abgleich.github.resume_permission_checks('reader')
  Abgleich.github.resume_label_updates
  checking&.each(&:join)
  original&.join
end

Then('excess reader requests should return HTTP {int} without a GitHub permission lookup') do |status|
  expect(@concurrent_excess.code).to eq(status), 'expected the worker to reject an excess concurrent lookup'
  expect(@checked_readers.map(&:code)).to eq([403, 403, 403, 403]), 'expected the bounded lookups to deny the reader'
  expect(@rate_excess.code).to eq(status), 'expected the worker to reject an excess repeat lookup'
  expect(Abgleich.github.permission_checks('reader')).to eq(4), 'expected only the permitted lookups to reach GitHub'
end

Then("the maintainer's label command should return HTTP {int}") do |status|
  expect(@label_response.code).to eq(status)
end

When("a status delivery's deadline expires while a later automerge command holds the slot") do
  first = Abgleich.github.add_comment('contribution', body: '/ab automerge')
  later = Abgleich.github.add_comment('later-contribution', body: '/ab automerge')
  writer = { id: 78, login: 'status-writer', type: 'User' }
  @waiting_comment = Abgleich.github.add_comment('contribution', body: '/ab status', author: writer)
  Abgleich.github.pause_label_updates
  Abgleich.github.pause_permission_checks('status-writer')
  Nonnative.start
  original = Thread.new { Abgleich.deliver_comment(first) }
  wait_for { Abgleich.github.label_update_waiting? }.to be(true), 'expected the first automerge command to hold the slot'

  waiting = Thread.new { timed_comment_delivery(@waiting_comment) }
  wait_for { Abgleich.github.permission_checks('status-writer') }.to eq(1), 'expected the earlier status permission check'
  # Separate the webhook deadlines so the later delivery can survive the earlier timeout.
  sleep 0.6
  Abgleich.github.resume_label_updates

  Abgleich.github.pause_label_updates
  next_label = Thread.new { Abgleich.deliver_comment(later) }
  wait_for { Abgleich.github.label_update_waiting? }.to be(true), 'expected the later automerge command to hold the slot'
  Abgleich.github.resume_permission_checks('status-writer')
  @waiting_response, @wait_duration = waiting.value
  @later_label_pending = Abgleich.github.label_update_waiting?

  Abgleich.github.resume_label_updates
  @label_responses = [original.value, next_label.value]
ensure
  Abgleich.github.resume_permission_checks('status-writer')
  Abgleich.github.resume_label_updates
  [original, waiting, next_label].compact.each(&:join)
end

Then('the status delivery should return HTTP {int} while the later command is still labeling') do |status|
  expect(@waiting_response.code).to eq(status)
  expect(@later_label_pending).to be(true), 'expected the later command to hold the slot at the earlier deadline'
  expect(@wait_duration).to be < 1.9, 'expected the wait to end with time reserved inside the two-second webhook deadline'
  expect(Abgleich.logs).to include(a_hash_including('msg' => 'github webhook failed', 'error' => 'context deadline exceeded'))
end

Then('the status comment should have no receipt feedback') do
  comment_id = @waiting_comment.fetch(:comment).fetch('id')
  expect(Abgleich.github.comment_reactions(comment_id)).to be_empty
end

Then('both automerge deliveries should return HTTP {int}') do |status|
  expect(@label_responses.map(&:code)).to eq([status, status])
end

When('a maintainer requests auto-merge while token acquisition is slow') do
  event = Abgleich.github.add_comment('contribution', body: '/ab automerge')
  Nonnative.start
  @comment_response, @command_duration = timed_comment_delivery(event)
end

def timed_comment_delivery(event)
  started = Process.clock_gettime(Process::CLOCK_MONOTONIC)
  response = Abgleich.deliver_comment(event)
  [response, Process.clock_gettime(Process::CLOCK_MONOTONIC) - started]
end

Then('all three maintainer deliveries should return HTTP {int}') do |status|
  statuses = @comment_responses.map(&:code)
  expect(statuses).to eq([status, status, status]), 'expected the automerge comment, its redelivery, and the status comment to complete'
end

Then("the reader's status delivery should return HTTP {int} before labeling finishes") do |status|
  expect(@reader_response.code).to eq(status), 'expected the reader to be denied while the command slot was busy'
  expect(@label_pending).to be(true), 'expected labeling to still be pending when the reader was denied'
end

Then('the comment delivery should return HTTP {int}') do |status|
  expect(@comment_response.code).to eq(status), "expected comment delivery HTTP #{status}, got #{@comment_response.code}"
end

Then(/^exactly (\d+) label requests? should reach GitHub$/) do |count|
  expect(Abgleich.github.label_requests).to eq(count.to_i), "expected #{count} label requests at GitHub"
end

Then('branch {string} should have no automerge label') do |branch|
  labels = Abgleich.github.pull_state(branch).fetch(:labels).map { |label| label.fetch(:name) }
  expect(labels).not_to include('automerge'), "expected the removed automerge label on #{branch} to stay removed"
end

Then('the installation token should be renewed') do
  wait_for { Abgleich.github.token_requests }.to be >= 2, 'expected installation-token renewal without restarting the worker'
end

Then('the command should finish within one second') do
  expect(@command_duration).to be < 1, 'expected the command deadline to bound waiting for another token request'
end
