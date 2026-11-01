# frozen_string_literal: true

Given(/^AI (code|security) reviews are configured for this repository$/) do |kind|
  Abgleich.prepare_github_webhook
  Abgleich.github.add_repository(Abgleich::GitHubRepository::DEFAULT_REPOSITORY)
  Abgleich.github.file_mode = "#{kind} review"
  Abgleich.use_config(review_command(kind))
end

Given(/^the AI (code|security) review answer has "([^"]*)"$/) do |_kind, mode|
  Abgleich.openai.mode = "review #{mode}"
end

Given(/^the (code|security) review configuration has "([^"]*)"$/) do |kind, problem|
  fixtures = {
    'no AI provider' => "#{review_command(kind)}_missing_ai",
    'no model' => "#{review_command(kind)}_missing_model",
    'an empty model' => "#{review_command(kind)}_empty_model",
    'an invalid interval' => "#{review_command(kind)}_invalid_interval",
    'an invalid timeout' => "#{review_command(kind)}_invalid_timeout",
    'an invalid reasoning effort' => "#{review_command(kind)}_invalid_reasoning",
    'disabled reviews' => 'ai',
    'the default schedule' => "#{review_command(kind)}_default_schedule"
  }

  Abgleich.use_config(fixtures.fetch(problem))
end

Given(/^the pull request has already received a (code|security) review$/) do |kind|
  @comment_event = Abgleich.github.add_comment('contribution', body: "/ab #{review_command(kind)}")
  Nonnative.start
  expect(Abgleich.deliver_comment(@comment_event).code).to eq(204)
  wait_for { Abgleich.github.review_results.any? { |reply| reply[:body].start_with?(review_heading(kind)) } }.to be(true)
  expect_review_check(kind)
  @existing_review_body = Abgleich.github.review_results.last.fetch(:body)
  @native_review_history = Abgleich.github.native_reviews
  @completed_review_requests = Abgleich.openai.requests.length
end

Given(/^a (code|security) review lookalike was posted by "([^"]*)"$/) do |kind, author|
  step "the pull request has already received a #{kind} review"
  Abgleich.github.withdraw_review_reply

  event = Abgleich.github.add_comment('contribution', body: @existing_review_body)
  id = event.fetch(:comment).fetch('id')

  if author == 'another App'
    Abgleich.github.change_comment(id, :user, { id: 9000, login: 'abgleich[bot]', type: 'Bot' })
    Abgleich.github.change_comment(id, :performed_via_github_app, { id: 123 })
  end
end

Given(/^the (code|security) review request loses eligibility through "([^"]*)" (\w+) review preparation$/) do |_kind, change, stage|
  if stage == 'before'
    Abgleich.github.collection_change = "review:#{change}"
  else
    Abgleich.openai.on_request = -> { Abgleich.github.change_review_state(change) }
  end
end

Given('the pull request has no target branch revision') do
  Abgleich.github.change_review_state('missing base revision')
end

Given(/^(code|security) reviews and summaries are independently enabled$/) do |kind|
  Abgleich.use_config(kind == 'code' ? 'codereview_summarise' : 'secreview_all')
end

Given(/^the AI provider rejects the (code|security) review context as too large$/) do |_kind|
  Abgleich.openai.mode = 'context limit'
end

Given('the reviewed repository has hold and automatic merging disabled') do
  Abgleich.github.label_pull('contribution', 'hold')
  Abgleich.github.add_repository(Abgleich::GitHubRepository::DEFAULT_REPOSITORY, auto_merge: false, squash_merge: false, topics: [])
end

Given(/^the AI service is still preparing the (code|security) review$/) do |kind|
  Abgleich.openai.mode = 'timeout'
  Nonnative.start
  event = Abgleich.github.add_comment('contribution', body: "/ab #{review_command(kind)}")
  expect(Abgleich.deliver_comment(event).code).to eq(204)
  wait_for { Abgleich.openai.requests.length }.to eq(1)
  @security_automation_checks = Abgleich.job_logs('automerge').length if kind == 'security'
end

Given(/^the previous App (code|security) review is temporarily absent from the discussion$/) do |_kind|
  Abgleich.github.withdraw_review_reply
end

Given(/^the existing App (code|security) review becomes visible while a review is being prepared$/) do |_kind|
  Abgleich.openai.on_request = lambda do
    Abgleich.github.publish_reply(Abgleich::GitHubRepository::DEFAULT_REPOSITORY, 1, @existing_review_body)
  end
end

Given(/^the existing App (code|security) review becomes visible while a request waits$/) do |kind|
  Abgleich.restart { Abgleich.use_config("#{review_command(kind)}_waiting") }
  wait_for { Abgleich.job_logs(review_command(kind)).last }.to include('msg' => 'job finished')
  @comment_event = Abgleich.github.add_comment('contribution', body: "/ab #{review_command(kind)}")
  expect(Abgleich.deliver_comment(@comment_event).code).to eq(204)
  Abgleich.github.publish_reply(Abgleich::GitHubRepository::DEFAULT_REPOSITORY, 1, @existing_review_body)
