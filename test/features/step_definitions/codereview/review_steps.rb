# frozen_string_literal: true

Given('AI code reviews are configured for this repository') do
  Abgleich.prepare_github_webhook
  Abgleich.github.add_repository(Abgleich::GitHubRepository::DEFAULT_REPOSITORY)
  Abgleich.github.file_mode = 'code review'
  Abgleich.use_config('codereview')
end

Given('the AI review answer has {string}') do |mode|
  Abgleich.openai.mode = "review #{mode}"
end

Given('the code review configuration has {string}') do |problem|
  fixtures = {
    'no AI provider' => 'codereview_missing_ai',
    'no model' => 'codereview_missing_model',
    'an empty model' => 'codereview_empty_model',
    'an invalid interval' => 'codereview_invalid_interval',
    'an invalid timeout' => 'codereview_invalid_timeout',
    'disabled reviews' => 'ai',
    'the default schedule' => 'codereview_default_schedule'
  }

  Abgleich.use_config(fixtures.fetch(problem))
end

Given('the pull request has already received a code review') do
  @comment_event = Abgleich.github.add_comment('contribution', body: '/ab codereview')
  Nonnative.start
  expect(Abgleich.deliver_comment(@comment_event).code).to eq(204)
  wait_for { Abgleich.github.replies.any? { |reply| reply[:body].start_with?("### 🤖 Abgleich code review\n") } }.to be(true)
  expect_review_check
  @existing_review_body = Abgleich.github.replies.last.fetch(:body)
end

Given('a code review lookalike was posted by {string}') do |author|
  step 'the pull request has already received a code review'
  Abgleich.github.withdraw_review_reply

  event = Abgleich.github.add_comment('contribution', body: @existing_review_body)
  id = event.fetch(:comment).fetch('id')

  if author == 'another App'
    Abgleich.github.change_comment(id, :user, { id: 9000, login: 'abgleich[bot]', type: 'Bot' })
    Abgleich.github.change_comment(id, :performed_via_github_app, { id: 123 })
  end
end

Given('the review request loses eligibility through {string} {word} inference') do |change, stage|
  if stage == 'before'
    Abgleich.github.collection_change = "review:#{change}"
  else
    Abgleich.openai.on_request = -> { Abgleich.github.change_review_state(change) }
  end
end

Given('the pull request has no target branch revision') do
  Abgleich.github.change_review_state('missing base revision')
end

Given('code reviews and summaries are independently enabled') do
  Abgleich.use_config('codereview_summarise')
end

Given('the AI provider rejects the review context as too large') do
  Abgleich.openai.mode = 'context limit'
end

Given('the reviewed repository has hold and automatic merging disabled') do
  Abgleich.github.label_pull('contribution', 'hold')
  Abgleich.github.add_repository(Abgleich::GitHubRepository::DEFAULT_REPOSITORY, auto_merge: false, squash_merge: false, topics: [])
end

Given('the worker is waiting for the code review provider') do
  Abgleich.openai.mode = 'timeout'
  Nonnative.start
  event = Abgleich.github.add_comment('contribution', body: '/ab codereview')
  expect(Abgleich.deliver_comment(event).code).to eq(204)
  wait_for { Abgleich.openai.requests.length }.to eq(1)
end

Given('a previous App review was temporarily removed') do
  Abgleich.github.withdraw_review_reply
end

Given('that App review is restored while inference is running') do
  Abgleich.openai.on_request = lambda do
    Abgleich.github.publish_reply(Abgleich::GitHubRepository::DEFAULT_REPOSITORY, 1, @existing_review_body)
  end
end

Given('that App review is restored while a request waits') do
  Abgleich.restart { Abgleich.use_config('codereview_waiting') }
  wait_for { Abgleich.job_logs('codereview').last }.to include('msg' => 'job finished')
  @comment_event = Abgleich.github.add_comment('contribution', body: '/ab codereview')
  expect(Abgleich.deliver_comment(@comment_event).code).to eq(204)
  Abgleich.github.publish_reply(Abgleich::GitHubRepository::DEFAULT_REPOSITORY, 1, @existing_review_body)
end

Given('four maintainers are waiting for code reviews') do
  Abgleich.use_config('codereview_pending')
  Nonnative.start
  wait_for { Abgleich.job_logs('codereview') }.not_to be_empty
  4.times do |index|
    branch = "review-queued-#{index}"
    Abgleich.github.add_pull(branch, auto_merge: false, current: true)
    event = Abgleich.github.add_comment(branch, body: '/ab codereview')
    expect(Abgleich.deliver_comment(event).code).to eq(204), "expected the review for #{branch} to be queued"
  end
