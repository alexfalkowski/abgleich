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
