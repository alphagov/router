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

end
