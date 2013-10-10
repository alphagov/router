require 'spec_helper'
require 'httpclient'
require 'json'

describe "functioning as a redirector" do

  start_backend_around_all :port => 3165, :type => :echo

  before :each do
    add_backend "backend-redirect", "http://localhost:3165/"
  end

  describe "basic redirection" do
    before :all do
      add_route "/foo", "backend-redirect", :redirect_to => "/bar"
      reload_routes
    end

    it "should issue a Location header for the target" do
      pending "Not yet implemented"

      response = HTTPClient.get(router_url("/foo"))

      expect(response.headers["Location"]).to_eq("/bar")
    end

    it "should issue a permanent redirect by default" do
      pending "Not yet implemented"

      response = HTTPClient.get(router_url("/foo"))

      response.status.should == 301
    end
  end

  describe "friendly URL redirection" do
    before :all do
      add_route "/xyz", "backend-redirect", :redirect_to => "/abc", :temporary_redirect => true
    end

    it "should issue a temporary redirect when required" do
      pending "Not yet implemented"

      response = HTTPClient.get(router_url("/xyz"))

      response.status.should == 302
    end
  end
end
