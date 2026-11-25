# frozen_string_literal: true

Given('the pull request received inline findings that are {string} after the rebase') do |outcome|
  Abgleich.openai.mode = 'review inline findings'
  complete_review_request('code')
  @original_review = Abgleich.github.native_reviews.last
  pull = Abgleich.github.pull_state('contribution')
  @original_revisions = [pull.dig(:head, :sha), pull.dig(:base, :sha)]
  @original_finding_ids = inline_finding_ids(@original_review)
  expect(@original_finding_ids.length).to eq(2), 'expected two earlier inline blockers'
  Abgleich.openai.mode = "review follow-up #{outcome}"
end

Given('an earlier follow-up on the rebased contribution found a new blocker and left one previous blocker unverified') do
  rebase_reviewed_contribution
  complete_review_request('code')
  partial_review = Abgleich.github.native_reviews.last
  expect(partial_review.fetch(:state)).to eq('CHANGES_REQUESTED')
  expect(partial_review.fetch(:body)).to include('**Previous findings still unverified**', "`#{@original_finding_ids.last}`")
  expect(partial_review.fetch(:comments).map { |comment| comment.fetch(:path) }).to eq(['worker_test.go'])
  @partial_finding_ids = inline_finding_ids(partial_review)
end

Given('GitHub redelivered the same request and the reviewer reported a different new blocker') do
  previous_id = Abgleich.github.native_reviews.last.fetch(:id)
  Abgleich.openai.mode = 'review follow-up partially verified after redelivery'
  expect(Abgleich.deliver_comment(@comment_event).code).to eq(204)
  wait_for { Abgleich.github.native_reviews.last.fetch(:id) }.to be > previous_id
  redelivered_review = Abgleich.github.native_reviews.last
  expect(redelivered_review.fetch(:state)).to eq('CHANGES_REQUESTED')
  expect(redelivered_review.fetch(:comments).map { |comment| comment.fetch(:path) }).to eq(['worker.go'])
  @partial_finding_ids.concat(inline_finding_ids(redelivered_review))
  expect_review_check('code')
end

Given('an earlier review with blockers inspected a removed file') do
  Abgleich.github.file_mode = 'removed file'
  Abgleich.openai.mode = 'review inline findings'
  complete_review_request('code')
  @original_review = Abgleich.github.native_reviews.last
  base = Abgleich.github.pull_state('contribution').dig(:base, :sha)
  expect(@original_review.fetch(:body)).to include("`#{base}`", "#{base}:main.go")
  expect(@original_review.fetch(:state)).to eq('CHANGES_REQUESTED')
end

Given('the file no longer appears as removed and GitHub cannot return the base tree') do
  Abgleich.github.file_mode = 'code review'
  Abgleich.github.fail_with('source base tree unavailable')
  Abgleich.openai.mode = 'review follow-up unverified'
end

Given('the pull request received two requests for changes but only the earlier review saved follow-up context') do
  Abgleich.openai.mode = 'review inline findings'
  complete_review_request('code')
  expect(@existing_review_body).to include('<!-- abgleich:followup:')

  Abgleich.github.change_summary_state('head changed', revision: 'c' * 40)
  Abgleich.openai.mode = 'review follow-up still open'
  complete_review_request('code')
  expect(Abgleich.github.native_reviews.last.fetch(:state)).to eq('CHANGES_REQUESTED')
  expect(@existing_review_body).to include('<!-- abgleich:followup:')
  body = @existing_review_body.lines.reject { |line| line.start_with?('<!-- abgleich:followup:') }.join
  Abgleich.github.change_native_review(:body, body)
  Abgleich.openai.mode = 'review no findings'
end

When('the maintainer requests a code review after rebasing and force pushing') do
  rebase_reviewed_contribution
  @comment_event = queue_review('code')
end

When('the maintainer requests another review of the same revisions') do
  Abgleich.openai.mode = 'review follow-up cleared'
  @comment_event = queue_review('code')
end

