# frozen_string_literal: true

Given('every configured review has already completed') do
  Nonnative.start
  event = Abgleich.github.add_comment('contribution', body: '/ab review')
  expect(Abgleich.deliver_comment(event).code).to eq(204), 'expected the aggregate review request to be accepted'
  %w[code security].each { |kind| wait_for_requested_review(kind, event) }
end

Given('the code review queue is full while both reviews are enabled') do
  Abgleich.openai.pause_responses
  4.times { |index| queue_branch_review('code', "code-review-queued-#{index}") }
  wait_for { Abgleich.openai.requests.length }.to eq(1), 'expected an active code review with three pending requests'
end

Given('the aggregate review is waiting behind active reviews') do
  Abgleich.openai.pause_responses
  queue_branch_review('code', 'busy-code-review')
  queue_branch_review('security', 'busy-security-review')
  wait_for { Abgleich.openai.requests.length }.to eq(2), 'expected both independent review jobs to be busy'
  @comment_event = Abgleich.github.add_comment('contribution', body: '/ab review')
  expect(Abgleich.deliver_comment(@comment_event).code).to eq(204), 'expected both reviews to be queued'
end

When('GitHub delivers the same aggregate review request twice while reviews are running') do
  Abgleich.openai.pause_responses
  Nonnative.start
  @comment_event = Abgleich.github.add_comment('contribution', body: '/ab review')
  expect(Abgleich.deliver_comment(@comment_event).code).to eq(204), 'expected both reviews to be queued'
  wait_for { Abgleich.openai.requests.length }.to eq(2), 'expected both reviews to be running before redelivery'
  expect(Abgleich.deliver_comment(@comment_event).code).to eq(204), 'expected redelivery to reuse the running reviews'
  Abgleich.openai.resume_responses
end

When('a maintainer requests all reviews while GitHub authorization is slow') do
  Nonnative.start
  Abgleich.github.fail_with('slow review authorization')
  @comment_event = Abgleich.github.add_comment('contribution', body: '/ab review')
  response = Abgleich.deliver_comment(@comment_event)
  expect(response.code).to eq(204), 'expected all reviews to be accepted within the webhook deadline'
ensure
  Abgleich.github.fail_with(nil)
end

When('maintainers request reviews while status and another review request are in progress') do
  @review_deliveries = all_review_deliveries
  status = Abgleich.github.add_comment('contribution', body: '/ab status')
  first_review = Abgleich.github.add_comment('contribution', body: '/ab review')
  Nonnative.start

  while_webhooks_wait_for_github([status, [:comment, status.fetch(:comment).fetch('id')]],
                                 [first_review, [:discussion, first_review.fetch(:issue).fetch(:number)]]) do
    @review_deliveries.each do |branch, _kinds, event|
      expect(Abgleich.deliver_comment(event).code).to eq(204), "expected the review request for #{branch} to be accepted during the overlap"
    end
  end

  expect(Abgleich.github.comment_reactions(status.fetch(:comment).fetch('id'))).to eq(%w[eyes +1])
  @review_deliveries.unshift(['contribution', %w[code security test], first_review])
end

When('two review requests are waiting for GitHub and maintainers submit more commands') do
  @review_deliveries = all_review_deliveries
  pending, competing = @review_deliveries.partition { |command, _kinds, _event| %w[review codereview].include?(command) }
  paused = pending.map { |_command, _kinds, event| [event, [:discussion, event.fetch(:issue).fetch(:number)]] }
  status = Abgleich.github.add_comment('contribution', body: '/ab status')
  Nonnative.start

  while_webhooks_wait_for_github(*paused) do
    competing.each do |command, _kinds, event|
      response = Abgleich.deliver_comment(event)
      expect(response.code).to eq(503), "expected /ab #{command} to be declined while both review slots are occupied"
      paths = Abgleich.github.request_paths_for(response.headers.fetch(:request_id))
      expect(paths).to be_empty, "expected rejected /ab #{command} to make no GitHub calls"
      expect(Abgleich.github.comment_reactions(event.fetch(:comment).fetch('id'))).to be_empty, "expected no feedback for /ab #{command}"
    end
    expect(Abgleich.deliver_comment(status).code).to eq(204), 'expected status to be available during review overload'
    expect(Abgleich.github.comment_reactions(status.fetch(:comment).fetch('id'))).to eq(%w[eyes +1])
  end

  competing.each do |command, _kinds, event|
    response = Abgleich.deliver_comment(event)
    expect(response.code).to eq(204), "expected /ab #{command} to be accepted after the review slots reopened"
    paths = Abgleich.github.request_paths_for(response.headers.fetch(:request_id))
    expect(paths).to include("/app/installations/#{Abgleich::INSTALLATION_ID}"), "expected installation validation for /ab #{command}"
  end
end

Then('every requested review should be published') do
  @review_deliveries.each do |branch, kinds, event|
    comment_id = event.fetch(:comment).fetch('id')
    expect(Abgleich.github.comment_reactions(comment_id)).to eq(%w[eyes +1]), "expected acceptance feedback for #{branch}"
    pull = Abgleich.github.pull_state(branch)
    kinds.each do |kind|
      body = wait_for_requested_review(kind, event).fetch(:body)
      expect(body).to include(pull.dig(:head, :sha), pull.dig(:base, :sha))
    end
  end
end

Then('each configured review should have exactly one result') do
  %w[code security].each do |kind|
    expect_review_check(kind)
    expect(reviews_for(kind).length).to eq(1), "expected one #{kind} result across individual and aggregate requests"
  end
end

Then('the security review should complete after the active reviews finish') do
  Abgleich.openai.resume_responses
  wait_for_requested_review('security', @comment_event)
  wait_for { reviews_for('code').length }.to eq(4), 'expected every earlier accepted code review to finish'
  expect_review_check('code')
  request = "#issuecomment-#{@comment_event.fetch(:comment).fetch('id')}"
  expect(reviews_for('code')).not_to include(a_hash_including(body: a_string_including(request)))
end
