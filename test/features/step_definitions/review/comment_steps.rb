# frozen_string_literal: true

Given('the reviewer reports {string}') do |description|
  modes = {
    'findings on added and removed lines' => 'inline findings',
    'a finding outside the changed lines' => 'mixed locations',
    'no supported findings' => 'no findings',
    'an unreadable answer' => 'malformed JSON',
    'examples with untrusted markup' => 'hostile text'
  }
  Abgleich.openai.mode = "review #{modes.fetch(description)}"
end

Given('the contribution renames a reviewed file') do
  Abgleich.github.file_mode = 'renamed review file'
end

Given('a legacy App code review exists for the current revision') do
  complete_review_request('code')
  Abgleich.github.withdraw_review_reply
  Abgleich.github.publish_reply(Abgleich::GitHubRepository::DEFAULT_REPOSITORY, 1, @existing_review_body)
end

Given('a native code review lookalike was posted by {string}') do |author|
  complete_review_request('code')
  Abgleich.github.change_native_review_author(author)
end

Given('the pull request has 105 earlier native reviews') do
  pull = Abgleich.github.pull_state('contribution')
  105.times do
    Abgleich.github.publish_review(Abgleich::GitHubRepository::DEFAULT_REPOSITORY, pull.fetch(:number),
                                   { 'body' => 'Earlier discussion', 'commit_id' => pull.dig(:head, :sha) })
  end
end

Given('native review history returns {string} on page {int}') do |response, page|
  @native_review_history = Abgleich.github.native_reviews
  Abgleich.github.fail_with("review history:#{page}:#{response}")
end

Given(/^105 native (code|security) review lookalikes were posted by "([^"]*)"$/) do |kind, author|
  complete_review_request(kind)
  Abgleich.github.change_native_review_author(author)
  pull = Abgleich.github.pull_state('contribution')
  104.times do
    Abgleich.github.publish_review(Abgleich::GitHubRepository::DEFAULT_REPOSITORY, pull.fetch(:number),
                                   { 'body' => @existing_review_body, 'commit_id' => pull.dig(:head, :sha) })
    Abgleich.github.change_native_review_author(author)
  end
  @review_kind = kind
  @review_request_counts = Abgleich::GitHubService.request_counts
end

Given('native review authorship returns {string}') do |problem|
  @native_review_history = Abgleich.github.native_reviews
  Abgleich.github.fail_with("review authorship:#{problem}")
end

Then(/^the (code|security) review should contain one overview$/) do |kind|
  wait_for_review(kind)
  reviews = Abgleich.github.native_reviews.select { |review| review[:body].start_with?(review_heading(kind)) }
  expect(reviews.length).to eq(1), 'expected one submitted review containing every finding'
  review = reviews.first
  pull = Abgleich.github.pull_state('contribution')
  expect(review).to include(state: 'CHANGES_REQUESTED', commit_id: pull.dig(:head, :sha))
  expect(review.fetch(:body)).to include(pull.dig(:head, :sha), pull.dig(:base, :sha), '#issuecomment-', ':v2:')
  expect_no_automation_mutations
end

Then('the inline findings should be:') do |table|
  comments = Abgleich.github.native_reviews.last.fetch(:comments)
  actual = comments.map { |comment| comment.slice(:path, :line, :side).transform_keys(&:to_s) }
  expected = table.hashes.map { |row| row.merge('line' => row.fetch('line').to_i) }
  expect(actual).to eq(expected), 'expected each finding beside the requested changed line'
  expect(comments).to all(include(body: a_string_matching(/work/)))
end

Then('no successful review should be posted as an ordinary comment') do
  expect(Abgleich.github.replies).to be_empty, 'expected the native review to own the overview'
end

Then('its overview should retain the finding at the unavailable location') do
  body = Abgleich.github.native_reviews.last.fetch(:body).delete('\\')
  expect(body).to include('The unavailable location needs inspection.', 'renamed.go', '3', 'RIGHT')
end

Then('the native review should have no inline findings') do
  expect(Abgleich.github.native_reviews.last.fetch(:comments)).to be_empty
end

Then('no native review should be submitted') do
  expect(Abgleich.github.native_reviews).to be_empty
end

Then('native review history should remain unchanged') do
  expect(Abgleich.github.native_reviews).to eq(@native_review_history), 'expected no new review after incomplete history'
end

Then('author checks should stay within the history request budget') do
  expect_review_check(@review_kind)
  counts = Abgleich::GitHubService.request_counts
  history_reads = counts.fetch(:review_history_reads) - @review_request_counts.fetch(:review_history_reads)
  author_checks = counts.fetch(:review_authorship_requests) - @review_request_counts.fetch(:review_authorship_requests)
  expect(author_checks).to be <= history_reads + 1, "expected author checks bounded by #{history_reads} history pages, got #{author_checks}"
