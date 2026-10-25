# frozen_string_literal: true

Given('AI summaries are configured for this repository') do
  Abgleich.prepare_github_webhook
  Abgleich.github.add_repository(Abgleich::GitHubRepository::DEFAULT_REPOSITORY)
  config = YAML.safe_load_file('.config/server.yaml')
  config['ai'] = { 'provider' => 'openai', 'openai' => { 'api_key' => 'nonnative-summary-key', 'url' => 'http://localhost:11001/openai/v1/' } }
  config['summarise'] = { 'model' => 'test-summary', 'interval' => '100ms', 'timeout' => '3s' }
  File.write('.config/server.yaml', YAML.dump(config))
end

Given('the repository topics are {string}') do |topics|
  Abgleich.github.add_repository(Abgleich::GitHubRepository::DEFAULT_REPOSITORY, topics: topics == 'none' ? [] : topics.split(', '))
end

Given('the pull request has {int} earlier comments') do |count|
  count.times { Abgleich.github.add_comment('contribution', body: 'Earlier discussion.') }
end

Given('an App summary follows those comments') do
  @existing_summary_body = "### 🤖 Abgleich summary\nAlready explained."
  Abgleich.github.publish_reply(Abgleich::GitHubRepository::DEFAULT_REPOSITORY, 1, @existing_summary_body)
end

Given('GitHub cannot retrieve the rest of the discussion') do
  Abgleich.github.fail_with('summary comments page:2')
end

Given('GitHub supplies discussion comments for another pull request') do
  Abgleich.github.fail_with('summary comment wrong PR')
end

Given('GitHub can retrieve only the first part of the changes') do
  Abgleich.github.fail_with('summary files page:2')
end

Given('GitHub provides {string}') do |information|
  modes = { 'an unidentified changed file' => 'missing file name', 'repeated changes' => 'duplicate file',
            'an incomplete list of changes' => 'incomplete file list', 'an unconfirmed list of changes' => 'missing file count' }
  Abgleich.github.file_mode = modes.fetch(information)
end

Given('the AI service is slow to answer') do
  Abgleich.openai.mode = 'slow'
end

Given('the AI service supplies {string}') do |answer|
  modes = { 'a refusal' => 'refusal', 'an unfinished answer' => 'incomplete',
            'no answer' => 'null response', 'an answer arriving late' => 'timeout' }
  Abgleich.openai.mode = modes.fetch(answer)
end

Given('its summary request was edited before delivery') do
  @comment_event = Abgleich.github.add_comment('contribution', body: '/ab summarise')
  Abgleich.github.change_comment(@comment_event.fetch(:comment).fetch('id'), :updated_at, '2026-01-01T00:00:06Z')
end

Given('the pull request has already been summarised') do
  @comment_event = Abgleich.github.add_comment('contribution', body: '/ab summarise')
  Nonnative.start
  expect(Abgleich.deliver_comment(@comment_event).code).to eq(204)
  wait_for { Abgleich.github.replies.any? { |reply| reply[:body].start_with?("### 🤖 Abgleich summary\n") } }.to be(true)
  expect_summary_check
  @existing_summary_body = Abgleich.github.replies.first.fetch(:body)
end

Given('new commits arrived before the worker restarted') do
  Abgleich.restart { Abgleich.github.change_summary_state('head changed') }
end

Given('the worker has restarted') do
  Abgleich.restart
end

Given('GitHub reports an error after publishing the summary') do
  Abgleich.github.fail_with('lost reply response')
end

Given('an earlier request failed while the AI service was busy') do
  Abgleich.openai.mode = 'rate limit'
  event = Abgleich.github.add_comment('contribution', body: '/ab summarise')
  Nonnative.start
  expect(Abgleich.deliver_comment(event).code).to eq(204)
  wait_for { Abgleich.github.replies.any? { |reply| reply[:body].include?('Summary failed') } }.to be(true)
  expect_summary_check
  expect(Abgleich.openai.requests.length).to eq(1), 'expected no automatic retries during the outage'
  expect(Abgleich.logs).to include(a_hash_including('msg' => 'ai generation', 'level' => 'error'))
  expect_healthy_worker
  Abgleich.openai.mode = nil
end

When('two maintainers request a summary before either receives it') do
  @comment_event = Abgleich.github.add_comment('contribution', body: '/ab summarise')
  Nonnative.start
  expect(Abgleich.deliver_comment(@comment_event).code).to eq(204)
  wait_for { Abgleich.openai.requests.length }.to eq(1), 'expected the first summary to be in progress'
  event = Abgleich.github.add_comment('contribution', body: '/ab summarise')
  expect(Abgleich.deliver_comment(event).code).to eq(204)
  expect(Abgleich.logs).not_to include(a_hash_including('msg' => 'ai generation')), 'expected both requests before generation finished'
