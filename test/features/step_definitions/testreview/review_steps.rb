# frozen_string_literal: true

Then('the test reviewer should have read-only access') do
  payload = Abgleich.openai.requests.find { |request| request.fetch('model') == 'test-quality' }
  expect(payload).not_to be_nil, 'expected the test model to receive the authorized source'
  instructions = payload.fetch('instructions')
  expect(instructions).to include('caller-facing behavior', 'existing coverage', 'untrusted',
                                  'determinism and isolation', 'without claiming',
                                  'The suite can pass for a known material defect')
  expect(payload.fetch('store')).to be(false), 'expected test inference response storage to remain disabled'
  names = payload.fetch('tools').map { |tool| tool.fetch('name') }
  expect(names).to contain_exactly('list_source_files', 'read_sources', 'read_submodule', 'read_go_dependency', 'read_go_reference')
  expect(payload).not_to have_key('max_output_tokens')
end

Then('the test reviewer should receive the weakened assertion') do
  input = Abgleich.openai.requests.first.fetch('input').to_json
  expect(input).to include('worker_test.go', '-func TestWorker', '+func TestWorker', 'func TestWorker(t *testing.T) { Worker() }')
end
