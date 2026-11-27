# frozen_string_literal: true

module Abgleich
  # Exercise competing signed deliveries while the fake GitHub API holds selected requests.
  module WebhookAssertions
    def while_webhooks_wait_for_github(*requests)
      deliveries = []
      Abgleich.github.with_paused_requests(*requests.map(&:last)) do
        deliveries = requests.map { |event, _request| Thread.new { Abgleich.deliver_comment(event) } }
        wait_for_webhook_requests(requests)
        yield
        expect(deliveries).to all(be_alive), 'expected the earlier deliveries to be waiting for GitHub'
      end
      expect_webhook_deliveries_to_finish(deliveries)
    ensure
      deliveries.each(&:join)
    end

    private

    def wait_for_webhook_requests(requests)
      requests.each { |request| Abgleich.github.wait_for_paused_request(*request.last) }
    end

    def expect_webhook_deliveries_to_finish(deliveries)
      expect(deliveries.map { |delivery| delivery.value.code }).to all(eq(204)), 'expected the earlier deliveries to finish after release'
    end
  end
end