end

When('a reader requests a summary') do
  @comment_event = Abgleich.github.add_comment('contribution', permission: 'read', body: '/ab summarise')
  Nonnative.start
  @comment_response = Abgleich.deliver_comment(@comment_event)
end

When('a maintainer requests status while waiting for a summary') do
  event = Abgleich.github.add_comment('contribution', body: '/ab summarise')
  Nonnative.start
  expect(Abgleich.deliver_comment(event).code).to eq(204)
  wait_for { Abgleich.openai.requests.length }.to eq(1), 'expected summary preparation to begin'
  started = Process.clock_gettime(Process::CLOCK_MONOTONIC)
  @comment_event = Abgleich.github.add_comment('contribution', body: '/ab status')
  @comment_response = Abgleich.deliver_comment(@comment_event)
  @command_duration = Process.clock_gettime(Process::CLOCK_MONOTONIC) - started
  expect(Abgleich.logs).not_to include(a_hash_including('msg' => 'ai generation')), 'expected status to finish before the summary'
end

Then('the summary should identify the request and the pull request revision') do
  wait_for do
    Abgleich.github.replies.any? { |reply| reply[:body].start_with?("### 🤖 Abgleich summary\n") }
  end.to be(true), 'expected a summary'
  summaries = Abgleich.github.replies.select { |reply| reply[:body].start_with?("### 🤖 Abgleich summary\n") }
  expect(summaries.length).to eq(1), 'expected one App summary'
  body = summaries.first.fetch(:body)
  expect(body).to include('The worker now processes requests predictably.', 'patches', 'not executed')
  expect(body).to include(Abgleich.github.pull_state('contribution').dig(:head, :sha))
  expect(body).to include("#issuecomment-#{@comment_event.fetch(:comment).fetch('id')}")
  expect(Abgleich.openai.requests.last.fetch('input')).to include('File: test.go'), 'expected all changed files to be considered'
end

Then(/^the request should be (accepted|temporarily declined|refused without write access)$/) do |outcome|
  statuses = { 'accepted' => 204, 'temporarily declined' => 503, 'refused without write access' => 403 }
  expect(@comment_response.code).to eq(statuses.fetch(outcome)), "expected the request to be #{outcome}"
end

Then('the original summary should be kept without another AI request') do
  expect_summary_check
  expect(Abgleich.openai.requests.length).to eq(1), 'expected the existing summary to prevent another AI request'
  expect(Abgleich.github.replies.map { |reply| reply[:body] }).to eq([@existing_summary_body]), 'expected the original summary to remain'
end

Then('the maintainer should receive failure feedback without a summary') do
  wait_for do
    Abgleich.github.replies.any? { |reply| reply[:body].include?('Summary failed') }
  end.to be(true), 'expected visible failure feedback'
  expect(Abgleich.github.replies.none? { |reply| reply[:body].start_with?("### 🤖 Abgleich summary\n") }).to be(true)
  expect(Abgleich.openai.requests.length).to eq(1), 'expected one AI request without automatic retries'
  expect_healthy_worker
end

Then('no source should be shared with the AI service') do
  expect_summary_check
  expect(Abgleich.openai.requests).to be_empty, 'expected no source to leave GitHub'
end

Then('the AI service should remain unused') do
  expect(Abgleich.openai.requests).to be_empty, 'expected no AI work'
end

Then('the maintainer should be told that the changes could not be read') do
  wait_for do
    Abgleich.github.replies.any? { |reply| reply[:body].include?('Summary failed') }
  end.to be(true), 'expected feedback that the pull request changes could not be read'
  expect(Abgleich.openai.requests).to be_empty, 'expected incomplete information to prevent sharing source'
  expect(Abgleich.github.replies.none? { |reply| reply[:body].start_with?("### 🤖 Abgleich summary\n") }).to be(true)
  expect_healthy_worker
end

Then(/^exactly (\d+) requests? should reach the AI service$/) do |count|
  expect(Abgleich.openai.requests.length).to eq(count.to_i), 'expected one AI request per requested attempt'
end

Then(/^the operator should see that summary generation (succeeded|failed)$/) do |outcome|
  level = outcome == 'succeeded' ? 'info' : 'error'
  wait_for { Abgleich.logs }.to include(a_hash_including('msg' => 'ai generation', 'level' => level))
end

