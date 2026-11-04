# frozen_string_literal: true

Given('the AI provider rejects the review because of {string}') do |problem|
  Abgleich.openai.mode = "diagnostic #{problem}"
end

Then('its failure comment should explain {string} with recovery advice') do |reason|
  @failure_body = Abgleich.github.replies.last.fetch(:body)
  visible, = @failure_body.split('<details>', 2)
  expect(visible).to include('### ❌ Abgleich', reason, '**🛠️ Recovery:**', 'Post a new `/ab ')
  expect(@failure_body).not_to include('<!-- abgleich:'), 'expected failed attempts to permit a fresh request'
  command = @comment_event.fetch(:comment).fetch('body').delete_prefix('/ab ')
  wait_for { Abgleich.job_logs(command) }.to include(a_hash_including('level' => 'error', 'error' => a_kind_of(String)))
end

Then('its folded failure details should identify the request and revisions') do
  visible, details = @failure_body.split('<details>', 2)
  pull = Abgleich.github.pull_state('contribution')
  request = "#issuecomment-#{@comment_event.fetch(:comment).fetch('id')}"
  model = @comment_event.fetch(:comment).fetch('body').include?('secreview') ? 'test-security' : 'test-review'
  expect(details).to include('<summary>🔎 Failure details</summary>', '**📍 Stage:** AI generation', request,
                             pull.dig(:head, :sha), pull.dig(:base, :sha), model.gsub('-', '\\-'),
                             '**📊 Tokens:** Unavailable', '</details>')
  expect(visible).not_to include(pull.dig(:head, :sha), pull.dig(:base, :sha), '**🌐 HTTP status:**')
end

Then('the provider failure diagnostics should be:') do |table|
  expected = table.rows_hash
  expected['http_status'] = expected.fetch('http_status').to_i
  expected['provider_request_id'] = 'req_0123456789abcdef0123456789abcdef'
  expected['provider_request_id'] = '' if Abgleich.openai.mode == 'diagnostic unrecognized diagnostics'
  wait_for { Abgleich.logs }.to include(a_hash_including('msg' => 'ai generation', 'level' => 'error', **expected))
  expect(@failure_body).to include("**🌐 HTTP status:** #{expected.fetch('http_status')}")
  expected.except('http_status').each_value do |value|
    expect(@failure_body).to include(value.gsub('_', '\\_')) unless value.empty?
  end
  if Abgleich.openai.mode == 'diagnostic unrecognized diagnostics'
    expect(@failure_body.scan('Unavailable or unrecognized.').length).to eq(3), 'expected unknown provider fields to be withheld'
  end
end

Then('provider error contents should remain private') do
  exposed = Abgleich.logs.to_json + @failure_body
  expect(exposed).not_to include('nonnative-summary-key', 'func Worker() bool', 'provider-private-detail', '@everyone',
                                 'https://untrusted.invalid', '<script>', 'sk-provider-secret')
end
