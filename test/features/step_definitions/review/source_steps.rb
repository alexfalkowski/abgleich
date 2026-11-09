# frozen_string_literal: true

Given('repository source and public Go dependencies are available for review') do
  Abgleich.prepare_github_webhook
  Abgleich.github.add_repository(Abgleich::GitHubRepository::DEFAULT_REPOSITORY)
  Abgleich.use_config('review_sources')
  Abgleich.openai.mode = 'review source lookup'
end

Given('dependency lookup is configured but reviews are disabled') do
  Abgleich.use_config('review_sources_only')
end

Given('related source is {string}') do |condition|
  Abgleich.github.fail_with("source #{condition}")
end

Given('related source becomes {string} during investigation') do |condition|
  Abgleich.openai.on_request = -> { Abgleich.github.fail_with("source #{condition}") }
end

Given('the AI provider rejects the review after source investigation') do
  Abgleich.openai.mode = 'review source context limit'
end

Given('the reviewed module has a workspace {string}') do |location|
  Abgleich.github.fail_with("source workspace #{location}")
  Abgleich.openai.mode = 'review source a nested Go module'
end

Given('the reviewer needs {string}') do |lookup|
  Abgleich.openai.mode = "review source #{lookup}"
end

Given('the reviewer remains uncertain after inspecting available source') do
  Abgleich.openai.mode = 'review source incomplete'
end

Given('external Go reference documentation is unavailable') do
  Abgleich.github.fail_with('source Go reference unavailable')
  Abgleich.openai.mode = 'review source the declared Go language rules'
end

Given('the AI service omits usage for one round of investigation') do
  Abgleich.openai.mode = 'review source a response without usage details'
end

Given('review authorization changes through {string} during investigation') do |change|
  Abgleich.openai.on_request = -> { Abgleich.github.change_review_state(change) }
end

Given('a patch-only review was published before source lookup was available') do
  event = Abgleich.github.add_comment('contribution', body: '/ab codereview')
  Nonnative.start
  expect(Abgleich.deliver_comment(event).code).to eq(204)
  wait_for { Abgleich.github.review_results.any? { |reply| reply[:body].start_with?(review_heading('code')) } }.to be(true)
  expect_review_check('code')
  legacy = Abgleich.github.review_results.last.fetch(:body).sub(':v2:', ':v1:')
  Abgleich.github.withdraw_review_reply
  Abgleich.github.publish_reply(Abgleich::GitHubRepository::DEFAULT_REPOSITORY, 1, legacy)
end

Then(/^the (code|security) review should explain the related implementation$/) do |kind|
  with_wait(timeout: 25) do
    wait_for { reviews_for(kind).last&.fetch(:body) }.to include(':v2:'), "expected a completed #{kind} review within its job budget"
  end
  body = review_text(kind)
  expect(body).to include('Verified related source', 'helper-context', 'dependency-query-context', 'Inspected source')
  expect(body).to include('helpers/guard.go', 'example.com/library@v1.0.0/query.go', '400 input', '160 output')
  expect_no_automation_mutations
end

Then('the reviewer should receive complete files at the reviewed revisions') do
  request = Abgleich.openai.requests.first
  expect(request.fetch('input').to_json).to include('complete-file-context'), 'expected complete changed files before review'
  expect(request.fetch('reasoning')).to include('effort' => 'high')
  pull = Abgleich.github.pull_state('contribution')
  expected = [pull.dig(:head, :sha)]
  expected << pull.dig(:base, :sha) if Abgleich.github.file_mode == 'removed file'
  actual = Abgleich.github.source_requests.map { |source| source.fetch(:revision) }.uniq
  expect(actual).to contain_exactly(*expected), 'expected source only at the reviewed head and deleted-file base'
end

Then('public dependency access should omit credentials') do
  expect(Abgleich.github.module_requests).to eq([nil]), 'expected one credential-free checksum-verified module download'
end

Then('the reviewer should receive the complete related file') do
  wait_for_review('code')
  source = review_source_results
  revision = Abgleich.github.pull_state('contribution').dig(:head, :sha)
  expected = "Source: helpers/guard.go at #{revision}\npackage worker\n// helper-context\nfunc Allowed() bool { return true }\n"
  expect(source).to include(a_string_including(expected)), 'expected the complete related file at the reviewed head revision'
  body = review_text('code')
  expect(body).to include('Inspected source', "#{revision}:helpers/guard.go")
end

Then(/^the (code|security) reviewer should discover dependency files and inspect only the selected file$/) do |kind|
  body = review_text(kind)
  source = review_source_results
  expected = "package alpha\n// dependency-alpha-context\nfunc Query() string { return \"alpha\" }\n"
  listings = source.first(3).map { |listing| listing.lines.map { |line| line.split("\t").first.strip } }
  expect(listings).to include(%w[go.mod pkg/alpha/query.go pkg/zeta/query.go query.go]), 'expected paths from the module root'
  expect(listings).to include(%w[pkg/alpha/query.go pkg/zeta/query.go]), 'expected complete dependency paths in name order'
  expect(listings).to include(['pkg/alpha/query.go']), 'expected file paths when listing a Go package'
  expect(source).to include(a_string_including(expected)), 'expected the complete selected dependency file'
  expect(source.join).not_to include('dependency-query-context', 'dependency-zeta-context')
  expect(body).to include('Inspected source', 'example.com/library@v1.0.0/pkg/alpha/query.go')
  expect(body).not_to include('example.com/library@v1.0.0/query.go', 'example.com/library@v1.0.0/pkg/zeta/query.go')
  expect_no_automation_mutations