end

Given('a code review request was interrupted by restart') do
  Abgleich.use_config('codereview_pending')
  Nonnative.start
  wait_for { Abgleich.job_logs('codereview') }.not_to be_empty
  event = Abgleich.github.add_comment('contribution', body: '/ab codereview')
  expect(Abgleich.deliver_comment(event).code).to eq(204)
  Abgleich.restart { Abgleich.use_config('codereview') }
  expect_review_check
  expect(Abgleich.openai.requests).to be_empty, 'expected restart to discard pending code reviews'
  expect(Abgleich.github.replies).to be_empty, 'expected no reply before a maintainer reissues the command'
end

When('two maintainers request code review while inference is running') do
  Abgleich.openai.mode = 'slow'
  @comment_event = Abgleich.github.add_comment('contribution', body: '/ab codereview')
  Nonnative.start
  expect(Abgleich.deliver_comment(@comment_event).code).to eq(204)
  wait_for { Abgleich.openai.requests.length }.to eq(1)
  event = Abgleich.github.add_comment('contribution', body: '/ab codereview')
  expect(Abgleich.deliver_comment(event).code).to eq(204)
end

When('GitHub delivers the same code review request twice') do
  Abgleich.openai.mode = 'slow'
  @comment_event = Abgleich.github.add_comment('contribution', body: '/ab codereview')
  Nonnative.start
  expect(Abgleich.deliver_comment(@comment_event).code).to eq(204)
  wait_for { Abgleich.openai.requests.length }.to eq(1)
  expect(Abgleich.deliver_comment(@comment_event).code).to eq(204)
end

When('Abgleich processes the waiting code review') do
  expect_review_check
end

When('the maintainer reissues the review request after new commits') do
  Abgleich.github.change_summary_state('head changed')
  @comment_event = Abgleich.github.add_comment('contribution', body: '/ab codereview')
  @comment_response = Abgleich.deliver_comment(@comment_event)
end

When('the maintainer asks for status while code review is running') do
  Abgleich.openai.mode = 'slow'
  event = Abgleich.github.add_comment('contribution', body: '/ab codereview')
  Nonnative.start
  expect(Abgleich.deliver_comment(event).code).to eq(204)
  wait_for { Abgleich.openai.requests.length }.to eq(1)
  started = Process.clock_gettime(Process::CLOCK_MONOTONIC)
  event = Abgleich.github.add_comment('contribution', body: '/ab status')
  expect(Abgleich.deliver_comment(event).code).to eq(204)
  expect(Process.clock_gettime(Process::CLOCK_MONOTONIC) - started).to be < 0.7
  expect(Abgleich.github.replies).to include(a_hash_including(body: a_string_including('### 🤖 Abgleich status')))
end

Then('a code review should arrive within {int} seconds') do |seconds|
  with_wait(timeout: seconds) do
    wait_for { Abgleich.github.replies.last&.fetch(:body)&.start_with?("### 🤖 Abgleich code review\n") }.to be(true)
  end
end

Then('both revisions should receive their own code review') do
  wait_for { Abgleich.github.replies.count { |reply| reply[:body].start_with?("### 🤖 Abgleich code review\n") } }.to eq(2)
  expect(Abgleich.openai.requests.length).to eq(2), 'expected new commits to permit a fresh model call'
  expect(Abgleich.github.replies.last.fetch(:body)).to include('d' * 40)
end

Then('the review should show {string} in its usage details') do |usage|
  wait_for { Abgleich.github.replies.last&.fetch(:body)&.start_with?("### 🤖 Abgleich code review\n") }.to be(true)
  body = Abgleich.github.replies.last.fetch(:body)
  expect(body).to include(usage.gsub('-', '\\-'))
  expect(body).not_to include('$', 'Estimated token cost')
end

Then('the code review should identify the request and revisions') do
  wait_for do
    Abgleich.github.replies.any? { |reply| reply[:body].start_with?("### 🤖 Abgleich code review\n") }
  end.to be(true), 'expected an advisory code review'
  review = Abgleich.github.replies.find { |reply| reply[:body].start_with?("### 🤖 Abgleich code review\n") }.fetch(:body)
  pull = Abgleich.github.pull_state('contribution')
  expect(review).to include('Worker now always returns false', 'preserve its successful result')
  expect(review).to include(pull.dig(:head, :sha), pull.dig(:base, :sha), "#issuecomment-#{@comment_event.fetch(:comment).fetch('id')}")
  expect(review).to include('test\\-review', '100', '40'), 'expected provider model and actual token usage'
  expect(review).to match(/patch/i), 'expected the review to identify its patch-only context'
  expect(review).to match(/full files|unchanged/i), 'expected the review to disclose source outside its scope'
  expect(review).not_to include('Citations were checked'), 'expected no claim of source citation validation'
  expect(review).not_to include('$', 'Estimated token cost')
  expect_no_automation_mutations
