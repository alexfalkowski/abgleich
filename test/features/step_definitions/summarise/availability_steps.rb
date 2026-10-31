# frozen_string_literal: true

Given('the operator configures {string} for the AI service') do |setting|
  fixtures = {
    'a 50ms request limit' => 'ai_timeout',
    'three retry attempts' => 'ai_retry',
    'a pause after one failed request' => 'ai_breaker',
    'an invalid trusted certificate' => 'ai_invalid_certificate'
  }

  Abgleich.use_config(fixtures.fetch(setting))
end

Given('the operator uses the default summary schedule') do
  Abgleich.use_config('summarise_default_schedule')
end

Then('the maintainer should receive a summary within {int} seconds') do |seconds|
  with_wait(timeout: seconds) do
    wait_for do
      Abgleich.github.replies.any? { |reply| reply[:body].start_with?("### 🤖 Abgleich summary\n") }
    end.to be(true), "expected the maintainer to receive a summary within #{seconds} seconds"
  end
end

Then('the maintainer should be asked to retry without another AI request') do
  wait_for { Abgleich.github.replies.length }.to eq(2), 'expected feedback for both summary requests'
  Abgleich.github.replies.each do |reply|
    expect(reply.fetch(:body)).to include('Summary failed')
  end
  expect(Abgleich.openai.requests.length).to eq(1), 'expected the temporary pause to prevent another AI request'
  expect_healthy_worker
end
