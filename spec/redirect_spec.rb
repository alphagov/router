require 'spec_helper'

describe "Redirection" do

  describe "simple exact redirect" do
    before :each do
      add_redirect_route("/foo", "/bar")
      add_redirect_route("/foo-temp", "/bar", :redirect_type => 'temporary')
      reload_routes
    end

    it "should redirect permanently by default" do
      response = router_request("/foo")
      expect(response.code).to eq(301)
    end

    it "should contain the redirect location" do
      response = router_request("/foo")
      expect(response.headers['Location']).to eq("/bar")
    end

    it "should redirect temporarily when asked to" do
      response = router_request("/foo-temp")
      expect(response.code).to eq(302)
    end
  end

  describe "prefix redirects" do
    before :each do
      add_redirect_route("/foo", "/bar", :prefix => true)
      reload_routes
    end

    it "should skip prefix routes" do
      response = router_request("/foo")
      expect(response.code).to eq(404)
    end
  end
end
