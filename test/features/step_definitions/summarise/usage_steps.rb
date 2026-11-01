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
  expected = { 'the model returned by the service' => 'Model:** other\\-model',
               'unavailable usage' => 'Tokens:** Unavailable' }.fetch(expected, expected)
  summary = wait_for_summary
  _, details = summary.fetch(:body).split('<details>', 2)
  expect(details).to include(expected), "expected the summary details to include #{expected.inspect}"
  expect(details).not_to include('Estimated token cost', '$'), 'expected provider usage without a dollar estimate'
end
