require 'spec_helper'
require 'httpclient'

describe "reload API endpoint" do

  describe "request handling" do
    it "should return 200 for POST /reload" do
      response = HTTPClient.post(api_url("/reload"))
      expect(response.status).to eq(200)
    end

    it "should return 404 for POST /foo" do
      response = HTTPClient.post(api_url("/foo"))
      expect(response.status).to eq(404)
    end

    it "should return 404 for POST /reload/foo" do
      response = HTTPClient.post(api_url("/reload/foo"))
      expect(response.status).to eq(404)
    end

    it "should return 405 for GET /reload" do
      response = HTTPClient.get(api_url("/reload"))
      expect(response.status).to eq(405)
      expect(response.reason).to eq("Method Not Allowed")
      expect(response.headers["Allow"]).to eq("POST")
    end
  end

  describe "healthcheck" do
    it "should respond with 200 and string 'OK' on /healthcheck" do
      response = HTTPClient.get(api_url("/healthcheck"))
      expect(response.status).to eq(200)
      expect(response).to have_response_body('OK')
    end

    it "should respond with 405 for other verbs" do
      response = HTTPClient.post(api_url("/healthcheck"))
      expect(response.status).to eq(405)
      expect(response.reason).to eq("Method Not Allowed")
      expect(response.headers["Allow"]).to eq("GET")
    end
  end

  describe "route stats" do
    context "with some routes loaded" do
      before :each do
        add_redirect_route("/foo", "/bar", :prefix => true)
        add_redirect_route("/baz", "/qux", :prefix => true)
        add_redirect_route("/foo", "/bar/baz", :prefix => false)
        reload_routes

        response = HTTPClient.get(api_url("/stats"))
        expect(response.status).to eq(200)
        @data = JSON.parse(response.body)
      end

      it "should return the number of routes loaded" do
        expect(@data["routes"]["count"]).to eq(3)
      end

      it "should return a checksum calculated from the sorted paths and route_types" do
        s = Digest::SHA1.new
        s << "/baz(true)"
        s << "/foo(false)"
        s << "/foo(true)"
        expect(@data["routes"]["checksum"]).to eq(s.hexdigest)
      end
    end

    context "with no routes" do
      before :each do
        reload_routes

        response = HTTPClient.get(api_url("/stats"))
        expect(response.status).to eq(200)
        @data = JSON.parse(response.body)
      end

      it "should return the number of routes loaded" do
        expect(@data["routes"]["count"]).to eq(0)
      end

      it "should return a checksum of empty string" do
        expected = Digest::SHA1.hexdigest("")
        expect(@data["routes"]["checksum"]).to eq(expected)
      end
    end

    it "should respond with 405 for other verbs" do
      response = HTTPClient.post(api_url("/stats"))
      expect(response.status).to eq(405)
      expect(response.reason).to eq("Method Not Allowed")
      expect(response.headers["Allow"]).to eq("GET")
    end
  end
end