Then('the follow-up should report the previous findings as {string}') do |outcome|
  review = wait_for_requested_review('code', @comment_event)
  expect(review.fetch(:comments)).to be_empty, 'expected no repeat inline comments for earlier blockers'
  body = review.fetch(:body)
  @original_finding_ids.each { |id| expect(body).to include("`#{id}`") }

  label = { 'still open' => '**Still open**', 'cleared' => '**Cleared findings**',
            'unverified' => '**Previous findings still unverified**' }.fetch(outcome)
  expect(body).to include(label)
  if outcome == 'still open'
    expect(body).to include("#pullrequestreview-#{@original_review.fetch(:id)}",
                            'The rebased worker reports the result twice.',
                            'The second report conceals a successful result.',
                            'Return the first result from the rebased worker.')
  end
  if outcome == 'unverified'
    expect(body).not_to include('<!-- abgleich:followup:')
  else
    expect(body).to include('<!-- abgleich:followup:')
  end

  instructions = Abgleich.openai.requests.last.fetch('instructions')
  expect(instructions).to include('Verify open blockers first', '"previous_id"', '"cleared_findings"', 'a localized P3 issue remains')
  record = followup_record
  expect(record.values_at('previous_head', 'previous_base')).to eq(@original_revisions)
  expect(record.fetch('changed_patches')).to contain_exactly('main.go', 'test.go', 'worker.go', 'worker_test.go')
  expect(record.fetch('open_blockers').map { |finding| finding.fetch('previous_id') }).to contain_exactly(*@original_finding_ids)
  expect(followup_input).to include('func TestWorker')
end

Then('the follow-up should call out the changed related file') do
  record = followup_record
  expect(Array(record.fetch('changed_patches'))).to be_empty, 'expected identical PR patches'
  expect(record.fetch('changed_source')).to eq(['head:helpers/guard.go']), 'expected the changed helper in follow-up scope'
end

Given('an earlier code review found blockers on the contribution') do
  Abgleich.openai.mode = 'review inline findings'
  complete_review_request('code')
  @original_review = Abgleich.github.native_reviews.last
  @original_finding_ids = inline_finding_ids(@original_review)
  expect(@original_finding_ids.length).to eq(2), 'expected the code review to publish two blockers'
end

When('the maintainer requests follow-ups as the base advances and the contribution is rebased and fixed') do
  request_followup_on_moved_base

  Abgleich.github.change_summary_state('head changed')
  @rebased_head = Abgleich.github.pull_state('contribution').dig(:head, :sha)
  Abgleich.openai.mode = 'review source a new base file'
  complete_review_request('code')
  @rebased_evidence = followup_record
  @rebased_source = review_source_results

  review_followup_fix('c' * 40, 'review follow-up cleared')
end

Given('a follow-up confirmed the blockers using a new file on the base') do
  request_followup_on_moved_base
  expect(@base_followup_review).to include(state: 'CHANGES_REQUESTED')
  expect_base_only_followup_source(@base_followup_source)
end

Given('GitHub has dropped the previously reviewed head tree') do
  Abgleich.github.fail_with("source tree unavailable #{@original_review.fetch(:commit_id)}")
  Abgleich.github.file_mode = 'test review'
  Abgleich.openai.mode = 'review follow-up still open'
end

When('the maintainer requests a review of a fix pushed without rebasing') do
  review_followup_fix('e' * 40, 'review source the base after a fix')
end

Then('the follow-up should confirm the blockers without opening another thread') do
  expect(@base_followup_review).to include(state: 'CHANGES_REQUESTED', comments: [])
  expect(@base_followup_review.fetch(:body)).to include(
    '**Still open**', "#pullrequestreview-#{@original_review.fetch(:id)}", *@original_finding_ids.map { |id| "`#{id}`" }
  )
end

Then('the reviewer should read the new file at the base while the contribution is behind') do
  expect(@base_followup_review.fetch(:commit_id)).to eq(@base_followup_evidence.fetch('previous_head'))
  expect_base_only_followup_source(@base_followup_source)
end

