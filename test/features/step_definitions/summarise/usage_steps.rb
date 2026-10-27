# frozen_string_literal: true

Given('the AI service reports {string} token usage') do |information|
  modes = { 'ordinary input' => nil, 'cached input' => 'cached usage', 'cache writes' => 'cache writes',
            'no usage' => 'missing usage', 'an invalid cached count' => 'invalid usage',
            'overlapping cache writes' => 'overlapping writes', 'negative usage' => 'negative usage',
            'no cache breakdown' => 'missing cache breakdown' }
  Abgleich.openai.mode = modes.fetch(information)
end

Given('the AI service identifies a different model in its response') do
  Abgleich.openai.mode = 'different response model'
end

Then('the summary details should include {string} without a dollar estimate') do |expected|
  wait_for do
    Abgleich.github.replies.any? { |reply| reply[:body].start_with?("### 🤖 Abgleich summary\n") }
  end.to be(true), 'expected a summary with model and token usage details'
  summary = Abgleich.github.replies.find { |reply| reply[:body].start_with?("### 🤖 Abgleich summary\n") }
  _, details = summary.fetch(:body).split('<details>', 2)
  expect(details).to include(expected), "expected the summary details to include #{expected.inspect}"
  expect(details).not_to include('Estimated token cost', '$'), 'expected provider usage without a dollar estimate'
end
