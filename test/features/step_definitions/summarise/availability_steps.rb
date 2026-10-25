# frozen_string_literal: true

Given('the operator configures {string} for the AI service') do |setting|
  settings = {
    'a 50ms request limit' => { 'timeout' => '50ms' },
    'three retry attempts' => { 'retry' => { 'attempts' => 3, 'backoff' => '10ms' } },
    'a pause after one failed request' => { 'breaker' => { 'consecutive_failures' => 1, 'timeout' => '1m' } },
    'an invalid trusted certificate' => { 'tls' => { 'ca' => 'invalid-ca' } }
  }
  config = YAML.safe_load_file('.config/server.yaml')
  config['ai']['client'] = settings.fetch(setting)
  File.write('.config/server.yaml', YAML.dump(config))
end

Then('the maintainer should be asked to retry without another AI request') do
  wait_for { Abgleich.github.replies.length }.to eq(2), 'expected feedback for both summary requests'
  Abgleich.github.replies.each do |reply|
    expect(reply.fetch(:body)).to include('Summary failed')
  end
  expect(Abgleich.openai.requests.length).to eq(1), 'expected the temporary pause to prevent another AI request'
  expect_healthy_worker
end