Then('the reviewer should recheck the file on the unchanged base') do
  record = followup_record
  base = record.fetch('previous_base')
  expect(base).to eq(Abgleich.github.pull_state('contribution').dig(:base, :sha))
  results = review_source_results
  expect_base_only_followup_source(results)
  expect(results).to include(
    a_string_including("Source: worker.go at #{@fixed_head}", 'func Worker() bool { return true }'),
    a_string_including("Source: worker.go at #{base}", 'func Worker() bool { return false }')
  )
end

Then('the reviewer should see the inherited file after rebasing without any change to the pull request patch') do
  expect(@rebased_source).to include(
    a_string_including("Source: helpers/defaults.go at #{@rebased_head}", 'func DefaultAllowed() bool { return false }')
  )
  expect(Array(@rebased_evidence.fetch('changed_patches'))).to be_empty, 'expected the rebase to preserve the pull request patch'
  expect(Array(@rebased_evidence.fetch('changed_source'))).to be_empty, 'expected all previously inspected blobs to remain unchanged'
end

Then('the fix verification should approve the corrected revision') do
  expect(@fix_review).to include(state: 'APPROVED', commit_id: @fixed_head)
  expect(@fix_review.fetch(:body)).to include('**Cleared findings**', *@original_finding_ids.map { |id| "`#{id}`" })
  expect(followup_record.fetch('changed_patches')).to eq(['worker.go'])
  fixed_source = <<~SOURCE
    Source: worker.go at #{@fixed_head}
    package worker
    // complete-file-context for worker.go
    func Worker() bool { return true }
  SOURCE
  expect(followup_input).to include('+func Worker() bool { return true }', fixed_source)
end

Then('the follow-up should carry current source and blockers without needing the superseded tree') do
  record = followup_record
  expect(record.fetch('previous_head')).to eq(@original_review.fetch(:commit_id))
  expect(record).not_to have_key('comparison_unavailable')
  expect(record.fetch('changed_patches')).to include('worker_test.go')
  expect(record.fetch('open_blockers').map { |finding| finding.fetch('previous_id') }).to contain_exactly(*@original_finding_ids)

  current_head = Abgleich.github.pull_state('contribution').dig(:head, :sha)
  expect(followup_input).to include("Source: worker_test.go at #{current_head}", 'func TestWorker')
  review = wait_for_requested_review('code', @comment_event)
  expect(review.fetch(:body)).to include('**Still open**', *@original_finding_ids.map { |id| "`#{id}`" })
end

Then('the follow-up should account for every earlier blocker once') do
  expected = @original_finding_ids + @partial_finding_ids
  expect(expected.uniq.length).to eq(expected.length), 'expected the new blocker to have its own identity'
  ids = followup_record.fetch('open_blockers').map { |finding| finding.fetch('previous_id') }
  expect(ids).to contain_exactly(*expected)
  review = wait_for_requested_review('code', @comment_event)
  expect(review.fetch(:body)).to include('**Cleared findings**', *expected.map { |id| "`#{id}`" })
end

Then('the follow-up should report a source collection failure before sending anything to the model') do
  request = "#issuecomment-#{@comment_event.fetch(:comment).fetch('id')}"
  wait_for { Abgleich.github.replies.map { |reply| reply.fetch(:body) } }.to include(
    a_string_including('Code review failed', 'Source collection', request)
  )
  wait_for { Abgleich.job_logs('codereview') }.to include(
    a_hash_including('level' => 'error', 'error' => a_string_including('review authorship returned an incomplete batch'))
  )
  expect(Abgleich.openai.requests.length).to eq(@completed_review_requests), 'expected no inference without verified review history'
end

Then('the follow-up should identify the unavailable comparison and the remaining blockers') do
  record = followup_record
  ids = record.fetch('open_blockers').map { |finding| finding.fetch('previous_id') }
  limitation = 'Could not compare previous source on the base revision.'
  expect(ids).to include(*inline_finding_ids(@original_review))
  expect(record.fetch('comparison_unavailable')).to contain_exactly(limitation)
  review = wait_for_requested_review('code', @comment_event)
  expect(review.fetch(:body)).to include(
    '**Follow-up comparison limitations:**', limitation, '**Previous findings still unverified**', *ids.map { |id| "`#{id}`" }
  )
