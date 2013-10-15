require 'spec_helper'
require 'httpclient'

describe "reload API endpoint" do

  before :each do
    reload_routes
  end

  describe "request handling" do
    it "should return 200 for POST /" do
      response = HTTPClient.post(api_url("/"))
      expect(response.status).to eq(200)
    end

    it "should return 404 for GET /" do
      response = HTTPClient.get(api_url("/"))
      expect(response.status).to eq(404)
    end
  end

end
