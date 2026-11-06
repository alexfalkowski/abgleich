# frozen_string_literal: true

Given('review source caching is enabled') do
  Abgleich.use_config('review_source_cache')
end

Given('review source caching is omitted') do
  Abgleich.use_config('review_sources')
end

Given('review source exceeds the cache value limit') do
  Abgleich.use_config('review_source_cache_small')
end

Given('a different shared cache backend is configured') do
  Abgleich.use_config('review_source_cache_remote')
end

Given('the contribution updates its related helper') do
  Abgleich.github.change_summary_state('head changed')
  Abgleich.github.fail_with('source changed helper')
end

Given('another repository has a contribution with identical files') do
  repository = 'alexfalkowski/another'
  Abgleich.github.add_repository(repository)
  @review_cache_branch = 'other-contribution'
  Abgleich.github.add_pull(@review_cache_branch, repository:, auto_merge: false, current: true)
end

Given('a previous code review could not retrieve repository source') do
  Abgleich.github.fail_with('source query errors')
  queue_review('code')
  expect_review_failure('code')
  Abgleich.github.fail_with(nil)
end

Given('the security requester loses write access during investigation') do
  Abgleich.openai.on_request = -> { Abgleich.github.fail_with('final authorization:revoked write access') }
end

When('the maintainer requests code review in the other repository') do
  @comment_event = queue_review('code', branch: @review_cache_branch)
end

Then(/^the (code|security) review should account for all reused source$/) do |kind|
  review = @comment_event ? wait_for_requested_review(kind, @comment_event) : wait_for_review(kind)
  body = review.fetch(:body).delete('\\').delete("\u200b")
  revision = Abgleich.github.pull_state(@review_cache_branch || 'contribution').dig(:head, :sha)
  expected = %w[main.go worker.go test.go helpers/guard.go go.mod].map { |path| "#{revision}:#{path}" }
  expect(body).to include('Inspected source', *expected)
  expect(body).not_to include('Limitations')
  model = kind == 'code' ? 'test-review' : 'test-security'
  payload = Abgleich.openai.requests.rfind { |request| request.fetch('model') == model }
  expect(payload.fetch('input').to_json).to include('complete-file-context', 'helper-context', 'module example.com/worker')
  expect_no_automation_mutations
end

Then('GitHub should have served {int} source tree(s) and {int} source download(s)') do |trees, downloads|
  counts = Abgleich::GitHubService.request_counts
  expect(counts.fetch(:source_tree_reads)).to eq(trees), 'expected complete repository trees to be reused at the same commit'
  expect(counts.fetch(:source_reads)).to eq(downloads), 'expected only new verified blobs to be downloaded from GitHub'
end

Then('the reviewer should receive the updated helper') do
  expect(review_source_results.join).to include('updated-helper-context', 'func Allowed() bool { return false }')
end

Then('the code review should exclude the previously read symbolic link') do
  review = wait_for_requested_review('code', @comment_event)
  body = review.fetch(:body).delete('\\').delete("\u200b")
  expect(body).to include('Limitations', 'helpers/guard.go')
  expect(review.fetch(:state)).to eq('COMMENTED'), 'expected excluded source to prevent approval'
  expect(review_source_results.join).not_to include('helper-context')
  expect_no_automation_mutations
end

Then('the security review should stop before sharing reused related source') do
  wait_for { Abgleich.job_logs('secreview') }.to include(
    a_hash_including('level' => 'error', 'error' => a_string_including('needs repository write access'))
  )
  expect_review_check('security')
  requests = Abgleich.openai.requests.select { |request| request.fetch('model') == 'test-security' }
  expect(requests.length).to eq(1), 'expected withdrawn consent to prevent any source-tool continuation'
  expect(requests.first.fetch('input').to_json).not_to include('helper-context')
  expect(reviews_for('security')).to be_empty, 'expected no published security review after consent is withdrawn'
end
