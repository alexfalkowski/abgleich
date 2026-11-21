# frozen_string_literal: true

require 'digest'

Given('both review commands are enabled') do
  Abgleich.use_config('secreview_all')
end

Given('all three review commands are enabled') do
  Abgleich.use_config('testreview_all')
end

Given('the PR description {string} a browser check') do |requirement|
  required_check = Abgleich::OpenAIService::COVERAGE_REQUIREMENT
  line = {
    'requires' => required_check,
    'requires a specific variant of' => "#{required_check} Exercise both success and recovery.",
    'suggests' => Abgleich::OpenAIService::COVERAGE_SUGGESTION,
    'mentions a declaration for' => "Example only: #{required_check}",
    'has a hidden template example for' => "<!--\n#{required_check}\n-->",
    'has an inline template example for' => "Template example <!--\n#{required_check}\n-->",
    'has a fenced example for' => "```text\n#{required_check}\n```",
    'has a tilde-fenced example for' => "~~~~\n#{required_check}\n~~~~",
    'has an indented example for' => "    #{required_check}",
    'hides an example and then requires' => "<!--\nTemplate example\n-->\n\n#{required_check}"
  }.fetch(requirement)
  description = "Update the loading state.\n\n#{line}\n\nExisting checks cover the status code."
  Abgleich.github.edit_pull_description('contribution', description)
end

Given('the author removes the browser requirement before source submission') do
  Abgleich.github.collection_change = -> { Abgleich.github.edit_pull_description('contribution', Abgleich::OpenAIService::COVERAGE_SUGGESTION) }
end

Given('the author removes the browser requirement during review') do
  Abgleich.openai.on_request = lambda do
    Abgleich.github.edit_pull_description('contribution', Abgleich::OpenAIService::COVERAGE_SUGGESTION)
  end
end

Given('the author edits the description while the worker checks prior review authorship') do
  Abgleich.github.review_authorship_change = lambda do
    Abgleich.github.edit_pull_description('contribution', Abgleich::OpenAIService::COVERAGE_REQUIREMENT)
  end
end

Given('a code review of the original description is in progress') do
  Abgleich.openai.pause_responses
  @stale_review_request = queue_review('code')
  wait_for { Abgleich.openai.requests.length }.to eq(1), 'expected inference to start before editing the description'
end

Given('the pull request has these earlier review assessments:') do |table|
  table.hashes.each do |row|
    Abgleich.openai.mode = "review #{row.fetch('answer')}"
    complete_review_request(row.fetch('kind'))
  end
end

Given('the worker restarts with only code reviews enabled') do
  Abgleich.restart { Abgleich.use_config('codereview') }
end

Given('a retired review feature has outstanding findings') do
  pull = Abgleich.github.pull_state('contribution')
  body = "Historical review\n<!-- abgleich:retiredreview:v2:#{'a' * 64} -->"
  Abgleich.github.publish_review(Abgleich::GitHubRepository::DEFAULT_REPOSITORY, pull.fetch(:number),
                                 { 'body' => body, 'commit_id' => pull.dig(:head, :sha), 'event' => 'REQUEST_CHANGES' })
end

Given('an earlier security review requested changes for a low-severity finding before the upgrade') do
  pull = Abgleich.github.pull_state('contribution')
  repository = Abgleich::GitHubRepository::DEFAULT_REPOSITORY
  identity = ['secreview', repository.downcase, pull.fetch(:number), pull.fetch(:node_id), pull.dig(:head, :sha), pull.dig(:base, :sha)]
  digest = Digest::SHA256.hexdigest(identity.join("\n"))
  body = <<~BODY
    ### 🤖 Abgleich security review

    **Assessment:** Findings.

    ❗ **Decision:** Request changes.

    🐛 **P3 · Clarify the status label**

    <!-- abgleich:secreview:v2:#{digest} -->
  BODY
  payload = { 'body' => body, 'commit_id' => pull.dig(:head, :sha), 'event' => 'REQUEST_CHANGES' }
  @earlier_security_review = Abgleich.github.publish_review(repository, pull.fetch(:number), payload)
  @native_review_history = Abgleich.github.native_reviews
end

Given('the review cannot inspect {string}') do |context|
  case context
  when 'changed-file patch'
    Abgleich.github.file_mode = 'missing patch'
  when 'changed-file list'
    Abgleich.github.file_mode = 'no changed files'
  when 'patches and binary source'
    Abgleich.github.file_mode = 'missing patches with binary file'
  else
    Abgleich.github.fail_with('source binary')
  end
end

Given('the contribution has a {string} binary file') do |status|
  Abgleich.github.file_mode = status == 'deleted' ? 'removed binary file' : 'mixed patches'
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

Given('both reviews previously requested changes before a code review of new commits reported {string}') do |answer|
  complete_review_request('code')
  complete_review_request('security')
  Abgleich.github.change_summary_state('head changed')
  Abgleich.openai.mode = "review #{answer}"
  complete_review_request('code')
  expect(reviews_for('code').last).to include(state: 'COMMENTED')