end

Then('every inline finding should escape model markup and preserve examples') do
  comments = Abgleich.github.native_reviews.last.fetch(:comments)
  expect(comments.length).to eq(1), 'expected the hostile finding to remain visible inline'
  expect_inert_review_text(comments.first.fetch(:body))
end

Then('a fresh App review should identify the current request') do
  wait_for { Abgleich.github.native_reviews.count { |review| review[:viewer_did_author] } }.to eq(1)
  review = Abgleich.github.native_reviews.find { |item| item[:viewer_did_author] }
  expect(review.fetch(:body)).to include("#issuecomment-#{@comment_event.fetch(:comment).fetch('id')}")
end

Then('the existing native review should prevent another model call') do
  expect_review_check('code')
  expect(Abgleich.openai.requests.length).to eq(@completed_review_requests), 'expected history to suppress repeated inference'
  expect(Abgleich.github.native_reviews).to eq(@native_review_history), 'expected no duplicate native review'
end

Then('both review commands should have their own conversations') do
  wait_for { Abgleich.github.native_reviews.length }.to eq(2)
  %w[code security].each do |kind|
    review = reviews_for(kind).last
    expect(review.fetch(:comments).length).to eq(2), "expected two separate #{kind} conversations"
  end
  expect(Abgleich.openai.requests.length).to eq(2)
end

Given('the pull request has these existing review conversations:') do |table|
  table.hashes.each do |row|
    Abgleich.github.seed_review_thread(row.fetch('conversation'), kind: row.fetch('kind'), state: row.fetch('state'))
  end
end

Given('100 current conversations precede an outdated App conversation') do
  100.times { |index| Abgleich.github.seed_review_thread("current-#{index}", state: 'current') }
  Abgleich.github.seed_review_thread('older-page', state: 'current')
end

Given('conversation {string} becomes {string} before resolution') do |_name, state|
  changes = { 'current' => { isOutdated: false }, 'resolved' => { isResolved: true },
              'cannot resolve' => { viewerCanResolve: false } }
  Abgleich.github.thread_recheck = changes.fetch(state)
end

Given('GitHub cannot resolve conversation {string} while other conversations progress') do |name|
  Abgleich.github.thread_failure = name
  Nonnative.start
  expect_failed_check('codereview')
  wait_for { Abgleich.github.review_threads.find { |thread| thread[:id] == 'available' }.fetch(:isResolved) }.to be(true)
  thread = Abgleich.github.review_threads.find { |item| item[:id] == name }
  expect(thread.fetch(:isResolved)).to be(false), 'expected the failed conversation to remain open'
  expect_healthy_worker
end

Given('GitHub loses the response after resolving a conversation') do
  Abgleich.github.thread_failure = 'lost response'
end

When('new commits make only conversation {string} outdated') do |name|
  Abgleich.github.change_summary_state('head changed')
  Abgleich.github.change_review_thread(name, isOutdated: true)
  Nonnative.start
end

When('scheduled review maintenance runs') do
  Nonnative.start
end

When('the worker restarts with those conversations already on GitHub') do
  Nonnative.start
  Abgleich.restart { Abgleich.github.change_review_thread('older-page', isOutdated: true) }
end

When('GitHub becomes able to resolve conversations again') do
  Abgleich.github.thread_failure = nil
end

Then('review conversations should be:') do |table|
  table.hashes.each do |row|
    name = row.fetch('conversation')
    expected = row.fetch('resolved') == 'true'
    wait_for do
      Abgleich.github.review_threads.find { |thread| thread[:id] == name }.fetch(:isResolved)
    end.to be(expected), "expected conversation #{name} resolved=#{expected}"
  end
end

Then('conversation {string} should receive exactly {int} resolution request across later checks') do |name, count|
  kind = Abgleich.server_config_path.include?('secreview') ? 'security' : 'code'
  2.times { expect_review_check(kind) }
  expect(Abgleich.github.thread_resolution_requests.count(name)).to eq(count), "expected #{count} resolution request for #{name}"
end

Then('only conversation {string} should receive a resolution request') do |name|
  expect_review_check('code')
  expect(Abgleich.github.thread_resolution_requests).to eq([name]), 'expected only the eligible App conversation to be resolved'
end

Then('conversation {string} should receive no resolution requests') do |name|
  expect_review_check('code')
  expect(Abgleich.github.thread_resolution_requests).not_to include(name), "expected no resolution writes for conversation #{name}"
end

Then('no AI inference should be requested for conversation maintenance') do
  kind = Abgleich.server_config_path.include?('secreview') ? 'security' : 'code'
  expect_review_check(kind)
  expect(Abgleich.openai.requests).to be_empty, 'expected existing conversations to be maintained without AI inference'
end