end

Then(/^the (code|security) review should finish after correcting the dependency path$/) do |kind|
  body = review_text(kind)
  source = review_source_results
  expect(source).to include('Source not found: example.com/library@v1.0.0/pkg/missing')
  expect(source).to include(a_string_including('dependency-alpha-context'))
  expect(body).to include('Inspected source', 'example.com/library@v1.0.0/pkg/alpha/query.go')
  expect(body).not_to include('Limitations', 'example.com/library@v1.0.0/pkg/missing')
  expect_no_automation_mutations
end

Then(/^the (code|security) review should finish after correcting the repository path "([^"]*)"$/) do |kind, path|
  body = review_text(kind)
  source = review_source_results
  expect(source).to include(a_string_including("Source not found: #{path}"))
  expect(source).to include(a_string_including('helper-context'))
  expect(body).to include('Inspected source', 'helpers/guard.go', ':v2:')
  expect(body).not_to include('Limitations', path)
  expect(Abgleich.github.source_requests).not_to include(a_hash_including(path:))
  expect_no_automation_mutations
end

Then('the code review should receive the declared Go language reference') do
  review = wait_for_review('code')
  expect(Abgleich.github.reference_requests).to eq([{ version: '1.22', authorization: nil }])
  expect(Abgleich.openai.requests.last.fetch('input').to_json).to include('go1.22-reference-marker')
  expect(review.fetch(:body)).to include('Go 1.22 release notes')
end

Then('both language lookups should receive the same complete reference') do
  references = review_source_results
  expect(references.length).to eq(2), 'expected the reviewer to receive both requested references'
  expect(references).to all(include('Declared Go language: 1.22.', 'untrusted reference data', 'go1.22-reference-marker'))
  expect(references.last).to eq(references.first), 'expected a repeated lookup to preserve the complete reference'
end

Then(/^the (code|security) review should disclose "([^"]*)"$/) do |kind, limitation|
  body = review_text(kind)
  expect(body).to include('Limitations', limitation)
  expect_no_automation_mutations
end

Then(/^the (code|security) review should disclose external source note "([^"]*)"$/) do |kind, note|
  expect(review_text(kind)).to include('**External source notes:**', note, 'do not by themselves prevent approval')
  expect_no_automation_mutations
end

Then(/^the (code|security) review should disclose the excluded binary file "([^"]*)"$/) do |kind, path|
  body = review_text(kind)
  excluded = body.split('**Excluded binary files:**').last.split('📝 **Overview**').first
  expect(excluded).to include(path, 'do not prevent approval')
  expect(body.split('**Excluded binary files:**').first).not_to include(path)
  inspected = body.split('**Inspected source:**', 2)[1].to_s
  expect(inspected).not_to include(path)
  submitted = Abgleich.openai.requests.last.fetch('input').to_json
  expect(submitted).to include('Source excluded:')
  expect(submitted).not_to include('binary-asset-context')
  expect_no_automation_mutations
end

Then(/^the (code|security) review should have no context limitations$/) do |kind|
  body = review_text(kind)
  expect(body).not_to include('**Limitations:**', 'Review completed with limitations')
  expect(body).to include(':v2:')
end

Then('the deleted binary file should be identified at the reviewed base revision') do
  base = Abgleich.github.pull_state('contribution').dig(:base, :sha)
  expect(Abgleich.github.native_reviews.last.fetch(:body)).to include("#{base}:image.png")
  expect(Abgleich.github.source_requests).to include(path: 'image.png', revision: base)
end

Then(/^no unverified (repository|helper|dependency) source should be shared$/) do |scope|
  submitted = Abgleich.openai.requests.last.fetch('input').to_json
  expect(submitted).not_to include(*review_source_markers(scope))
end

Then('the public dependency proxy should receive {int} requests without credentials') do |count|
  expect(Abgleich.github.module_requests).to eq(Array.new(count)), "expected #{count} credential-free proxy requests"
end

Then('the symbolic link should not be followed') do
  expect(Abgleich.github.source_requests).not_to include(a_hash_including(path: 'helpers/guard.go'))
end

Then('the excluded repository path {string} should not be fetched') do |path|
  expect(Abgleich.github.source_requests).not_to include(a_hash_including(path:))
end

Then('deleted source should be read from the reviewed base revision') do
  base = Abgleich.github.pull_state('contribution').dig(:base, :sha)
  expect(Abgleich.github.source_requests).to include(path: 'main.go', revision: base)
end

