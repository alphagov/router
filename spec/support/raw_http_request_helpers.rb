require 'socket'

module RawHTTPRequestHelpers

  def raw_http_request(url, headers = {})
    http_version = headers.delete(:http_version) || "1.1"
    uri = URI.parse(url)
    s = TCPSocket.new(uri.host, uri.port)
    s.write("GET #{uri.request_uri} HTTP/#{http_version}\r\n")
    headers.each { |k, v| s.write("#{k}: #{v}\r\n") }
    s.write("\r\n")
    s.close_write

    headers, body, found_body = [], "", false
    s.each_line do |line|
      if !found_body and line.strip == ""
        found_body = true
        next
      end
      if found_body
        body << line
      else
        headers << line.strip
      end
    end
    [headers, body]
  ensure
    s.close if s
  end

  def raw_http_1_0_request(url, headers = {})
    raw_http_request(url, headers.merge(:http_version => "1.0"))
  end
end

RSpec.configuration.include(RawHTTPRequestHelpers)
