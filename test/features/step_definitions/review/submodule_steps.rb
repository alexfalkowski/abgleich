# frozen_string_literal: true

Given('shared tooling is available from the submodule URL {string}') do |url|
  Abgleich.github.add_repository('alexfalkowski/bin', auto_merge: false)
  Abgleich.github.submodule = { url: }
  Abgleich.openai.mode = 'review source shared tooling'
end

Given('the shared tooling submodule is {string}') do |condition|
  options = { url: 'git@github.com:alexfalkowski/bin.git' }
  case condition
  when 'owned by someone else'
    options[:url] = 'git@github.com:someone/bin.git'
    Abgleich.github.add_repository('someone/bin')
  when 'outside GitHub' then options[:url] = 'git@example.invalid:alexfalkowski/bin.git'
  when 'using HTTPS' then options[:url] = 'https://github.com/alexfalkowski/bin.git'
  when 'using an SSH URL' then options[:url] = 'ssh://git@github.com/alexfalkowski/bin.git'
  when 'using a relative URL' then options[:url] = '../bin.git'
  when 'inaccessible to the App', 'private', 'without visibility metadata' then nil
  when 'at a different commit' then options[:sha] = 'd' * 40
  when 'at a different path' then options[:path] = 'other'
  when 'missing its tree' then options[:unavailable] = true
  else raise "Unknown submodule condition: #{condition}"
  end
  unless condition == 'inaccessible to the App'
    private_repository = { 'private' => true, 'without visibility metadata' => nil }.fetch(condition, false)
    Abgleich.github.add_repository('alexfalkowski/bin', auto_merge: false, private: private_repository)
  end
  Abgleich.github.submodule = options
  Abgleich.openai.mode = 'review source shared tooling'
end

Given('the contribution updates the shared tooling submodule') do
  Abgleich.github.file_mode = 'submodule update'
end

Given('the changed submodule patch is unavailable') do
  Abgleich.github.file_mode = 'submodule update missing patch'
end

Then(/^the (code|security) reviewer should receive the pinned shared tooling$/) do |kind|
  body = review_text(kind)
  outputs = review_source_results
  expect(outputs.first).to eq("bin\tsubmodule")
  expect(outputs[1]).to include("Submodule: bin (alexfalkowski/bin): #{'b' * 40}", "README.md\t", "build/check\t")
  expect(outputs[2]).to include("Source: bin (alexfalkowski/bin): #{'b' * 40}:README.md", 'shared-tooling-pinned-context for README.md')
  expect(outputs[3]).to include("build/check\t")
  expect(outputs.last).to include('shared-tooling-pinned-context for build/check')
  expect(Abgleich.github.source_requests).to include(path: 'README.md', revision: 'b' * 40, repository: 'alexfalkowski/bin')
  expect(body).to include('Inspected source', "bin (alexfalkowski/bin): #{'b' * 40}:README.md")
  expect_no_automation_mutations
end

Then('no shared tooling source should be submitted') do
  wait_for_review('code')
  expect(Abgleich.openai.requests.last.fetch('input').to_json).not_to include('shared-tooling-pinned-context')
  expect(Abgleich.github.source_requests).not_to include(a_hash_including(repository: 'alexfalkowski/bin'))
  expect(Abgleich.github.source_requests).not_to include(a_hash_including(repository: 'someone/bin'))
end

Then('the reviewer should receive shared tooling through normal file lookup') do
  wait_for_review('code')
  expect(review_source_results.first).to include('Submodule: bin (alexfalkowski/bin)',
                                                 'shared-tooling-pinned-context for README.md',
                                                 'shared-tooling-pinned-context for build/check', 'helper-context')
  expect_no_automation_mutations
end

Then('the code review should finish after correcting the submodule path') do
  body = review_text('code')
  expect(review_source_results).to include(a_string_including('Source not found: bin/missing'),
                                           a_string_including('shared-tooling-pinned-context'))
  expect(body).not_to include('Limitations')
end