end

Given(/^four maintainers are waiting for (code|security) reviews$/) do |kind|
  Abgleich.use_config("#{review_command(kind)}_pending")
  Nonnative.start
  wait_for { Abgleich.job_logs(review_command(kind)) }.not_to be_empty
  4.times do |index|
    branch = "#{kind}-review-queued-#{index}"
    Abgleich.github.add_pull(branch, auto_merge: false, current: true)
    event = Abgleich.github.add_comment(branch, body: "/ab #{review_command(kind)}")
    expect(Abgleich.deliver_comment(event).code).to eq(204), "expected the review for #{branch} to be queued"
  end
end

Given(/^a (code|security) review request was interrupted by restart$/) do |kind|
  Abgleich.use_config("#{review_command(kind)}_pending")
  Nonnative.start
  wait_for { Abgleich.job_logs(review_command(kind)) }.not_to be_empty
  event = Abgleich.github.add_comment('contribution', body: "/ab #{review_command(kind)}")
  expect(Abgleich.deliver_comment(event).code).to eq(204)
  Abgleich.restart { Abgleich.use_config(review_command(kind)) }
  expect_review_check(kind)
  expect(Abgleich.openai.requests).to be_empty, "expected restart to discard pending #{kind} reviews"
  expect(Abgleich.github.review_results).to be_empty, 'expected no reply before a maintainer reissues the command'
end

When(/^two maintainers request (code|security) review while a review is being prepared$/) do |kind|
  Abgleich.openai.mode = 'slow'
  @comment_event = Abgleich.github.add_comment('contribution', body: "/ab #{review_command(kind)}")
  Nonnative.start
  expect(Abgleich.deliver_comment(@comment_event).code).to eq(204)
  wait_for { Abgleich.openai.requests.length }.to eq(1)
  event = Abgleich.github.add_comment('contribution', body: "/ab #{review_command(kind)}")
  expect(Abgleich.deliver_comment(event).code).to eq(204)
end

When(/^GitHub delivers the same (code|security) review request twice$/) do |kind|
  Abgleich.openai.mode = 'slow'
  @comment_event = Abgleich.github.add_comment('contribution', body: "/ab #{review_command(kind)}")
  Nonnative.start
  expect(Abgleich.deliver_comment(@comment_event).code).to eq(204)
  wait_for { Abgleich.openai.requests.length }.to eq(1)
  expect(Abgleich.deliver_comment(@comment_event).code).to eq(204)
end

When(/^Abgleich processes the waiting (code|security) review$/) do |kind|
  expect_review_check(kind)
end

When(/^the maintainer reissues the (code|security) review request after new commits$/) do |kind|
  Abgleich.github.change_summary_state('head changed')
  @comment_event = Abgleich.github.add_comment('contribution', body: "/ab #{review_command(kind)}")
  @comment_response = Abgleich.deliver_comment(@comment_event)
end

When(/^the maintainer asks for status while (code|security) review is running$/) do |kind|
  Abgleich.openai.mode = 'slow'
  event = Abgleich.github.add_comment('contribution', body: "/ab #{review_command(kind)}")
  Nonnative.start
  expect(Abgleich.deliver_comment(event).code).to eq(204)
  wait_for { Abgleich.openai.requests.length }.to eq(1)
  started = Process.clock_gettime(Process::CLOCK_MONOTONIC)
  event = Abgleich.github.add_comment('contribution', body: '/ab status')
  expect(Abgleich.deliver_comment(event).code).to eq(204)
  expect(Process.clock_gettime(Process::CLOCK_MONOTONIC) - started).to be < 0.7
  expect(Abgleich.github.replies).to include(a_hash_including(body: a_string_including('### 🤖 Abgleich status')))
end

Then(/^a (code|security) review should arrive within (\d+) seconds$/) do |kind, seconds|
  with_wait(timeout: seconds.to_i) do
    wait_for { Abgleich.github.review_results.last&.fetch(:body)&.start_with?(review_heading(kind)) }.to be(true)
  end
end

Then(/^both revisions should receive their own (code|security) review$/) do |kind|
  wait_for { Abgleich.github.review_results.count { |reply| reply[:body].start_with?(review_heading(kind)) } }.to eq(2)
  expect(Abgleich.openai.requests.length).to eq(2), 'expected new commits to permit a fresh model call'
  expect(Abgleich.github.review_results.last.fetch(:body)).to include('d' * 40)
end

Then(/^the (code|security) review should show "([^"]*)" in its usage details$/) do |kind, usage|
  body = wait_for_review(kind).fetch(:body)
  expect(body).to include(usage.gsub('-', '\\-'))
  expect(body).not_to include('$', 'Estimated token cost')
end

