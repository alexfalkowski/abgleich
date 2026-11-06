# frozen_string_literal: true

Given('both review commands are enabled') do
  Abgleich.use_config('secreview_all')
end

Given('the review cannot inspect {string}') do |context|
  case context
  when 'changed-file patch'
    Abgleich.github.file_mode = 'missing patch'
  when 'changed-file list'
    Abgleich.github.file_mode = 'no changed files'
  when 'patches and binary source'
    Abgleich.github.file_mode = 'missing patch'
    Abgleich.github.fail_with('source binary')
  else
    Abgleich.github.fail_with('source binary')
  end
end

Given(/^the (code|security) review can inspect the missing context again$/) do |kind|
  Abgleich.github.file_mode = "#{kind} review"
end

Given('the reviewed contribution has new commits') do
  Abgleich.github.change_summary_state('head changed')
end

Given('a maintainer dismissed the latest review findings') do
  Abgleich.github.change_native_review(:state, 'DISMISSED')
end

Given('GitHub returns a different review verdict') do
  Abgleich.github.fail_with('review publication:wrong state')
end

Given('both reviews previously requested changes before a clean code review of new commits') do
  complete_review_request('code')
  complete_review_request('security')
  Abgleich.github.change_summary_state('head changed')
  Abgleich.openai.mode = 'review no findings'
  complete_review_request('code')
  expect(reviews_for('code').last).to include(state: 'COMMENTED')
end

When('maintainers request both reviews together') do
  Abgleich.openai.pause_responses
  queue_review('code')
  queue_review('security')
  wait_for { Abgleich.openai.requests.length }.to eq(2), 'expected both independent reviews to reach inference'
  Abgleich.openai.resume_responses
end

Then(/^the (code|security) review decision should be "([^"]*)"$/) do |kind, decision|
  pull = Abgleich.github.pull_state('contribution')
  expected = { state: decision, commit_id: pull.dig(:head, :sha) }
  wait_for { reviews_for(kind).last&.slice(:state, :commit_id) }.to eq(expected), 'expected the requested verdict for the current head'
  review = reviews_for(kind).last
  expect(review).to include(state: decision, commit_id: pull.dig(:head, :sha))
  text = { 'APPROVED' => 'Approve.', 'CHANGES_REQUESTED' => 'Request changes.', 'COMMENTED' => 'Comment.' }.fetch(decision)
  emoji = { 'APPROVED' => '✅', 'CHANGES_REQUESTED' => '❗', 'COMMENTED' => '⚠️' }.fetch(decision)
  expect(review.fetch(:body)).to include("#{emoji} **Decision:** #{text}", '**Assessment:**', '📝 **Overview**')
  expect_review_check(kind)
end

Then('its decision should explain that the other review still requests changes') do
  expect(Abgleich.github.native_reviews.last.fetch(:body)).to include('Approval withheld because the other review still requests changes')
end

Then(/^the (code|security) review assessment should be "([^"]*)"$/) do |kind, assessment|
  body = wait_for_review(kind).fetch(:body)
  expect(body).to include("**Assessment:** #{assessment}.")
  expect(body).not_to include('**Assessment:** Incomplete review.')
end

Then(/^the (code|security) review decision should explain "([^"]*)"$/) do |kind, reason|
  body = review_text(kind)
  expect(body.split('📝 **Overview**').first).to include(reason)
end

Then(/^the (code|security) review should identify "([^"]*)" before its overview$/) do |kind, limitation|
  body = review_text(kind)
  expect(body.split('📝 **Overview**').first).to include('Limitations', limitation)
end

Then(/^the (code|security) review overview should explain remaining work$/) do |kind|
  overview = review_text(kind).split('📝 **Overview**').last.split('<details>').first
  expect(overview).to include('Changed callers remain unreviewed.')
end

Then("the clean security review should preserve the code review's decision") do
  wait_for_review('security')
  reviews = Abgleich.github.native_reviews
  changes = reviews.index { |review| review[:state] == 'CHANGES_REQUESTED' }
  expect(reviews.drop(changes + 1)).not_to include(a_hash_including(state: 'APPROVED'))
  expect(reviews_for('security').last.fetch(:body)).to include('**Assessment:** No findings.')
end

Then('no review should change merge automation') do
  expect_no_automation_mutations
end

Then('the incomplete review should have no success marker') do
  expect(Abgleich.github.native_reviews.first.fetch(:body)).not_to include('<!-- abgleich:')
end

Then(/^the (code|security) review publication should be reported as failed$/) do |kind|
  wait_for { Abgleich.job_logs(review_command(kind)) }.to include(
    a_hash_including('level' => 'error', 'error' => a_string_including('mismatched content or submission state'))
  )
  expect_review_check(kind)
  expect(Abgleich.logs).not_to include(a_hash_including('msg' => 'published pull request review'))
  expect(Abgleich.github.native_reviews.length).to eq(1), 'expected no automatic retry of review publication'
end
