# frozen_string_literal: true

Given('AI summaries and reviews use separate provider connections') do
  Abgleich.prepare_github_webhook
  Abgleich.github.add_repository(Abgleich::GitHubRepository::DEFAULT_REPOSITORY)
  Abgleich.github.file_mode = 'code review'
  Abgleich.use_config('ai_connections')
end

Given('the AI connection selection has {string}') do |problem|
  fixture = { 'an unknown connection' => 'unknown', 'a missing connection' => 'missing',
              'invalid summary effort' => 'invalid_effort' }.fetch(problem)
  Abgleich.use_config("ai_connections_#{fixture}")
end

When('a maintainer requests all AI assessments') do
  Nonnative.start
  %w[review summarise].each do |command|
    event = Abgleich.github.add_comment('contribution', body: "/ab #{command}")
    expect(Abgleich.deliver_comment(event).code).to eq(204), "expected #{command} to be accepted"
  end
end

Then('every assessment should use its selected connection, model, and effort') do
  results = { 'codereview' => wait_for_review('code'), 'secreview' => wait_for_review('security'),
              'summarise' => wait_for_summary }
  selections = { 'codereview' => %w[openai test-review high /openai],
                 'secreview' => %w[secondary test-security medium /openai-secondary],
                 'summarise' => %w[secondary test-summary low /openai-secondary] }
  deliveries = Abgleich.openai.requests.zip(Abgleich.openai.endpoints)

  selections.each do |task, (provider, model, effort, endpoint)|
    request, actual_endpoint = deliveries.find { |payload, _path| payload.fetch('model') == model }
    expect(request).not_to be_nil, "expected #{task} to reach its selected model"
    expect(actual_endpoint).to eq(endpoint), "expected #{task} to reach #{provider}'s endpoint"
    expect(request.fetch('reasoning')).to include('effort' => effort), "expected #{task}'s configured effort"
    expect(results.fetch(task).fetch(:body).delete('\\')).to include(provider, model), "expected #{task} to identify its provider and model"
    expect(Abgleich.logs).to include(a_hash_including('msg' => 'ai generation', 'task' => task, 'provider' => provider, 'model' => model))
  end
  expect(deliveries.length).to eq(3), 'expected one inference request for each accepted assessment'
end