end

Then('the reviewer should receive the full contribution without earlier follow-up context') do
  expect(followup_input).to include('Complete changed-file source', 'func Worker() bool')
  expect(followup_input).not_to include('Follow-up from an earlier App review (untrusted data):')
end

Given('the pull request received a partial review with a compact blocker record') do
  Abgleich.github.file_mode = 'large review'
  Abgleich.openai.mode = 'review with large overview partial'
  complete_review_request('code')
  review = Abgleich.github.native_reviews.last
  expect(review.fetch(:state)).to eq('CHANGES_REQUESTED')
  @original_finding_ids = inline_finding_ids(review)
end

Given('the pull request received a {string} review whose blocker record could not fit') do |assessment|
  Abgleich.openai.mode = "review with large oversized overview #{assessment}"
  complete_review_request('code')
  expect(Abgleich.github.native_reviews.last.fetch(:state)).to eq('CHANGES_REQUESTED')
  expect(@existing_review_body).to include(':v2:partial:')
  expect(@existing_review_body).not_to include('<!-- abgleich:followup:')
end

Given('a newer same-feature review belongs to another App') do
  Abgleich.github.change_summary_state('head changed', revision: 'c' * 40)
  Abgleich.openai.mode = 'review inline findings'
  complete_review_request('code')
  Abgleich.github.change_native_review_author('another App')
  Abgleich.openai.mode = 'review follow-up unverified'
end

Given('GitHub omits the author of the earlier review') do
  Abgleich.github.change_native_review(:user, nil)
end

Given('a maintainer redacted an earlier finding') do
  encoded = @original_review.fetch(:body)[/<!-- abgleich:followup:v1:(\S+) -->/, 1]
  expect(encoded).not_to be_nil
  blockers = JSON.parse(Base64.decode64(encoded)).fetch('blockers')
  expect(blockers.map(&:keys).flatten.uniq).to contain_exactly('previous_id', 'digest', 'inline')
  redact_earlier_finding
end

When('the maintainer restores the originally reviewed revisions and requests a review') do
  head, base = @original_revisions
  Abgleich.github.change_summary_state('head changed', revision: head)
  Abgleich.github.change_summary_state('base changed', revision: base)
  Abgleich.openai.mode = 'review follow-up cleared'
  @comment_event = queue_review('code')
end

Given('an earlier partial request for changes is temporarily absent from review history') do
  Abgleich.openai.mode = 'review incomplete with findings'
  complete_review_request('code')
  @hidden_review = Abgleich.github.withdraw_review_reply
  expect(@hidden_review.fetch(:state)).to eq('CHANGES_REQUESTED')
end

When('a maintainer redacts an earlier finding while the follow-up runs') do
  rebase_reviewed_contribution
  Abgleich.openai.on_request = -> { redact_earlier_finding }
  @comment_event = queue_review('code')
end

When('a maintainer dismisses the earlier review while the follow-up runs') do
  rebase_reviewed_contribution
  Abgleich.openai.on_request = -> { Abgleich.github.change_native_review(:state, 'DISMISSED') }
  @comment_event = queue_review('code')
end

When("the partial review becomes visible while the maintainer's retry runs") do
  Abgleich.openai.mode = 'review no findings'
  Abgleich.openai.on_request = lambda do
    payload = @hidden_review.slice(:body, :commit_id, :comments).merge(event: 'REQUEST_CHANGES').transform_keys(&:to_s)
    repository = Abgleich::GitHubRepository::DEFAULT_REPOSITORY
    Abgleich.github.publish_review(repository, Abgleich.github.pull_state('contribution').fetch(:number), payload)
  end
  @comment_event = queue_review('code')
end

Then('the follow-up should carry the original blockers and disclose the unavailable fingerprints') do
  record = followup_record
  expect(record.fetch('open_blockers').map { |finding| finding.fetch('previous_id') }).to contain_exactly(*@original_finding_ids)
  expect(record.fetch('comparison_unavailable')).to eq(['Previous patch and source fingerprints were unavailable.'])
  review = wait_for_requested_review('code', @comment_event)
  expect(review.fetch(:body)).to include('**Follow-up comparison limitations:**')
