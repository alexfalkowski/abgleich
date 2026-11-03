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

    # Review kinds keep their distinct command and result identities in shared steps.
    def review_command(kind)
      { 'code' => 'codereview', 'security' => 'secreview' }.fetch(kind)
    end

    def review_heading(kind)
      "### 🤖 Abgleich #{kind} review\n"
    end

    def reviews_for(kind)
      Abgleich.github.review_results.select { |review| review.fetch(:body).start_with?(review_heading(kind)) }
    end

    # Wait for publication before inspecting a successful result's contents.
    def wait_for_review(kind)
      wait_for { reviews_for(kind) }.not_to be_empty, "expected an advisory #{kind} review"
      reviews_for(kind).last
    end

    def summaries
      Abgleich.github.replies.select { |reply| reply.fetch(:body).start_with?("### 🤖 Abgleich summary\n") }
    end

    def wait_for_summary
      wait_for { summaries }.not_to be_empty, 'expected an App summary'
      summaries.last
    end

    # Reading links keep the instruction below its file at the reviewed head.
    def summary_file_location(path)
      repository = Abgleich::GitHubRepository::DEFAULT_REPOSITORY
      sha = Abgleich.github.pull_state('contribution').dig(:head, :sha)
      "- [#{path}](https://github.com/#{repository}/blob/#{sha}/#{path})\\\n  "
    end

    def expect_inert_review_text(body)
      expect(body).to include('&lt;script&gt;', "@\u200beveryone", '\\[click\\]')
      expect(body).not_to include('<script>', '@everyone', '[click](')
      expect(body.delete('\\')).to include('api_key=example', 'token=example')
    end

    def expect_review_usage(body, kind)
      model = { 'code' => 'test\\-review', 'security' => 'test\\-security' }.fetch(kind)
      expect(body).to include(model, '100', '40'), 'expected provider model and actual token usage'
      expect(body).not_to include('$', 'Estimated token cost')
    end

    def expect_review_scope(body)
      expect(body).to match(/patch/i), 'expected the review to identify its changed-file context'
      expect(body).to include('read-only lookups', 'Code and tests were not executed', 'Findings are advisory')
      expect(body).not_to include('Citations were checked'), 'expected no claim of source citation validation'
    end

    # Translate human source descriptions to the fixture evidence checked by the harness.
    def source_evidence(description)
      {
        'the final changed file' => 'File: file21.go', 'the missing patch notice' => 'Patch unavailable from GitHub.',
        'the end of the combined changes' => 'test.go-patch-end',
        'the end of the large change' => 'main.go-patch-end',
        'the change to main.go' => 'File: main.go', 'the change to test.go' => 'File: test.go',
        'the updated Worker implementation' => 'func Worker() bool { return false }',
        'the updated Lookup implementation' => 'func Lookup(query string)'
      }.fetch(description)
    end

    # Observe this command's scheduled work before asserting absent results or source.
    def expect_review_check(kind)
      command = review_command(kind)
      next_check = Abgleich.job_logs(command).length + 1
      wait_for { Abgleich.job_logs(command).length }.to be >= next_check, "expected a completed #{kind} review check"
    end

    def wait_for_review_cancellation(kind)
      wait_for { Abgleich.github.review_cancellations(kind) }.not_to be_empty, "expected a #{kind} review cancellation"
      Abgleich.github.review_cancellations(kind).last.fetch(:body)
    end

    # Advisory requests must leave merge automation, labels, and branches untouched.
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