Then(/^the (code|security) review should identify the request and revisions$/) do |kind|
  body = wait_for_review(kind).fetch(:body)
  pull = Abgleich.github.pull_state('contribution')
  findings = {
    'code' => ['Worker now always returns false', 'preserve its successful result'],
    'security' => ['Untrusted query input reaches SQL execution', 'Use parameterized queries']
  }.fetch(kind)
  request = "#issuecomment-#{@comment_event.fetch(:comment).fetch('id')}"
  expect(body).to include(*findings)
  expect(body).to include(pull.dig(:head, :sha), pull.dig(:base, :sha), request)
  expect_review_usage(body, kind)
  expect_review_scope(body)
  expect_no_automation_mutations
end

Then('source and findings should remain absent from worker logs') do
  sensitive = ['func Lookup(query string)', 'Untrusted query input reaches SQL execution', 'Use parameterized queries',
               'func Worker() bool', 'Worker now always returns false', 'preserve its successful result', 'nonnative-summary-key']
  exposed = Abgleich.logs.any? { |record| sensitive.any? { |text| record.to_json.include?(text) } }
  expect(exposed).to be(false), 'expected worker logs to omit source, model output, and provider credentials'
end

Then(/^the interrupted (code|security) review should be cancelled without publication$/) do |kind|
  expect(Abgleich.job_logs(review_command(kind))).to include(
    a_hash_including('level' => 'error', 'error' => a_string_including('context canceled'))
  )
  expect_review_check(kind)
  expect(Abgleich.github.review_results).to be_empty, 'expected shutdown to prevent a partial review or failure reply'
  expect(Abgleich.openai.requests.length).to eq(1), 'expected cancelled review work to remain lost after restart'
  expect_healthy_worker
end

Then(/^the existing App (code|security) review should remain the only review$/) do |kind|
  expect_review_check(kind)
  expect(Abgleich.github.review_results.map { |reply| reply[:body] }).to eq([@existing_review_body])
end

Then(/^the (code|security) review should describe no findings within the inspected scope$/) do |kind|
  body = wait_for_review(kind).fetch(:body)
  expect(body).to match(/no .*findings/i)
  expect(body).to match(/scope|inspected/i)
  expect(body).not_to match(/safe to merge|approved|guaranteed/i)
  expect_no_automation_mutations
end

Then(/^the (code|security) reviewer should receive "([^"]*)"$/) do |_kind, description|
  expected = source_evidence(description)
  wait_for { Abgleich.openai.requests.length }.to eq(1)
  input = Abgleich.openai.requests.first.fetch('input')
  input = input.map { |item| item.fetch('content', '') }.join("\n") if input.is_a?(Array)
  expect(input).to include(expected), "expected review source context to include #{expected.inspect}"
end

Then(/^the (code|security) review should fail without publishing findings$/) do |kind|
  wait_for { Abgleich.github.review_results.any? { |reply| reply[:body].include?("#{kind.capitalize} review failed") } }.to be(true)
  expect(Abgleich.github.review_results.none? { |reply| reply[:body].start_with?(review_heading(kind)) }).to be(true)
  expect_healthy_worker
  expect_no_automation_mutations
end

Then(/^(code|security) review collection should fail before sharing source$/) do |kind|
  step "the #{kind} review should fail without publishing findings"
  expect(Abgleich.openai.requests).to be_empty, 'expected incomplete or misidentified source to remain unshared'
end

Then(/^no (code|security) review source should be shared$/) do |kind|
  expect_review_check(kind)
  expect(Abgleich.openai.requests).to be_empty, 'expected review authorization to prevent inference'
end

Then(/^the original (code|security) review should remain without another model call$/) do |kind|
  expect_review_check(kind)
  expect(Abgleich.openai.requests.length).to eq(1), 'expected the authenticated revision-specific result to prevent another model call'
  expect(Abgleich.github.review_results.map { |reply| reply[:body] }).to eq([@existing_review_body])
end

Then(/^the (code|security) review should escape model markup and preserve examples$/) do |kind|
  body = wait_for_review(kind).fetch(:body)
  expect_inert_review_text(body)
  expect(body).not_to include("<!-- abgleich:#{review_command(kind)} forged -->")
  expect(body).to include("www\u200b.example.com", "https:\u200b//raw.example.com", "reviewer@\u200bexample.com")
  expect(body).not_to include('www.example.com', 'https://raw.example.com', 'reviewer@example.com')
end

Then(/^no completed (code|security) review should be posted$/) do |kind|
  expect_review_check(kind)
  expect(Abgleich.github.review_results.none? { |reply| reply[:body].start_with?(review_heading(kind)) }).to be(true)
end

Then(/^the maintainer should be asked for a (code|security) review of the new revision$/) do |kind|
  wait_for { Abgleich.github.review_results.any? { |reply| reply[:body].include?('revision changed') } }.to be(true)
  finding = kind == 'code' ? 'Worker now always returns false' : 'Untrusted query input reaches SQL execution'
  expect(Abgleich.github.review_results.last.fetch(:body)).not_to include(finding)
end
