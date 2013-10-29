require 'spec_helper'
require 'httpclient'
require 'json'

describe "functioning as a reverse proxy" do

  start_backend_around_all :port => 3163, :type => :echo
  before :each do
    add_backend "backend", "http://localhost:3163/"
    add_backend_route "/foo", "backend", :prefix => true
    reload_routes
  end

  describe "connecting to the backend" do

    it "should log and return 502 if the connection to the backend is refused" do
      add_backend "not-running", "http://localhost:3164/"
      add_backend_route "/not-running", "not-running"
      reload_routes

      response = HTTPClient.get(router_url("/not-running"), :header => {
        "X-Varnish" => "12345678",
      })
      expect(response.code).to eq(502)

      log_details = last_router_error_log_entry
      expect(log_details["@fields"]).to eq({
        "error" => "dial tcp 127.0.0.1:3164: connection refused",
        #"remote_addr" => "127.0.0.1",
        #"request" => "GET /not-running HTTP/1.1",
        #"request_method" => "GET",
        "status" => 502,
        #"upstream_addr" => "localhost:3164",
        #"varnish_id" => "12345678",
      })
      expect(Time.parse(log_details["@timestamp"]).to_i).to be_within(5).of(Time.now.to_i)
    end

    describe "handling connect timeout" do
      # This test requires a firewall block rule for connections to localhost:3170
      # This is necessary to simulate a connection timeout

      start_router_around_all :port => 3167, :api_port => 3166, :extra_env => {"ROUTER_BACKEND_CONNECT_TIMEOUT" => "0.3s"}

      before :each do
        add_backend "firewall-blocked", "http://localhost:3170/"
        add_backend_route "/blocked", "firewall-blocked"
        reload_routes(3166)
      end

      it "should log and return a 504 if the connection times out in the configured time" do
        unless ENV["RUN_FIREWALL_DEPENDENT_TESTS"]
          pending "Need firewall block rule"
        end

        start = Time.now
        response = router_request("/blocked", :port => 3167)
        duration = Time.now - start

        expect(response.code).to eq(504)
        expect(duration).to be_within(0.11).of(0.4) # Expect between 0.29 and 0.51

        log_details = last_router_error_log_entry
        expect(log_details["@fields"]).to eq({
          "error" => "dial tcp 127.0.0.1:3170: i/o timeout",
          #"remote_addr" => "127.0.0.1",
          #"request" => "GET /not-running HTTP/1.1",
          #"request_method" => "GET",
          "status" => 504,
          #"upstream_addr" => "localhost:3164",
          #"varnish_id" => "12345678",
        })
        expect(Time.parse(log_details["@timestamp"]).to_i).to be_within(5).of(Time.now.to_i)
      end
    end

    describe "response header timeout" do
      start_router_around_all :port => 3167, :api_port => 3166, :extra_env => {"ROUTER_BACKEND_HEADER_TIMEOUT" => "0.3s"}
      start_backend_around_all :port => 3160, :type => :tarpit, "response-delay" => "1s"
      start_backend_around_all :port => 3161, :type => :tarpit, "response-delay" => "0.1s", "body-delay" => "0.5s"

      before :each do
        add_backend "tarpit1", "http://localhost:3160/"
        add_backend "tarpit2", "http://localhost:3161/"
        add_backend_route "/tarpit1", "tarpit1"
        add_backend_route "/tarpit2", "tarpit2"
        reload_routes(3166)
      end

      it "should log and return a 504 if a backend takes longer than the configured response timeout to start returning a response" do
        response = router_request("/tarpit1", :port => 3167)
        expect(response.code).to eq(504)

        log_details = last_router_error_log_entry
        expect(log_details["@fields"]).to eq({
          "error" => "net/http: timeout awaiting response headers",
          #"remote_addr" => "127.0.0.1",
          #"request" => "GET /not-running HTTP/1.1",
          #"request_method" => "GET",
          "status" => 504,
          #"upstream_addr" => "localhost:3164",
          #"varnish_id" => "12345678",
        })
        expect(Time.parse(log_details["@timestamp"]).to_i).to be_within(5).of(Time.now.to_i)
      end

      it "should still return the response if the body takes longer than the header timeout" do
        response = router_request("/tarpit2", :port => 3167)
        expect(response.code).to eq(200)
        expect(response).to have_response_body("Tarpit")
      end
    end
  end

  describe "header handling" do
    it "should pass through most http headers to the backend" do
      response = HTTPClient.get(router_url("/foo"), :header => {
        "Foo" => "bar",
        "User-Agent" => "Router test suite 2.7182",
      })
      headers = JSON.parse(response.body)["Request"]["Header"]

      expect(headers["Foo"].first).to eq("bar")
      expect(headers["User-Agent"].first).to eq("Router test suite 2.7182")
    end

    it "should set the Host header to the backend hostname" do
      response = HTTPClient.get(router_url("/foo"), :header => {"Host" => "www.example.com"})
      data = JSON.parse(response.body)

      expect(data["Request"]["Host"]).to eq("localhost:3163")
    end

    it "should not add a default User-Agent if there isn't one in the request" do
      # Most http libraries add a default User-Agent header.
      headers, body = raw_http_request(router_url("/foo"), "Host" => "localhost")
      data = JSON.parse(body)

      expect(data["Request"]["Header"]["User-Agent"]).to be_nil
    end

    it "should add the client IP to X-Forwardrd-For" do
      response = HTTPClient.get(router_url("/foo"))
      headers = JSON.parse(response.body)["Request"]["Header"]
      expect(headers["X-Forwarded-For"].first).to eq("127.0.0.1")

      response = HTTPClient.get(router_url("/foo"), :header => {"X-Forwarded-For" => "10.9.8.7"})
      headers = JSON.parse(response.body)["Request"]["Header"]
      expect(headers["X-Forwarded-For"].first).to eq("10.9.8.7, 127.0.0.1")
    end

    describe "setting the Via header" do
      # See https://tools.ietf.org/html/rfc2616#section-14.45

      it "should add itself to the Via request header for an HTTP/1.1 request" do
        response = HTTPClient.get(router_url("/foo"))
        headers = JSON.parse(response.body)["Request"]["Header"]
        expect(headers["Via"].first).to eq("1.1 router")

        response = HTTPClient.get(router_url("/foo"), :header => {"Via" => "1.0 fred, 1.1 barney"})
        headers = JSON.parse(response.body)["Request"]["Header"]
        expect(headers["Via"].first).to eq("1.0 fred, 1.1 barney, 1.1 router")
      end

      it "should add itself to the Via request header for an HTTP/1.0 request" do
        headers, body = raw_http_1_0_request(router_url("/foo"))
        headers = JSON.parse(body)["Request"]["Header"]
        expect(headers["Via"].first).to eq("1.0 router")

        headers, body = raw_http_1_0_request(router_url("/foo"), "Via" => "1.0 fred, 1.1 barney")
        headers = JSON.parse(body)["Request"]["Header"]
        expect(headers["Via"].first).to eq("1.0 fred, 1.1 barney, 1.0 router")
      end

      it "should add itself to the Via response heaver" do
        response = HTTPClient.get(router_url("/foo"))
        expect(response.headers["Via"]).to eq("1.1 router")

        response = HTTPClient.get(router_url("/foo?simulate_response_via=1.0+fred,+1.1+barney"))
        expect(response.headers["Via"]).to eq("1.0 fred, 1.1 barney, 1.1 router")

        headers, body = raw_http_1_0_request(router_url("/foo"))
        # The version here needs to be the version of the backend response to the
        # router, not the original request
        expect(headers.find {|h| h =~ /\AVia: / }).to eq("Via: 1.1 router")
      end
    end
  end

  describe "request verb, path, quesy and body handling" do
    it "should use the same verb when proxying" do
      response = HTTPClient.post(router_url("/foo"))
      request_data = JSON.parse(response.body)["Request"]
      expect(request_data["Method"]).to eq("POST")

      response = HTTPClient.delete(router_url("/foo"))
      request_data = JSON.parse(response.body)["Request"]
      expect(request_data["Method"]).to eq("DELETE")
    end

    it "should pass through the request path unmodified" do
      response = HTTPClient.post(router_url("/foo/bar/baz.json"))
      request_data = JSON.parse(response.body)["Request"]
      expect(request_data["RequestURI"]).to eq("/foo/bar/baz.json")
    end

    it "should pass through the query string unmodified" do
      response = HTTPClient.post(router_url("/foo/bar?baz=qux"))
      request_data = JSON.parse(response.body)["Request"]
      expect(request_data["RequestURI"]).to eq("/foo/bar?baz=qux")
    end

    it "should pass through the request body unmodified" do
      response = HTTPClient.post(router_url("/foo"), :body => "I am the request body.  Woohoo!")
      data = JSON.parse(response.body)
      expect(data["Body"]).to eq("I am the request body.  Woohoo!")
    end
  end

  describe "handling a backend with a non '/' path" do
    before :each do
      add_backend "backend-with-path", "http://localhost:3163/something"
      add_backend_route "/foo/bar", "backend-with-path", :prefix => true
      reload_routes
    end

    it "should merge the 2 paths" do
      response = HTTPClient.get(router_url("/foo/bar"))
      request_data = JSON.parse(response.body)["Request"]
      expect(request_data["RequestURI"]).to eq("/something/foo/bar")
    end

    it "should preserve the request query string" do
      response = HTTPClient.get(router_url("/foo/bar?baz=qux"))
      request_data = JSON.parse(response.body)["Request"]
      expect(request_data["RequestURI"]).to eq("/something/foo/bar?baz=qux")
    end
  end

  describe "supporting http/1.0 requests" do
    it "should work with incoming http/1.0 requests" do
      headers, body = raw_http_1_0_request(router_url("/foo"), "Host" => "www.example.com")

      expect(headers.first).to eq("HTTP/1.0 200 OK")
      request_details = JSON.parse(body)["Request"]
      expect(request_details["RequestURI"]).to eq("/foo")
    end

    it "should not require a Host header" do
      headers, body = raw_http_1_0_request(router_url("/foo"))

      expect(headers.first).to eq("HTTP/1.0 200 OK")
      request_details = JSON.parse(body)["Request"]
      expect(request_details["RequestURI"]).to eq("/foo")
    end

    it "should proxy them to the backend as http/1.1 requests" do
      headers, body = raw_http_1_0_request(router_url("/foo"), "Host" => "www.example.com")
      data = JSON.parse(body)

      expect(data["Request"]["Proto"]).to eq("HTTP/1.1")
    end
  end
end
