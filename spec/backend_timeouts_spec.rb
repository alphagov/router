require 'spec_helper'

describe "timeouts on backend connections" do

  describe "connect timeout" do
    # This test requires a firewall block rule for connections to localhost:3170
    # This is necessary to simulate a connection timeout

    start_router_around_all :port => 3167, :api_port => 3166, :extra_env => {"ROUTER_BACKEND_CONNECT_TIMEOUT" => "0.3s"}

    before :each do
      add_backend "firewall-blocked", "http://localhost:3170/"
      add_backend_route "/blocked", "firewall-blocked"
      reload_routes(3166)
    end

    it "should return a 504 if the connection times out in the configured time" do
      unless ENV["RUN_FIREWALL_DEPENDENT_TESTS"]
        pending "Need firewall block rule"
      end

      start = Time.now
      response = router_request("/blocked", :port => 3167)
      duration = Time.now - start

      expect(response.code).to eq(504)
      expect(duration).to be_within(0.11).of(0.4) # Expect between 0.29 and 0.51
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

    it "should return a 504 if a backend takes longer than the configured response timeout to start returning a response" do
      response = router_request("/tarpit1", :port => 3167)
      expect(response.code).to eq(504)
    end

    it "should still return the response if the body takes longer than the header timeout" do
      response = router_request("/tarpit2", :port => 3167)
      expect(response.code).to eq(200)
      expect(response).to have_response_body("Tarpit")
    end
  end
end
