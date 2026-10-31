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

Given('the reviewed module has a workspace {string}') do |location|
  Abgleich.github.fail_with("source workspace #{location}")
  Abgleich.openai.mode = 'review source a nested Go module'
end

Given('the reviewer needs {string}') do |lookup|
  Abgleich.openai.mode = "review source #{lookup}"
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
  wait_for { Abgleich.github.replies.any? { |reply| reply[:body].start_with?(review_heading('code')) } }.to be(true)
  expect_review_check('code')
  legacy = Abgleich.github.replies.last.fetch(:body).sub(':v2:', ':v1:')
  Abgleich.github.withdraw_review_reply
  Abgleich.github.publish_reply(Abgleich::GitHubRepository::DEFAULT_REPOSITORY, 1, legacy)
end

Then(/^the (code|security) review should explain the related implementation$/) do |kind|
  with_wait(timeout: 25) do
    wait_for do
      Abgleich.github.replies.any? { |reply| reply[:body].start_with?(review_heading(kind)) && reply[:body].include?(':v2:') }
    end.to be(true), "expected a completed #{kind} review within the source fixture's job budget"
  end
  body = Abgleich.github.replies.last.fetch(:body).delete('\\').delete("\u200b")
  expect(body).to include('Verified related source', 'helper-context', 'dependency-query-context', 'Inspected source')
  expect(body).to include('helpers/guard.go', 'example.com/library@v1.0.0/query.go', '400 input', '160 output')
  first_input = Abgleich.openai.requests.first.fetch('input').to_json
  expect(first_input).to include('complete-file-context'), 'expected complete changed files before review'
  expect(Abgleich.openai.requests.first.fetch('reasoning')).to include('effort' => 'high')
  revision = Abgleich.github.pull_state('contribution').dig(:head, :sha)
  revisions = [revision]
  revisions << Abgleich.github.pull_state('contribution').dig(:base, :sha) if Abgleich.github.file_mode == 'removed file'
  expect(Abgleich.github.source_requests.map { |request| request.fetch(:revision) }.uniq).to contain_exactly(*revisions)
  expect_no_automation_mutations
end

Then('public dependency access should omit credentials') do
  expect(Abgleich.github.module_requests).to eq([nil]), 'expected one credential-free checksum-verified module download'
end

Then('the reviewer should receive the complete related file') do
  wait_for { Abgleich.github.replies.any? { |reply| reply[:body].start_with?(review_heading('code')) } }.to be(true)
  outputs = Abgleich.openai.requests.last.fetch('input').select { |item| item['type'] == 'function_call_output' }
  source = outputs.map { |item| item.fetch('output') }
  revision = Abgleich.github.pull_state('contribution').dig(:head, :sha)
  expected = "Source: helpers/guard.go at #{revision}\npackage worker\n// helper-context\nfunc Allowed() bool { return true }\n"
  expect(source).to include(expected), 'expected the complete related file at the reviewed head revision'
  body = Abgleich.github.replies.last.fetch(:body).delete('\\').delete("\u200b")
  expect(body).to include('Inspected source', "#{revision}:helpers/guard.go")
end

Then('the reviewer should discover the dependency packages and inspect one') do
  wait_for { Abgleich.github.replies.any? { |reply| reply[:body].start_with?(review_heading('code')) } }.to be(true)
  outputs = Abgleich.openai.requests.last.fetch('input').select { |item| item['type'] == 'function_call_output' }
  source = outputs.map { |item| item.fetch('output') }
  expect(source).to include("pkg/alpha/query.go\npkg/zeta/query.go"), 'expected complete dependency paths in name order'
  expect(source).to include(a_string_including('dependency-alpha-context')), 'expected the selected package source'
  body = Abgleich.github.replies.last.fetch(:body).delete('\\').delete("\u200b")
  expect(body).to include('Inspected source', 'example.com/library@v1.0.0/pkg/alpha/query.go')
end

Then('the code review should receive the declared Go language reference') do
  wait_for { Abgleich.github.replies.any? { |reply| reply[:body].start_with?(review_heading('code')) } }.to be(true)
  expect(Abgleich.github.reference_requests).to eq([{ version: '1.22', authorization: nil }])
  expect(Abgleich.openai.requests.last.fetch('input').to_json).to include('go1.22-reference-marker')
  expect(Abgleich.github.replies.last.fetch(:body)).to include('Go 1.22 release notes')
end

