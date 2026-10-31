# frozen_string_literal: true

When('an operator checks worker {string}') do |name|
  opts = {
    headers: { request_id: SecureRandom.uuid },
    read_timeout: 10, open_timeout: 10
  }

  @response = Nonnative.observability.send(name, opts)
end

Then('the worker should report healthy') do
  expect(@response.code).to eq(200)
  expect(@response.body.strip).to eq('SERVING')
end

Then('the operator should receive monitoring metrics') do
  expect(@response.code).to eq(200)
  expect(@response.body).to include('go_info')
end
