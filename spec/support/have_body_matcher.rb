require 'rspec/expectations'

RSpec::Matchers.define :have_response_body do |expected|
  match do |actual|
    body_string(actual) == expected
  end

  failure_message_for_should do |actual|
    "expected response body '#{expected}', got '#{body_string(actual)}'"
  end

  def body_string(response)
    response.body.strip
  end
end