Then('both language lookups should receive the same complete reference') do
  outputs = Abgleich.openai.requests.last.fetch('input').select { |item| item['type'] == 'function_call_output' }
  references = outputs.map { |item| item.fetch('output') }
  expect(references.length).to eq(2), 'expected the reviewer to receive both requested references'
  expect(references).to all(include('Declared Go language: 1.22.', 'untrusted reference data', 'go1.22-reference-marker'))
  expect(references.last).to eq(references.first), 'expected a repeated lookup to preserve the complete reference'
end

Then(/^the (code|security) review should disclose "([^"]*)"$/) do |kind, limitation|
  wait_for { Abgleich.github.replies.any? { |reply| reply[:body].start_with?(review_heading(kind)) } }.to be(true)
  expect(Abgleich.github.replies.last.fetch(:body).delete('\\').delete("\u200b")).to include('Unavailable source', limitation)
  expect_no_automation_mutations
end

Then('unavailable or unverified source should not be shared') do
  problem = Abgleich.github.failure
  submitted = Abgleich.openai.requests.last.fetch('input').to_json
  if ['source unavailable', 'source from another revision', 'source binary'].include?(problem)
    expect(submitted).not_to include('complete-file-context', 'helper-context')
    expect(Abgleich.github.module_requests).to be_empty
  elsif problem == 'source unverified against the dependency checksum'
    expect(submitted).not_to include('dependency-query-context')
  elsif problem == 'source redirected to an unrelated origin'
    expect(Abgleich.github.module_requests).to eq([nil]), 'expected the credential-free proxy request without following its redirect'
    expect(submitted).not_to include('dependency-query-context')
  elsif problem == 'source behind a symbolic link'
    expect(submitted).not_to include('helper-context')
    expect(Abgleich.github.source_requests.none? { |request| request[:path] == 'helpers/guard.go' }).to be(true)
  elsif problem.start_with?('source workspace ') ||
        ['source managed by a Go workspace', 'source vendored in the repository'].include?(problem)
    expect(Abgleich.github.module_requests).to be_empty, 'expected repository-controlled dependencies to avoid the public proxy'
    expect(submitted).not_to include('dependency-query-context')
  end
end

Then('deleted source should be read from the reviewed base revision') do
  base = Abgleich.github.pull_state('contribution').dig(:base, :sha)
  expect(Abgleich.github.source_requests).to include(path: 'main.go', revision: base)
end

Then('the source-enabled review should coexist with the old patch-only review') do
  bodies = Abgleich.github.replies.map { |reply| reply[:body] }
  expect(bodies.length).to eq(2), 'expected the new review scope to permit one source-enabled result'
  expect(bodies.first).to include(':v1:')
  expect(bodies.last).to include(':v2:')
end

Then('the rejected path or dependency should not be fetched') do
  expect(Abgleich.github.module_requests).to be_empty
  expect(Abgleich.github.source_requests.none? { |request| request[:path].include?('..') }).to be(true)
end

Then(/^the (code|security) review should stop before sharing related source$/) do |kind|
  expect_review_check(kind)
  expect(Abgleich.openai.requests.length).to eq(1), 'expected no continuation after consent or revisions change'
  expect(Abgleich.github.source_requests.none? { |request| request[:path] == 'helpers/guard.go' }).to be(true)
  expect(Abgleich.github.module_requests).to be_empty
  expect(Abgleich.github.replies.none? { |reply| reply[:body].start_with?(review_heading(kind)) }).to be(true)
end

Then('the reviewer should receive every file from the directory') do
  outputs = Abgleich.openai.requests.last.fetch('input').select { |item| item['type'] == 'function_call_output' }
  source = outputs.map { |item| item.fetch('output') }.join("\n")
  (1..3).each do |index|
    expect(source).to include("directory-context for helpers/part#{index}.go"), 'expected complete text from every directory file'
  end
  expect(source).to include('directory-tail-context'), 'expected the final directory file to remain complete'
end

Then(/^the (code|security) review should use all requested source$/) do |kind|
  counts = { 'review source several rounds of investigation' => 3, 'review source several related files' => 3 }
  count = counts.fetch(Abgleich.openai.mode)
  wait_for { Abgleich.github.replies.any? { |reply| reply[:body].start_with?(review_heading(kind)) } }.to be(true)
  outputs = Abgleich.openai.requests.last.fetch('input').select { |item| item['type'] == 'function_call_output' }
  expect(outputs.length).to eq(count), 'expected every requested source lookup to reach the model'
  expect(outputs.map { |item| item.fetch('output') }).to all(include('helpers/guard.go'))
  body = Abgleich.github.replies.last.fetch(:body)
  expect(body).to include('Inspected related source paths')
  expect(body).not_to include('Unavailable source')
  expect_no_automation_mutations
end