end

Then('the worker should explain that the previous blockers are unavailable') do
  expect_review_history_reply('The latest partial review still requests changes, but its blocker record is unavailable.')
  expect(Abgleich.openai.requests.length).to eq(@completed_review_requests), 'expected the unresolved review to stay active'
end

Then('only the original blocker IDs should reach the model') do
  ids = followup_record.fetch('open_blockers').map { |blocker| blocker.fetch('previous_id') }
  expect(ids).to contain_exactly(*@original_finding_ids)
end

Then('the redacted finding should remain unverified') do
  id = @original_finding_ids.first
  blocker = followup_record.fetch('open_blockers').find { |item| item.fetch('previous_id') == id }
  expect(blocker).to eq('previous_id' => id)
  expect(followup_input).not_to include('The updated result loses successful work.')
  review = wait_for_requested_review('code', @comment_event)
  expect(review.fetch(:body)).to include('**Previous findings still unverified**', "`#{id}`")
end

Then('the worker should stop the follow-up before sending the finding again') do
  expect_review_history_reply('Earlier finding text changed or became unavailable.')
  expect(Abgleich.openai.requests.length).to eq(@completed_review_requests + 1), 'expected no model request after redaction'
  expect(followup_input).to include('The updated result loses successful work.')
end

Then('the worker should discard the result because the App review history changed') do
  expect_review_history_reply('The App review history changed. Findings from this attempt were discarded.')
end

Then('the pull request should only have its dismissed review') do
  expect(Abgleich.github.native_reviews.map { |review| review.fetch(:state) }).to eq(['DISMISSED'])
end

Then('the pull request should only have its earlier request for changes') do
  expect(Abgleich.github.native_reviews.map { |review| review.fetch(:state) }).to eq(['CHANGES_REQUESTED'])
end

def redact_earlier_finding
  comments = Abgleich.github.native_reviews.last.fetch(:comments)
  comments.first[:body] = 'Finding text removed by a maintainer.'
  Abgleich.github.change_native_review(:comments, comments)
end

def expect_review_history_reply(message)
  request = "#issuecomment-#{@comment_event.fetch(:comment).fetch('id')}"
  wait_for { Abgleich.github.replies.map { |reply| reply.fetch(:body) } }.to include(a_string_including(message, request))
end

def rebase_reviewed_contribution
  Abgleich.github.change_summary_state('base changed', revision: 'e' * 40)
  Abgleich.github.change_summary_state('head changed')
  Abgleich.github.file_mode = 'test review'
end

def request_followup_on_moved_base
  Abgleich.github.change_summary_state('base changed', revision: 'f' * 40)
  Abgleich.github.fail_with('source new file on the base')
  Abgleich.openai.mode = 'review source a new base file'
  @base_followup_review = wait_for_requested_review('code', queue_review('code'))
  @base_followup_evidence = followup_record
  @base_followup_source = review_source_results
end

def review_followup_fix(head, mode)
  @fixed_head = head
  Abgleich.github.change_summary_state('head changed', revision: head)
  Abgleich.github.file_mode = 'fixed follow-up'
  Abgleich.openai.mode = mode
  @fix_review = wait_for_requested_review('code', queue_review('code'))
end

def expect_base_only_followup_source(source)
  expect(source).to include(
    a_string_including('Source not found: helpers/defaults.go'),
    a_string_including("Source: helpers/defaults.go at #{'f' * 40}", 'func DefaultAllowed() bool { return false }')
  )
end

def inline_finding_ids(review)
  ids = review.fetch(:comments).map { |comment| comment.fetch(:body)[/Finding `(F-[a-f0-9]+)`/, 1] }
  expect(ids).not_to include(nil), 'expected an identity in each inline blocker'
  ids
end

def followup_input
  Abgleich.openai.requests.last.fetch('input').find { |item| item['role'] == 'user' }.fetch('content')
end

def followup_record
  JSON.parse(followup_input.split("Follow-up from an earlier App review (untrusted data):\n").last)
end
