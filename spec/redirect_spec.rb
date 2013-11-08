require 'spec_helper'
require 'time'

describe "Redirection" do
  CACHE_EXPIRES_PERIOD = 86_400 # 24 hours in seconds

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

    it "should not preserve the query string" do
      response = router_request("/foo?baz=qux")
      expect(response.headers['Location']).to eq("/bar")
    end

    it "should contain cache headers of 24hrs" do
      response = router_request("/foo")
      expect(response.headers['Cache-Control']).to eq("max-age=86400, public")
      expires = Time.parse(response.headers['Expires'])
      expect(expires).to be_within(10).of(Time.now + CACHE_EXPIRES_PERIOD)
    end
  end

  describe "prefix redirects" do
    before :each do
      add_redirect_route("/foo", "/bar", :prefix => true)
      add_redirect_route("/foo-temp", "/bar-temp", :prefix => true, :redirect_type => 'temporary')
      reload_routes
    end

    it "should redirect permanently to the destination" do
      response = router_request("/foo")
      expect(response.code).to eq(301)
      expect(response.headers['Location']).to eq("/bar")
    end

    it "should redirect temporarily to the destination when asked to" do
      response = router_request("/foo-temp")
      expect(response.code).to eq(302)
      expect(response.headers['Location']).to eq("/bar-temp")
    end

    it "should preserve extra path sections when redirecting" do
      response = router_request("/foo/baz")
      expect(response.headers['Location']).to eq("/bar/baz")
    end

    it "should preserve the query string when redirecting" do
      response = router_request("/foo?baz=qux")
      expect(response.headers['Location']).to eq("/bar?baz=qux")
    end

    it "should contain cache headers of 24hrs" do
      response = router_request("/foo")
      expect(response.headers['Cache-Control']).to eq("max-age=86400, public")
      expires = Time.parse(response.headers['Expires'])
      expect(expires).to be_within(10).of(Time.now + CACHE_EXPIRES_PERIOD)
    end
  end

  describe "external redirects" do
    before :each do
      add_redirect_route("/foo", "http://foo.example.com/foo")
      add_redirect_route("/bar", "http://bar.example.com/bar", :prefix => true)
      reload_routes
    end

    describe "exact redirect" do
      it "should redirect to the external URL" do
        response = router_request("/foo")
        expect(response.headers['Location']).to eq("http://foo.example.com/foo")
      end

      it "should not preserve the query string" do
        response = router_request("/foo?bar=baz")
        expect(response.headers['Location']).to eq("http://foo.example.com/foo")
      end
    end

    describe "prefix redirect" do

      it "should redirect to the external URL" do
        response = router_request("/bar")
        expect(response.headers['Location']).to eq("http://bar.example.com/bar")
      end

      it "should preserve the path" do
        response = router_request("/bar/baz")
        expect(response.headers['Location']).to eq("http://bar.example.com/bar/baz")
      end

      it "should preserve the query string" do
        response = router_request("/bar?baz=qux")
        expect(response.headers['Location']).to eq("http://bar.example.com/bar?baz=qux")
      end
    end
  end
end
