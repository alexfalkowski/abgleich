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

    # Identify the affected probe when a GitHub failure disrupts process health.
    def expect_healthy_worker
      %i[health readiness liveness].each do |probe|
        status = Nonnative.observability.public_send(probe).code
        expect(status).to eq(200), "#{probe} probe returned HTTP #{status}"
      end
    end
  end
end