Then('the summary should leave pull request automation unchanged') do
  expect_no_automation_mutations
end

Then('the maintainer should receive status before the AI service finishes') do
  expect(@command_duration).to be < 1, 'expected summary preparation to leave status responsive'
  expect(Abgleich.github.replies).to include(a_hash_including(body: a_string_including('### 🤖 Abgleich status')))
end

Then('the maintainer should be asked to request a summary of the new revision') do
  wait_for { Abgleich.github.replies.any? { |reply| reply[:body].include?('revision changed') } }.to be(true)
  expect(Abgleich.github.replies.last.fetch(:body)).not_to include('The worker now processes requests predictably.')
end

Then('no summary should be posted after permission is withdrawn') do
  wait_for { Abgleich.openai.requests.length }.to eq(1), 'expected preparation before permission changed'
  expect_summary_check
  expect(Abgleich.github.replies).to be_empty, 'expected withdrawn permission to prevent publication'
end

Given('a summary request was interrupted by a worker restart') do
  config = YAML.safe_load_file('.config/server.yaml')
  config['summarise']['interval'] = '10s'
  File.write('.config/server.yaml', YAML.dump(config))
  Nonnative.start
  wait_for { Abgleich.job_logs('summarise') }.not_to be_empty
  @comment_event = Abgleich.github.add_comment('contribution', body: '/ab summarise')
  expect(Abgleich.deliver_comment(@comment_event).code).to eq(204)
  expect(Abgleich.openai.requests).to be_empty, 'expected the interrupted request to be waiting'
  Abgleich.restart do
    config['summarise']['interval'] = '100ms'
    File.write('.config/server.yaml', YAML.dump(config))
  end
  expect_summary_check
  expect(Abgleich.openai.requests).to be_empty, 'expected a restart to discard pending work until reissue'
  expect(Abgleich.github.replies).to be_empty, 'expected no reply before the maintainer reissues the request'
end

Then('no summary should be posted') do
  expect(Abgleich.github.replies).to be_empty, 'expected no App reply'
end

Then('no completed summary should be posted') do
  expect(Abgleich.github.replies.none? { |reply| reply[:body].start_with?("### 🤖 Abgleich summary\n") }).to be(true)
end

Given('the contribution contains {string}') do |changes|
  modes = { 'more than twenty changed files' => 'file budget', 'large combined changes' => 'patch budget',
            'an oversized individual change' => 'oversized patch', 'a change without a text diff' => 'missing patch' }
  Abgleich.github.file_mode = modes.fetch(changes)
end

Then('only permitted source should be shared with an explanation of {string}') do |omission|
  wait_for { Abgleich.openai.requests.length }.to eq(1)
  input = Abgleich.openai.requests.first.fetch('input')
  reasons = { 'omitted files' => 'file limit', 'omitted patches' => 'patch budget', 'unavailable changes' => 'Patch omitted' }
  expect(input).to include(reasons.fetch(omission)), 'expected omitted changes to be explained to the AI service'
  expect(input).not_to include('file21.go', 'withheld-source'), 'expected excluded source to remain private'
end

Given('a comment resembling an App summary was posted by {string}') do |author|
  event = Abgleich.github.add_comment('contribution', body: "### 🤖 Abgleich summary\nAn unrelated comment.")
  id = event.fetch(:comment).fetch('id')
  if author == 'another App'
    Abgleich.github.change_comment(id, :user, { id: 9000, login: 'abgleich[bot]', type: 'Bot' })
    Abgleich.github.change_comment(id, :performed_via_github_app, { id: 123 })
  end
end

Given('the AI service includes markup and mentions in its answer') do
  Abgleich.openai.mode = 'hostile prose'
end

Then('the summary should show the answer without active markup or mentions') do
  wait_for { Abgleich.github.replies.last&.fetch(:body)&.include?('alert') }.to be(true)
  body = Abgleich.github.replies.last.fetch(:body)
  expect(body).to include('&lt;script&gt;', "@\u200beveryone", '\\[click\\]\\(', '\\!\\[track\\]\\(')
  expect(body).not_to include('<script>', '@everyone', '[click](', '![track](')
end

Given('the revision changes through {string} while the summary is prepared') do |change|
  modes = { 'a new pull request commit' => 'head changed', 'a new target branch commit' => 'base changed' }
  Abgleich.openai.on_request = -> { Abgleich.github.change_summary_state(modes.fetch(change)) }
end

Given('summary permission is withdrawn through {string} while the answer is prepared') do |change|
  modes = { 'loss of write access' => 'permission revoked', 'an edited request' => 'request edited' }
  Abgleich.openai.on_request = -> { Abgleich.github.change_summary_state(modes.fetch(change)) }
