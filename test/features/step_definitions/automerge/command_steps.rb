# frozen_string_literal: true

Given('GitHub App comment commands are configured') do
  Abgleich.prepare_github_webhook
end

Given('an earlier command completed before its installation token approached expiry') do
  Abgleich.github.token_lifetime = 61
  Abgleich.github.add_pull('earlier-contribution', auto_merge: false, current: true)
  Nonnative.start
  response = Abgleich.deliver_comment(Abgleich.github.add_comment('earlier-contribution'))
  expect(response.code).to eq(204), 'expected the earlier App command to complete'
  wait_for { Time.now.utc >= Abgleich.github.token_renews_at }.to be(true), 'expected installation token renewal to become due'
end

Given('comment deliveries are disabled') do
  config = YAML.safe_load_file('.config/server.yaml')
  config['github'].delete('webhook')
  File.write('.config/server.yaml', YAML.dump(config))
end

Given('installation token acquisition exceeds the command deadline') do
  config = YAML.safe_load_file('.config/server.yaml')
  config['github']['webhook']['timeout'] = '200ms'
  File.write('.config/server.yaml', YAML.dump(config))
  Abgleich.github.fail_with('installation token timeout')
end

Given('installation token acquisition is still running when a command arrives') do
  config = YAML.safe_load_file('.config/server.yaml')
  config['scheduler']['timeout'] = '10s'
  config['github']['webhook']['timeout'] = '200ms'
  File.write('.config/server.yaml', YAML.dump(config))
  Abgleich.github.fail_with('installation token timeout')
  Nonnative.start
  wait_for { Abgleich::GitHubService.request_counts.fetch(:installation_requests) }.to be >= 1, 'expected token acquisition to be running'
end

Given('its comment delivery has {string}') do |problem|
  @comment_event = Abgleich.github.add_comment('contribution')
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
  @comment_event = Abgleich.github.add_comment('contribution')
  if event == 'an edited comment'
    @comment_event[:action] = 'edited'
  else
    @comment_event[:issue].delete(:pull_request)
  end
end

Given('its original command comment now has {string}') do |change|
  @comment_event = Abgleich.github.add_comment('contribution')
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

Given('the comment for branch {string} was created {string} a {string} event') do |branch, timing, event|
  @comment_event = Abgleich.github.add_comment(branch)
  Abgleich.github.label_pull(branch, 'bug')
  occurred_at = {
    'before' => '2026-01-01T00:00:06Z',
    'at the same time' => '2026-01-01T00:00:05Z',
    'after' => '2026-01-01T00:00:04Z'
  }.fetch(timing)
  Abgleich.github.revoke_comment_request(branch, event, occurred_at:)
end

Given('an earlier comment labeled branch {string} before its label was removed') do |branch|
  Abgleich.github.add_pull(branch, auto_merge: false, current: true)
  @comment_event = Abgleich.github.add_comment(branch)
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
  @comment_event = Abgleich.github.add_comment('contribution')
  Nonnative.start
  expect(Abgleich.deliver_comment(@comment_event).code).to eq(204)
  @comment_response = Abgleich.deliver_comment(@comment_event)
end

When('GitHub delivers the same automerge comment concurrently') do
  @comment_event = Abgleich.github.add_comment('contribution')
  Nonnative.start
  requests = Array.new(2) { Thread.new { Abgleich.deliver_comment(@comment_event) } }
  responses = requests.map(&:value)
  expect(responses.map(&:code) - [204, 503]).to be_empty, 'expected completion or a busy response for concurrent deliveries'
  @comment_response = responses.find { |response| response.code == 204 }
  expect(@comment_response).not_to be_nil, 'expected one concurrent command to complete'
end

When('GitHub redelivers an automerge comment whose first response was lost') do
  @comment_event = Abgleich.github.add_comment('contribution')
  Nonnative.start
  expect(Abgleich.deliver_comment(@comment_event).code).to eq(503)
  @comment_response = Abgleich.deliver_comment(@comment_event)
end

When('a maintainer requests auto-merge while token acquisition is slow') do
  event = Abgleich.github.add_comment('contribution')
  Nonnative.start
  started = Process.clock_gettime(Process::CLOCK_MONOTONIC)
  @comment_response = Abgleich.deliver_comment(event)
  @command_duration = Process.clock_gettime(Process::CLOCK_MONOTONIC) - started
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
