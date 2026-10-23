# frozen_string_literal: true

Given('GitHub cannot add comment reactions because of {string}') do |problem|
  failure = { 'an API failure' => 'reaction HTTP', 'a slow response' => 'reaction timeout',
              'a mismatched response' => 'mismatched reaction', 'an incomplete response' => 'incomplete reaction' }.fetch(problem)
  Abgleich.github.fail_with(failure)
end

Given('the command has a short processing deadline') do
  config = YAML.safe_load_file('.config/server.yaml')
  config['github']['webhook']['timeout'] = '200ms'
  File.write('.config/server.yaml', YAML.dump(config))
end

Given('GitHub takes longer than one second to acknowledge the command outcome') do
  Abgleich.github.fail_with('slow final reaction')
end

When('a maintainer requests auto-merge while GitHub is still applying the label') do
  @comment_event = Abgleich.github.add_comment('contribution')
  Nonnative.start
  delivery = Thread.new { Abgleich.deliver_comment(@comment_event) }
  comment_id = @comment_event.fetch(:comment).fetch('id')
  wait_for { Abgleich.github.comment_reactions(comment_id) }.to include('eyes')
  @processing_reactions = Abgleich.github.comment_reactions(comment_id)
  @command_pending = delivery.alive?
  @comment_response = delivery.value
ensure
  delivery&.join
end

Then('receipt should have been visible before the command completed') do
  expect(@processing_reactions).to eq(['eyes']), 'expected only the receipt reaction while the command was processing'
  expect(@command_pending).to be(true), 'expected the delivery to be waiting for label processing when receipt became visible'
end

Then('the command comment should show {string} feedback') do |outcome|
  expected = { 'receipt and acceptance' => %w[eyes +1], 'receipt and rejection' => %w[eyes confused],
               'receipt' => ['eyes'], 'no' => [] }.fetch(outcome)
  comment_id = @comment_event.fetch(:comment).fetch('id')
  reactions = Abgleich.github.comment_reactions(comment_id)
  expect(reactions).to match_array(expected), "expected #{outcome} feedback on comment #{comment_id}, got #{reactions}"
end

Then('the reaction failure should be logged') do
  expected_warning = { 'msg' => 'github comment reaction failed', 'level' => 'warn', 'error' => a_string_matching(/.+/) }
  expect(Abgleich.logs).to include(a_hash_including(expected_warning)), 'expected a warning explaining the reaction failure'
end
