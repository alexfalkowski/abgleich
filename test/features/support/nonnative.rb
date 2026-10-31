# frozen_string_literal: true

Nonnative.configure do |config|
  config.load_file('nonnative.yml')

  # Each scenario waits for graceful worker shutdown before starting another process.
  server = config.process_by_name('server')
  server.environment = server.environment.to_h.merge('GORACE' => "#{ENV.fetch('GORACE', '')} atexit_sleep_ms=0".strip)
end

Around do |_scenario, run|
  Nonnative.clear_pool
  Nonnative.configuration.process_by_name('server').log = "reports/server-#{SecureRandom.uuid}.log"
  run.call
end
