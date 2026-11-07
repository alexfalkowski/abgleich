# frozen_string_literal: true

Given('the reviewed repository is private') do
  Abgleich.github.add_repository(Abgleich::GitHubRepository::DEFAULT_REPOSITORY, private: true)
end

Given('the reviewer needs public GitHub source') do
  Abgleich.openai.mode = 'review source public file'
end

Given('the reviewer needs a public GitHub directory') do
  Abgleich.openai.mode = 'review source public directory'
end

Given('the reviewer needs public GitHub source with a corrected path') do
  Abgleich.openai.mode = 'review source public corrected path'
end

Given('the reviewer requests public GitHub source with an invalid {string}') do |field|
  Abgleich.openai.mode = "review source public invalid #{field}"
end

Given('the public GitHub source has {string}') do |problem|
  Abgleich.github.fail_with("public #{problem}")
end

Given('the upstream tag moves after its directory is listed') do
  Abgleich.openai.on_request = lambda do
    Abgleich.github.fail_with('public moved tag') if Abgleich.openai.requests.length > 1
  end
end

Then(/^the (code|security) review should disclose the inspected upstream revision$/) do |kind|
  body = review_text(kind)
  expect(review_source_results).to include(a_string_including('Public GitHub source (untrusted data)', 'external-chart-context'))
  expect(body).to include("upstream/chart@#{Abgleich::GitHubService::PUBLIC_REVISION}:chart/values.yaml", ':v2:')
  expect(body).not_to include('Limitations')
  expect_no_automation_mutations
end

Then('upstream GitHub requests should omit credentials') do
  requests = Abgleich.github.public_requests
  expect(requests).not_to be_empty
  expect(requests).to all(include(authorization: nil))
end

Then('the upstream file should use the first resolved revision') do
  expect(review_source_results.first).to include('Public GitHub directory', 'chart/values.yaml')
  commits = Abgleich.github.public_requests.map { |request| request.fetch(:path) }.grep(%r{/commits/})
  expect(commits).to eq(['/repos/upstream/chart/commits/v1.0.0', "/repos/upstream/chart/commits/#{Abgleich::GitHubService::PUBLIC_REVISION}"])
end

Then('the upstream source should be unavailable to the code reviewer') do
  body = review_text('code')
  expect(body).to include('Limitations', 'Review completed with limitations')
  expect(body).not_to include('external-chart-context', ':v2:')
  expect(wait_for_review('code').fetch(:state)).to eq('COMMENTED')
  expect_no_automation_mutations
end

Then('no upstream GitHub requests should be sent') do
  expect(Abgleich.github.public_requests).to be_empty
end

Then('the reviewer should receive an upstream not-found result') do
  expect(review_source_results.first).to include('Source not found:', 'chart/missing.yaml')
end
