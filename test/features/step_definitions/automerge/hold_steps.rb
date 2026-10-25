# frozen_string_literal: true

Given('the hold label is added before automation') do
  Abgleich.github.fail_with('hold before mutation')
end

Given('the hold label is added before the command labels the pull request') do
  Abgleich.github.fail_with('hold before labeling')
end

Given('the worker has repeatedly respected the hold on branch {string}') do |branch|
  Nonnative.start
  wait_for { Abgleich.job_logs('automerge').length }.to be >= 2, "expected two completed checks while #{branch} is held"
  expect_successful_check('automerge')
  expect(Abgleich::GitHubService.request_counts.fetch(:mutation_requests)).to eq(0), "expected no mutations while #{branch} is held"
end

Given('the automerge label was removed from branch {string}') do |branch|
  Abgleich.github.revoke_comment_request(branch)
end

When('the hold label is removed from branch {string}') do |branch|
  Abgleich.github.remove_label(branch, 'hold')
end

When('the worker restarts with the hold still present') do
  Abgleich.restart
end

Then('branch {string} should receive no worker mutations') do |branch|
  expected_checks = Abgleich.job_logs('automerge').length + 2
  wait_for { Abgleich.job_logs('automerge').length }.to be >= expected_checks, "expected two more completed checks for #{branch}"
  pull = Abgleich.github.pull_state(branch)
  expect(pull.fetch(:actions)).to be_empty, "expected no merge actions on branch #{branch}"
  expect(Abgleich.github.rebased?(branch)).to be(false), "expected no rebase of branch #{branch}"
  expect_successful_check('automerge')
end

Then('branch {string} should retain its existing auto-merge') do |branch|
  merge_method = Abgleich.github.pull_state(branch).dig(:auto_merge, :merge_method)
  expect(merge_method).to eq('squash'), "expected existing auto-merge on #{branch} to remain enabled"
end

Then('branch {string} should retain the {string} label') do |branch, label|
  labels = Abgleich.github.pull_state(branch).fetch(:labels).map { |current| current.fetch(:name) }
  expect(labels).to include(label), "expected #{branch} to retain its #{label} label"
end
