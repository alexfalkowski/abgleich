# frozen_string_literal: true

module Abgleich
  # Shared assertions evaluated in the Cucumber scenario's RSpec context.
  module Assertions
    # A later repository notification completes after earlier queued work,
    # exposing accidental checks without relying on an arbitrary sleep.
    def expect_event_barrier
      repository = 'alexfalkowski/event-control'
      Abgleich.github.add_repository(repository)
      event = Abgleich.github.repository_notification(repository:)
      expect(Abgleich.deliver_event('push', event).code).to eq(204), 'expected the control notification to be queued'
      wait_for { Abgleich.logs }.to include(a_hash_including('msg' => 'automerge event finished', 'repository' => repository))
    end

    # Observe a later completed summary check before asserting that no work occurred.
    def expect_summary_check
      next_check = Abgleich.job_logs('summarise').length + 1
      wait_for { Abgleich.job_logs('summarise').length }.to be >= next_check, 'expected a completed summary check after the request'
    end

    # Advisory requests must leave labels, reviews, and branches untouched.
    def expect_no_automation_mutations
      activity = { mutation_requests: Abgleich::GitHubService.request_counts.fetch(:mutation_requests),
                   label_requests: Abgleich.github.label_requests, accepted_actions: Abgleich.github.progress.fetch(:actions) }
      expect(activity).to eq(mutation_requests: 0, label_requests: 0, accepted_actions: 0), 'expected automation to remain untouched'
      expect_no_accepted_rebases
      expect_no_accepted_merge_actions
    end

    # Successful checks can warn about overruns, but must not carry an error.
    def expect_successful_check(job)
      wait_for do
        check = Abgleich.job_logs(job).last
        check && %w[info warn].include?(check['level']) && !check.key?('error')
      end.to be(true), "expected the latest #{job} check to finish successfully without an error"
    end

    # Wait for the latest completed check, so an earlier failure cannot satisfy recovery setup.
    def expect_failed_check(job)
      wait_for { Abgleich.job_logs(job).last }.to include('level' => 'error', 'error' => a_string_matching(/.+/))
    end

    # Skipped, rejected, and unconfirmed updates must not be reported as accepted.
    def expect_no_accepted_rebases
      expect(Abgleich.rebase_logs).to be_empty, 'expected no accepted rebase requests in the worker logs'
    end

    # Failed or unconfirmed merge operations must not be logged as accepted.
    def expect_no_accepted_merge_actions
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
