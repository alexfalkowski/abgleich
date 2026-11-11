# frozen_string_literal: true

Given('the reviewed project uses {string}') do |language|
  Abgleich.github.file_mode = language == 'Go and Ruby' ? 'mixed' : language
  Abgleich.openai.mode = 'review no findings'
end

Given('Ruby dependency lookup encounters {string}') do |condition|
  Abgleich.github.fail_with("Ruby #{condition}")
  Abgleich.openai.mode = "review source Ruby #{condition}"
end

Given('Ruby dependency source is still downloading') do
  Abgleich.github.fail_with('Ruby slow download')
  Abgleich.openai.mode = 'review source Ruby single lookup'
  Nonnative.start
  event = Abgleich.github.add_comment('contribution', body: '/ab codereview')
  expect(Abgleich.deliver_comment(event).code).to eq(204)
  wait_for { Abgleich.github.gem_requests.any? { |item| item[:path].end_with?('.gem') } }.to be(true)
end

Then(/^the (code|security) reviewer should receive guides for "([^"]*)"$/) do |kind, languages|
  wait_for_review(kind)
  prompt = Abgleich.openai.requests.first.fetch('instructions')
  expect(prompt).to include('# Shared review rules', "# #{kind.capitalize} review")
  %w[Go Ruby Cucumber].each do |language|
    heading = "# #{language} review context"
    if languages.split(', ').include?(language)
      expect(prompt).to include(heading), "expected guidance for #{language}"
    else
      expect(prompt).not_to include(heading), "expected unrelated #{language} guidance to be omitted"
    end
  end
end

Then(/^the (code|security) reviewer should inspect the locked Ruby gem with "([^"]*)" verification$/) do |kind, verification|
  body = review_text(kind)
  source = review_source_results
  expect(source.join).to include('lib/example.rb', 'ruby-gem-source-context', verification)
  expect(source.last).to include('Excerpt lines 2-2:', '2: # ruby-gem-source-context')
  expect(source.join).not_to include('def self.call'), 'expected only the requested source excerpt'
  expect(body).to include('Inspected source', 'example@1.2.3', 'lib/example.rb (lines 2-2)')
  expect(body).not_to include('External source notes', 'Limitations')
  expect(Abgleich.github.gem_requests.length).to eq(2), 'expected one metadata read and one archive download despite repeated lookups'
  expect(Abgleich.github.gem_requests.map { |item| item[:authorization] }).to all(be_nil), 'expected credential-free gem requests'
  expect_no_automation_mutations
end

Then('unverified Ruby source should be withheld after {int} registry requests') do |count|
  wait_for_review('code')
  expect(Abgleich.openai.requests.last.fetch('input').to_json).not_to include('ruby-gem-source-context')
  expect(Abgleich.github.gem_requests.length).to eq(count), "expected #{count} requests for the rejected Ruby source"
  inspected = review_text('code').split('**Inspected source:**', 2).last.to_s
  expect(inspected).not_to include('example@1.2.3'), 'expected rejected gem source not to be recorded as inspected'
end

Then('Ruby archive rescans should share the review byte budget') do
  source = review_source_results
  expect(source.first).to include('ruby-gem-source-context'), 'expected the first verified source lookup to succeed'
  expect(source.last).to include('Ruby gem byte budget exceeded'), 'expected the cached archive rescan to use the same budget'
  expect(source.last).not_to include('ruby-gem-source-context'), 'expected no partially validated source after budget exhaustion'
  expect(Abgleich.github.gem_requests.length).to eq(2), 'expected the verified compressed archive to be reused'
end

Then('Ruby source lookup should recover through the HTTP client') do
  wait_for_review('code')
  requests = Abgleich.github.gem_requests
  expect(requests.length).to eq(3), 'expected one retry followed by one archive download'
  expect(review_source_results.join).to include('ruby-gem-source-context')
  expect(requests.map { |item| item[:authorization] }).to all(be_nil), 'expected credential-free retries'
end

Then('RubyGems should receive the locked artifact {string}') do |artifact|
  expect(Abgleich.github.gem_requests.last.fetch(:path)).to eq("/gems/#{artifact}")
end

Then('the reviewer should receive only source tools for {string}') do |languages|
  expected = %w[list_source_files read_sources read_submodule]
  { 'Go' => %w[read_go_reference read_go_dependency], 'Ruby' => ['read_ruby_dependency'] }.each do |language, tools|
    expected.concat(tools) if languages.split(', ').include?(language)
  end
  actual = Abgleich.openai.requests.first.fetch('tools').map { |tool| tool.fetch('name') }
  expect(actual).to contain_exactly(*expected), 'expected only source tools applicable to this repository'
end

Then('a rate-limited gem lookup should not block the other review') do
  bodies = %w[code security].map { |kind| review_text(kind) }
  requests = Abgleich.github.gem_requests
  expect(bodies.count { |body| body.include?('RubyGems rate limited') }).to eq(1), 'expected one disclosed rate limit'
  expect(bodies.count { |body| body.include?('example@1.2.3') }).to eq(1), 'expected the other review to inspect the gem'
  expect(requests.length).to eq(3), 'expected one rate-limited metadata read and one independent gem download'
  expect(requests.map { |item| item[:authorization] }).to all(be_nil)
end