Then('the source-enabled review should coexist with the old patch-only review') do
  bodies = Abgleich.github.review_results.map { |reply| reply[:body] }
  expect(bodies.length).to eq(2), 'expected the new review scope to permit one source-enabled result'
  expect(bodies.first).to include(':v1:')
  expect(bodies.last).to include(':v2:')
end

Then('the rejected path or dependency should not be fetched') do
  expect(Abgleich.github.module_requests).to be_empty
  expect(Abgleich.github.source_requests.none? { |request| request[:path].include?('..') }).to be(true)
end

Then(/^the (code|security) review should stop before sharing related source$/) do |kind|
  # A scheduled idle check can finish before the queued review starts.
  wait_for { Abgleich.openai.requests.length }.to be >= 1, "expected the #{kind} review investigation to begin"
  expect_review_check(kind)
  expect(Abgleich.openai.requests.length).to eq(1), 'expected no continuation after consent or revisions change'
  expect(Abgleich.github.source_requests.none? { |request| request[:path] == 'helpers/guard.go' }).to be(true)
  expect(Abgleich.github.module_requests).to be_empty
  expect(Abgleich.github.review_results.none? { |reply| reply[:body].start_with?(review_heading(kind)) }).to be(true)
end

Then(/^the (code|security) reviewer should receive the related files together$/) do |kind|
  wait_for_review(kind)
  sources = review_source_results
  expect(sources.length).to eq(1), 'expected related paths to share one source lookup'
  expect(sources.first).to include('helper-context', 'module example.com/worker')
  expect_no_automation_mutations
end

Then('the review should use at most {int} pull request checks') do |budget|
  counts = Abgleich::GitHubService.request_counts
  expect(counts.fetch(:pull_request_reads)).to be <= budget, 'expected related source to fit within the GitHub request budget'
  expect(counts.fetch(:comment_reads)).to eq(0), 'expected fresh discussion reads to also authorize the request'
end

Then('the review should use at most {int} GitHub source reads') do |budget|
  count = Abgleich::GitHubService.request_counts.fetch(:source_reads)
  expect(count).to be <= budget, 'expected grouped source access to fit within the GitHub request budget'
end

Then('the reviewer should receive an empty regular file') do
  body = review_text('code')
  revision = Abgleich.github.pull_state('contribution').dig(:head, :sha)
  expect(review_source_results).to include("Source: helpers/guard.go at #{revision}\n\n")
  expect(body).to include('Inspected source', "#{revision}:helpers/guard.go")
  expect(body).not_to include('Limitations')
end

Then('the reviewer should receive all {int} files from the directory') do |count|
  wait_for_review('code')
  source = review_source_results.join("\n")
  (1..count).each do |index|
    expect(source).to include("directory-context for helpers/part#{index}.go"), "expected directory file #{index} without a local file cap"
  end
  expect(source).to include('directory-tail-context'), 'expected the final directory file to remain complete'
  expect_no_automation_mutations
end

Then(/^the (code|security) review should use all requested source$/) do |kind|
  review = wait_for_review(kind)
  sources = review_source_results
  expect(sources.length).to eq(3), 'expected every requested source lookup to reach the model'
  expect(sources).to all(include('helpers/guard.go'))
  body = review.fetch(:body)
  expect(body).to include('Inspected related source paths')
  expect(body).not_to include('Limitations')
  expect_no_automation_mutations
end

Then(/^the (code|security) reviewer should receive only the requested source lines$/) do |kind|
  body = review_text(kind)
  source = review_source_results
  expect(source.first).to match(/^query\.go\t\d+ bytes$/)
  expect(source.last).to include('Excerpt lines 2-2:', '2: // dependency-query-context')
  expect(source.join).not_to include('package worker', 'package library', 'func Query', 'unrelated-source-padding')
  expect(body).to include('query.go (lines 2-2)')
  Abgleich.openai.requests.each do |request|
    expect(request.to_json.bytesize).to be < 20_000, 'expected compact dependency excerpts'
  end
  expect(Abgleich::GitHubService.request_counts.fetch(:source_reads)).to eq(3), 'expected manifest reuse across dependency reads'
  expect(Abgleich.github.module_requests).to eq([nil]), 'expected one credential-free verified dependency download'
end

Then('the excerpt should not claim inspection of the related file') do
  expect(review_text('code')).not_to include('example.com/library@v1.0.0/query.go')
end

Then('the dependency excerpt should contain no source lines') do
  expect(review_source_results.last).to include('No lines in the requested range.')
end

Then(/^the (code|security) generation log should identify "([^"]*)" as its largest source response$/) do |kind, tool|
  expected = { 'msg' => 'ai generation', 'task' => review_command(kind),
               'largest_tool' => tool, 'largest_tool_output_bytes' => review_source_results.map(&:bytesize).max }
  wait_for { Abgleich.logs.select { |record| record['msg'] == 'ai generation' && record['task'] == review_command(kind) } }
    .to contain_exactly(a_hash_including(expected))
  expect(Abgleich.logs.to_json).not_to include('helper-context', 'dependency-query-context', 'unrelated-source-padding')
end
