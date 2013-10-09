require 'spec_helper'
require 'httpclient'
require 'json'

describe "functioning as a reverse proxy" do

  describe "header handling" do
    start_backend_around_all :port => 3163, :type => :echo
    before :each do
      add_backend "backend", "http://localhost:3163/"
      add_route "/foo", "backend"
      reload_routes
    end

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
      pending "Host header munging hasn't been completed yet"
      response = HTTPClient.get(router_url("/foo"), :header => {"Host" => "www.example.com"})
      data = JSON.parse(response.body)

      expect(data["Request"]["Host"]).to eq("localhost:3163")
    end

    it "should add the client IP to X-Forwardrd-For" do
      response = HTTPClient.get(router_url("/foo"))
      headers = JSON.parse(response.body)["Request"]["Header"]
      expect(headers["X-Forwarded-For"].first).to eq("127.0.0.1")

      response = HTTPClient.get(router_url("/foo"), :header => {"X-Forwarded-For" => "10.9.8.7"})
      headers = JSON.parse(response.body)["Request"]["Header"]
      expect(headers["X-Forwarded-For"].first).to eq("10.9.8.7, 127.0.0.1")
    end
  end
end
