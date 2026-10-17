# frozen_string_literal: true

Nonnative.configure do |config|
  config.load_file('nonnative.yml')
end

Around do |_scenario, run|
  Nonnative.clear_pool
  Nonnative.configuration.process_by_name('server').log = "reports/server-#{SecureRandom.uuid}.log"
  run.call
end
