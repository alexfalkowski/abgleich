# frozen_string_literal: true

module Abgleich
  # Review request setup and checks shared by Cucumber step definitions.
  module ReviewContext
    def queue_review(kind, branch: 'contribution')
      Nonnative.start
      event = Abgleich.github.add_comment(branch, body: "/ab #{review_command(kind)}")
      expect(Abgleich.deliver_comment(event).code).to eq(204), "expected the review for #{branch} to be queued"
      event
    end

    def queue_branch_review(kind, branch)
      Abgleich.github.add_pull(branch, auto_merge: false, current: true)
      queue_review(kind, branch:)
    end

    def hold_review_worker(kind)
      Abgleich.openai.pause_responses
      queue_branch_review(kind, 'busy-review')
      wait_for { Abgleich.openai.requests.length }.to eq(1), 'expected the earlier review to be running'
    end

    def complete_review_request(kind)
      @comment_event = queue_review(kind)
      review = wait_for_requested_review(kind, @comment_event)
      @existing_review_body = review.fetch(:body)
      @native_review_history = Abgleich.github.native_reviews
      @completed_review_requests = Abgleich.openai.requests.length
    end

    def wait_for_requested_review(kind, event)
      request = "#issuecomment-#{event.fetch(:comment).fetch('id')}"
      review = nil
      wait_for do
        review = reviews_for(kind).find { |result| result[:viewer_did_author] && result[:body].include?(request) }
      end.not_to be_nil, "expected the App's #{kind} review for the current request"
      expect_review_check(kind)
      review
    end

    def fill_review_queue(kind)
      Abgleich.use_config("#{review_command(kind)}_pending")
      Abgleich.openai.pause_responses
      4.times { |index| queue_branch_review(kind, "#{kind}-review-queued-#{index}") }
      wait_for { Abgleich.openai.requests.length }.to eq(1), 'expected one active review while the remaining requests wait'
    end

    def interrupt_review_request(kind)
      Abgleich.use_config("#{review_command(kind)}_pending")
      hold_review_worker(kind)
      @comment_event = queue_review(kind)

      Abgleich.restart do
        Abgleich.openai.reset
        Abgleich.use_config(review_command(kind))
      end

      expect_lost_review_requests(kind)
    end

    def expect_lost_review_requests(kind)
      expect_review_check(kind)
      expect(Abgleich.openai.requests).to be_empty, "expected restart to discard pending #{kind} reviews"
      expect(Abgleich.github.review_results).to be_empty, 'expected no reply before a maintainer reissues the command'
    end

    def expect_review_failure(kind)
      wait_for { Abgleich.github.review_results.any? { |reply| reply[:body].include?("#{kind.capitalize} review failed") } }.to be(true)
      expect(reviews_for(kind)).to be_empty, 'expected no completed review after failure'
      expect_healthy_worker
      expect_no_automation_mutations
    end
  end
end
