require 'spec_helper'

describe "timeouts on backend connections" do

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