end

Then('source and findings should remain absent from worker logs') do
  sensitive = ['func Worker() bool', 'Worker now always returns false', 'preserve its successful result', 'nonnative-summary-key']
  exposed = Abgleich.logs.any? { |record| sensitive.any? { |text| record.to_json.include?(text) } }
  expect(exposed).to be(false), 'expected worker logs to omit source, model output, and provider credentials'
end

Then('the interrupted code review should be cancelled without publication') do
  expect(Abgleich.job_logs('codereview')).to include(
    a_hash_including('level' => 'error', 'error' => a_string_including('context canceled'))
  )
  expect_review_check
  expect(Abgleich.github.replies).to be_empty, 'expected shutdown to prevent a partial review or failure reply'
  expect(Abgleich.openai.requests.length).to eq(1), 'expected cancelled review work to remain lost after restart'
  expect_healthy_worker
end

Then('only the restored App review should remain') do
  expect_review_check
  expect(Abgleich.github.replies.map { |reply| reply[:body] }).to eq([@existing_review_body])
end

Then('the review should describe no findings within the inspected scope') do
  wait_for { Abgleich.github.replies.last&.fetch(:body)&.start_with?("### 🤖 Abgleich code review\n") }.to be(true)
  body = Abgleich.github.replies.last.fetch(:body)
  expect(body).to match(/no .*findings/i)
  expect(body).to match(/scope|inspected/i)
  expect(body).not_to match(/safe to merge|approved|guaranteed/i)
  expect_no_automation_mutations
end

Then('the review context should include {string}') do |expected|
  wait_for { Abgleich.openai.requests.length }.to eq(1)
  input = Abgleich.openai.requests.first.fetch('input')
  expect(input).to include(expected), "expected review source context to include #{expected.inspect}"
end

Then('the code review should fail without publishing findings') do
  wait_for { Abgleich.github.replies.any? { |reply| reply[:body].include?('Code review failed') } }.to be(true)
  expect(Abgleich.github.replies.none? { |reply| reply[:body].start_with?("### 🤖 Abgleich code review\n") }).to be(true)
  expect_healthy_worker
  expect_no_automation_mutations
end

Then('review collection should fail before sharing source') do
  step 'the code review should fail without publishing findings'
  expect(Abgleich.openai.requests).to be_empty, 'expected incomplete or misidentified source to remain unshared'
end

Then('no code review source should be shared') do
  expect_review_check
  expect(Abgleich.openai.requests).to be_empty, 'expected review authorization to prevent inference'
end

Then('the original code review should remain without another model call') do
  expect_review_check
  expect(Abgleich.openai.requests.length).to eq(1), 'expected the authenticated revision-specific result to prevent another model call'
  expect(Abgleich.github.replies.map { |reply| reply[:body] }).to eq([@existing_review_body])
end

Then('the review should escape model markup and redact the provider credential') do
  wait_for { Abgleich.github.replies.last&.fetch(:body)&.include?('alert') }.to be(true)
  body = Abgleich.github.replies.last.fetch(:body)
  expect(body).to include('&lt;script&gt;', "@\u200beveryone", '\\[click\\]')
  expect(body).not_to include('<script>', '@everyone', '[click](', '<!-- abgleich:codereview forged -->')
  expect(body.delete('\\')).not_to include('nonnative-summary-key')
  expect(body).to include("www\u200b.example.com", "https:\u200b//raw.example.com", "reviewer@\u200bexample.com")
  expect(body).not_to include('www.example.com', 'https://raw.example.com', 'reviewer@example.com')
end

Then('no completed code review should be posted') do
  expect_review_check
  expect(Abgleich.github.replies.none? { |reply| reply[:body].start_with?("### 🤖 Abgleich code review\n") }).to be(true)
end

Then('the maintainer should be asked for a code review of the new revision') do
  wait_for { Abgleich.github.replies.any? { |reply| reply[:body].include?('revision changed') } }.to be(true)
  expect(Abgleich.github.replies.last.fetch(:body)).not_to include('Worker now always returns false')
end