end

Given('the request changes through {string} before any source is shared') do |change|
  modes = { 'loss of write access' => 'permission revoked', 'a new pull request commit' => 'head changed' }
  Abgleich.github.collection_change = modes.fetch(change)
end

Given('repository topics are removed before any source is shared') do
  Abgleich.github.collection_change = 'topic removed'
end

Given('repository topics are removed while the answer is prepared') do
  Abgleich.openai.on_request = -> { Abgleich.github.change_summary_state('topic removed') }
end

Given('the repository allows summaries but has automatic merging disabled') do
  Abgleich.github.add_repository(
    Abgleich::GitHubRepository::DEFAULT_REPOSITORY,
    auto_merge: false, squash_merge: false
  )
end

Given('the AI service credential is {string}') do |state|
  sources = { 'unavailable' => 'file:secrets/missing-ai-key', 'blank' => 'file:secrets/github-webhook-whitespace-secret' }
  config = YAML.safe_load_file('.config/server.yaml')
  config['ai']['openai']['api_key'] = sources.fetch(state)
  File.write('.config/server.yaml', YAML.dump(config))
end

Given('the operator selects an unsupported AI service') do
  config = YAML.safe_load_file('.config/server.yaml')
  config['ai']['provider'] = 'unsupported'
  File.write('.config/server.yaml', YAML.dump(config))
end

Given('AI summaries are disabled') do
  config = YAML.safe_load_file('.config/server.yaml')
  config.delete('ai')
  config.delete('summarise')
  File.write('.config/server.yaml', YAML.dump(config))
end

Given('the operator enables summaries without an AI service') do
  config = YAML.safe_load_file('.config/server.yaml')
  config.delete('ai')
  File.write('.config/server.yaml', YAML.dump(config))
end

Given('a maintainer is waiting for a summary') do
  config = YAML.safe_load_file('.config/server.yaml')
  config['summarise']['interval'] = '1s'
  File.write('.config/server.yaml', YAML.dump(config))
  Nonnative.start
  wait_for { Abgleich.job_logs('summarise') }.not_to be_empty
  @comment_event = Abgleich.github.add_comment('contribution', body: '/ab summarise')
  expect(Abgleich.deliver_comment(@comment_event).code).to eq(204)
end

Given('an App summary arrives while they wait') do
  @existing_summary_body = "### 🤖 Abgleich summary\nAlready explained."
  Abgleich.github.publish_reply(Abgleich::GitHubRepository::DEFAULT_REPOSITORY, 1, @existing_summary_body)
end

Given('another App summary arrives while the AI service is answering') do
  @existing_summary_body = "### 🤖 Abgleich summary\nAlready explained."
  Abgleich.openai.on_request = lambda do
    Abgleich.github.publish_reply(Abgleich::GitHubRepository::DEFAULT_REPOSITORY, 1, @existing_summary_body)
  end
end

When('Abgleich processes the waiting request') do
  expect_summary_check
end

Then('the existing summary should satisfy the waiting request') do
  expect(Abgleich.github.replies.map { |reply| reply[:body] }).to eq([@existing_summary_body])
  expect(Abgleich.openai.requests).to be_empty, 'expected the waiting request to reuse the existing summary'
end

Then('the existing App summary should remain the only summary') do
  expect_summary_check
  expect(Abgleich.github.replies.map { |reply| reply[:body] }).to eq([@existing_summary_body])
end

Given('four maintainers are waiting for their summaries') do
  config = YAML.safe_load_file('.config/server.yaml')
  config['summarise']['interval'] = '10s'
  File.write('.config/server.yaml', YAML.dump(config))
  Nonnative.start
  wait_for { Abgleich.job_logs('summarise') }.not_to be_empty
  4.times do |index|
    branch = "queued-#{index}"
    Abgleich.github.add_pull(branch, auto_merge: false, current: true)
    event = Abgleich.github.add_comment(branch, body: '/ab summarise')
    expect(Abgleich.deliver_comment(event).code).to eq(204), "expected the summary for #{branch} to be accepted"
  end
end

Then('ordinary automation should continue while those requests wait') do
  next_checks = Abgleich.job_logs('automerge').length + 2
  wait_for { Abgleich.job_logs('automerge').length }.to be >= next_checks, 'expected ordinary automation to continue while summaries wait'
  expect_successful_check('automerge')
  expect(Abgleich.openai.requests).to be_empty, 'expected the accepted requests to remain waiting'
end
