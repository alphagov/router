require 'json'

module LogHelpers
  LOGFILE = "/tmp/router.error.json"

  def last_error_log_line
    `tail -n 1 #{LOGFILE.shellescape}`.chomp
  end

  def last_error_log_details
    JSON.parse(last_error_log_line)
  end
end

RSpec.configuration.include(LogHelpers)
