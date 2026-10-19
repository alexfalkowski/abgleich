# frozen_string_literal: true

module Abgleich
  # Shared assertions evaluated in the Cucumber scenario's RSpec context.
  module Assertions
    # Successful checks can warn about overruns, but must not carry an error.
    def expect_successful_check
      wait_for do
        check = Abgleich.job_logs.last
        check && %w[info warn].include?(check['level']) && !check.key?('error')
      end.to be(true), 'expected the latest branch check to finish successfully without an error'
    end

    # Wait for the latest completed check, so an earlier failure cannot satisfy recovery setup.
    def expect_failed_check
      wait_for { Abgleich.job_logs.last }.to include('level' => 'error', 'error' => a_string_matching(/.+/))
    end

    # Skipped, rejected, and unconfirmed updates must not be reported as accepted.
    def expect_no_accepted_rebases
      expect(Abgleich.rebase_logs).to be_empty, 'expected no accepted rebase requests in the worker logs'
    end

    # Failed or unconfirmed Dependabot mutations must not be logged as accepted.
    def expect_no_accepted_dependabot_actions
      expect(Abgleich.logs).not_to include(a_hash_including('msg' => 'enabled pull request auto-merge'))
      expect(Abgleich.logs).not_to include(a_hash_including('msg' => 'merged pull request'))
    end

    # Identify the affected probe when a GitHub failure disrupts process health.
    def expect_healthy_worker
      %i[health readiness liveness].each do |probe|
        status = Nonnative.observability.public_send(probe).code
        expect(status).to eq(200), "#{probe} probe returned HTTP #{status}"
      end
    end
  end
end