end

Given('the clearing code review used the pre-upgrade assessment label') do
  body = reviews_for('code').last.fetch(:body)
  expect(body).to include('**Assessment:** Advisory feedback.')
  Abgleich.github.change_native_review(:body, body.sub('**Assessment:** Advisory feedback.', '**Assessment:** Low-severity findings.'))
end

When('maintainers request both reviews together') do
  Abgleich.openai.pause_responses
  queue_review('code')
  queue_review('security')
  wait_for { Abgleich.openai.requests.length }.to eq(2), 'expected both independent reviews to reach inference'
  Abgleich.openai.resume_responses
end

When('a maintainer removes the browser requirement and requests another code review') do
  Abgleich.github.edit_pull_description('contribution', Abgleich::OpenAIService::COVERAGE_SUGGESTION)
  @comment_event = queue_review('code')
  Abgleich.openai.resume_responses
end

When('the author removes the browser requirement before the queued review starts') do
  Abgleich.github.edit_pull_description('contribution', Abgleich::OpenAIService::COVERAGE_SUGGESTION)
  Abgleich.openai.resume_responses
end

Then(/^the (code|security|test) review decision should be "([^"]*)"$/) do |kind, decision|
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

Then(/^the (code|security) review should fail because authorship cannot be verified$/) do |kind|
  wait_for { Abgleich.job_logs(review_command(kind)) }.to include(
    a_hash_including('level' => 'error', 'error' => a_string_including('review authorship returned an incomplete batch'))
  ), "expected the #{kind} review to finish with an authorship verification failure"
  expect(reviews_for(kind)).to be_empty, "expected no completed #{kind} review after unverified authorship"
end

Then('its decision should explain that the other review still requests changes') do
  expect(Abgleich.github.native_reviews.last.fetch(:body)).to include('Approval withheld because the other review still requests changes')
end

Then('the review overview should show the advisory about browser coverage') do
  overview = review_text('code').split('📝 **Overview**').last.split('<details>').first
  expect(overview).to include('Advisory feedback', 'P2 · Exercise loading in a browser')
end

Then('the coverage finding should open {int} inline conversations') do |expected|
  comments = wait_for_review('code').fetch(:comments)
  expect(comments.length).to eq(expected)
end

Then('the maintainer should be asked for a new review of the updated description') do
  event = @stale_review_request || @comment_event
  request = "#issuecomment-#{event.fetch(:comment).fetch('id')}"
  notice = nil
  wait_for do
    notice = Abgleich.github.replies.map { |reply| reply.fetch(:body) }.find do |body|
      body.include?('description changed') && body.include?(request)
    end
  end.not_to be_nil, 'expected a stale notice for the original review request'
  expect(notice).to include('discarded', 'Post a new `/ab codereview`')
  expect(notice).not_to include('<!-- abgleich:')
  expect_review_check('code')
end

Then('only the fresh code review should publish a verdict') do
  review = wait_for_requested_review('code', @comment_event)
  expect(review.fetch(:state)).to eq('APPROVED')
  expect(reviews_for('code')).to contain_exactly(review)
  requests = Abgleich.openai.requests
  expect(requests.length).to eq(2), 'expected inference for both descriptions'
  expect(requests.first.fetch('input').to_json).to include(Abgleich::OpenAIService::COVERAGE_REQUIREMENT)
  expect(requests.last.fetch('input').to_json).to include(Abgleich::OpenAIService::COVERAGE_SUGGESTION)
end

Then('no native review should be submitted for the contribution') do
  expect(Abgleich.github.native_reviews_for('contribution')).to be_empty
end

Then(/^the (code|security|test) review assessment should be "([^"]*)"$/) do |kind, assessment|
  body = wait_for_review(kind).fetch(:body)
  expect(body).to include("**Assessment:** #{assessment}.")
  expect(body).not_to include('**Assessment:** Incomplete review.')
end

Then(/^the (code|security|test) review decision should explain "([^"]*)"$/) do |kind, reason|
  body = review_text(kind)
  expect(body.split('📝 **Overview**').first).to include(reason)
end

Then('the review overview should show the low-severity finding at {string}') do |location|
  overview = Abgleich.github.native_reviews.last.fetch(:body).split('📝 **Overview**').last.split('<details>').first.delete('\\')
  expect(overview.lines.map(&:chomp)).to include("📍 **Reported location:** #{location}.", '🐛 **P3 · Clarify the status label**')
end

Then('the earlier security review should still request changes') do
  expect(reviews_for('security')).to contain_exactly(@earlier_security_review.merge(state: 'CHANGES_REQUESTED'))
end

Then('reusing the earlier security review should need no model call') do
  # The first completion may belong to a job that drained its queue before delivery.
  2.times { expect_review_check('security') }
  expect(Abgleich.openai.requests).to be_empty, 'expected the stored security review to prevent inference'
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
