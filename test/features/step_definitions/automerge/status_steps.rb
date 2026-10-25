# frozen_string_literal: true

Given('scheduled pull request listing is unavailable') do
  Abgleich.github.fail_repository(Abgleich::GitHubRepository::DEFAULT_REPOSITORY, 'list')
end

Given('its status request is {string}') do |condition|
  @comment_event = Abgleich.github.add_comment('contribution', body: '/ab status')
  case condition
  when 'edited' then @comment_event[:action] = 'edited'
  when 'now different text' then Abgleich.github.change_comment(@comment_event[:comment]['id'], :body, '/ab automerge')
  when 'now edited' then Abgleich.github.change_comment(@comment_event[:comment]['id'], :updated_at, '2026-01-01T00:00:06Z')
  when 'a bot request'
    Abgleich.github.change_comment(@comment_event[:comment]['id'], :user, { id: 77, login: 'maintainer', type: 'Bot' })
  when 'from another author'
    Abgleich.github.change_comment(@comment_event[:comment]['id'], :user, { id: 88, login: 'someone', type: 'User' })
  when 'for another PR'
    url = 'http://localhost:11001/github/repos/alexfalkowski/infraops/issues/999'
    Abgleich.github.change_comment(@comment_event[:comment]['id'], :issue_url, url)
  end
end

Given('its {string} changes during status assessment') do |field|
  path = { 'head commit' => 'head.sha', 'base commit' => 'base.sha', 'labels' => 'labels', 'identity' => 'node_id',
           'draft' => 'draft', 'state' => 'state', 'readiness' => 'mergeable_state', 'auto-merge' => 'auto_merge' }.fetch(field)
  Abgleich.github.fail_with("before mutation:#{path}")
end

Given('its repository {string} setting changes during status assessment') do |setting|
  field = { 'auto-merge' => 'auto_merge', 'squash merging' => 'squash_merge' }.fetch(setting)
  Abgleich.github.fail_with("status settings:#{field}")
end

Given('its status authorization changes during assessment through {string}') do |change|
  Abgleich.github.fail_with("status authorization:#{change}")
end

When('a maintainer requests status while publication is slow') do
  @comment_event = Abgleich.github.add_comment('contribution', body: '/ab status')
  Nonnative.start
  started = Process.clock_gettime(Process::CLOCK_MONOTONIC)
  @comment_response = Abgleich.deliver_comment(@comment_event)
  @command_duration = Process.clock_gettime(Process::CLOCK_MONOTONIC) - started
end

When('GitHub delivers the same status request twice') do
  @comment_event = Abgleich.github.add_comment('contribution', body: '/ab status')
  Nonnative.start
  expect(Abgleich.deliver_comment(@comment_event).code).to eq(204), 'expected the first status request to complete'
  @comment_response = Abgleich.deliver_comment(@comment_event)
end

Then('the status reply should explain {string}') do |decision|
  replies = Abgleich.github.replies
  expect(replies.length).to eq(1), "expected one published status reply explaining #{decision}"
  body = replies.first.fetch(:body)
  expect(body).to start_with("### 🤖 Abgleich status\n\n")
  expect(body).to include("**#{decision}**"), "expected status decision #{decision}, got #{body}"
  expect(body).to match(/\n[^\n\w*\s]+ \*\*#{Regexp.escape(decision)}\*\*/), 'expected an emoji beside the written decision'
  expect(body).to include('📝 **Reason**', '👉 **Next step**', '**Policy:**')
  expect(body).to match(/\*\*Policy:\*\* [^\n]+/), 'expected the applicable automation policy'
  expect(body).to match(/📝 \*\*Reason\*\*\n[^\n]+/), 'expected an explanation for the observed decision'
  expect(body).to match(/👉 \*\*Next step\*\*\n[^\n]+/), 'expected an actionable next step'
  expect(body.length).to be < 4000, 'expected a bounded status explanation'
end

Then('the status reply should identify the observed revision and request with auto-merge {string}') do |state|
  body = Abgleich.github.replies.last.fetch(:body)
  sha = Abgleich.github.pull_state('contribution').dig(:head, :sha)
  expect(body).to include("**GitHub auto-merge:** #{state}  \n", "**Commit:** `#{sha}`  \n")
  expect(body).to match(/\*\*Checked:\*\* \d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2} UTC/)
  checked_at = Time.strptime(body.match(/\*\*Checked:\*\* (.+ UTC)/)[1], '%Y-%m-%d %H:%M:%S %Z')
  expect(checked_at).to be_between(Time.now.utc - 10, Time.now.utc), 'expected the actual UTC assessment time for this request'
  comment_id = @comment_event.fetch(:comment).fetch('id')
  expect(body).to include("[Status request](#issuecomment-#{comment_id})")
end

Then('the status reply should include {string}') do |text|
  expect(Abgleich.github.replies.last.fetch(:body)).to include(text)
end

Then('the status request should cause no automation mutations') do
  expect_no_automation_mutations
end

Then('no status reply should be published') do
  expect(Abgleich.github.replies).to be_empty, 'expected no status reply for an untrusted or inapplicable request'
  expect(Abgleich.github.reply_requests).to eq(0), 'expected no attempt to publish a status reply'
end

Then('exactly {int} status publication attempts should reach GitHub') do |count|
  expect(Abgleich.github.reply_requests).to eq(count), "expected #{count} status publication attempts without automatic retries"
end

Then('status publication should not be reported as accepted') do
  comment_id = @comment_event.fetch(:comment).fetch('id')
  expect(Abgleich.github.comment_reactions(comment_id)).not_to include('+1'), 'expected no acceptance for unconfirmed publication'
  expect(Abgleich.logs).not_to include(a_hash_including('msg' => 'published pull request comment'))
end
