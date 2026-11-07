# frozen_string_literal: true

Given('GitHub cannot retrieve the complete security review discussion') do
  Abgleich.github.fail_with('summary comments page:2')
end

Given('two security review requests are waiting for the same revision') do
  Abgleich.use_config('secreview_waiting')
  Nonnative.start
  wait_for { Abgleich.job_logs('secreview') }.not_to be_empty
  2.times do
    event = Abgleich.github.add_comment('contribution', body: '/ab secreview')
    @comment_event ||= event
    expect(Abgleich.deliver_comment(event).code).to eq(204)
  end
end

Given('a security review previously failed at the provider') do
  Abgleich.openai.mode = 'refusal'
  @comment_event = Abgleich.github.add_comment('contribution', body: '/ab secreview')
  Nonnative.start
  expect(Abgleich.deliver_comment(@comment_event).code).to eq(204)
  expect_review_failure('security')
  Abgleich.openai.mode = ''
end

Given('the target branch has advanced since the security review') do
  Abgleich.github.change_summary_state('base changed')
end

Given('the contribution uses parameterized SQL safely') do
  Abgleich.github.file_mode = 'safe security change'
end

Then('ordinary automation should complete a check while a security review is being prepared') do
  wait_for { Abgleich.job_logs('automerge').length }.to be > @security_automation_checks
  expect_successful_check('automerge')
  expect(Abgleich.logs).not_to include(a_hash_including('msg' => 'ai generation', 'task' => 'secreview'))
end

Then('the security reviewer should have read-only access') do
  payload = Abgleich.openai.requests.find { |request| request.fetch('model') == 'test-security' }
  expect(payload).not_to be_nil, 'expected the security model to receive the authorized patches'
  expect(payload.fetch('instructions')).to include('attacker-controlled input', 'prerequisites', 'mitigation', 'untrusted')
  expect(payload.fetch('store')).to be(false), 'expected security inference response storage to remain disabled'
  names = payload.fetch('tools').map { |tool| tool.fetch('name') }
  expect(names).to contain_exactly('list_source_files', 'read_sources', 'read_go_dependency', 'read_go_reference')
  expect(payload).not_to have_key('max_output_tokens')
end
