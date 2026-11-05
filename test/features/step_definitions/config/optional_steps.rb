# frozen_string_literal: true

Given('optional AI commands are disabled with the provider {string}') do |state|
  Abgleich.use_config({ 'configured' => 'ai', 'omitted' => 'webhook' }.fetch(state))
end

Then('only merge automation should run in the background') do
  wait_for { Abgleich.job_logs('automerge').length }.to be >= 2, 'expected repeated merge checks while optional jobs stay disabled'
  %w[cleanup summarise codereview secreview].each do |job|
    expect(Abgleich.job_logs(job)).to be_empty, "expected no #{job} job when its configuration is omitted"
  end
end
