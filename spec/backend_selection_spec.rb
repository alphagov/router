require 'spec_helper'

describe "Selecting a backend based on the routing data" do

  describe "simple exact routes" do
    start_backend_around_all :port => 3160, :identifier => "backend 1"
    start_backend_around_all :port => 3161, :identifier => "backend 2"

    before :each do
      add_backend("backend-1", "http://localhost:3160/")
      add_backend("backend-2", "http://localhost:3161/")
      add_route("/foo", "backend-1")
      add_route("/bar", "backend-2")
      add_route("/baz", "backend-1")
      reload_routes
    end

    it "should route a matching request to the corresponding backend" do
      response = router_request("/foo")
      expect(response).to have_response_body("backend 1")

      response = router_request("/bar")
      expect(response).to have_response_body("backend 2")

      response = router_request("/baz")
      expect(response).to have_response_body("backend 1")
    end

    it "should 404 for children of the exact routes" do
      response = router_request("/foo/bar")
      expect(response.code).to eq("404")
    end

    it "should 404 for non-matching requests" do
      response = router_request("/wibble")
      expect(response.code).to eq("404")

      response = router_request("/")
      expect(response.code).to eq("404")

      response = router_request("/foo.json")
      expect(response.code).to eq("404")
    end
  end

  describe "simple prefix routes" do
    start_backend_around_all :port => 3160, :identifier => "backend 1"
    start_backend_around_all :port => 3161, :identifier => "backend 2"

    before :each do
      add_backend("backend-1", "http://localhost:3160/")
      add_backend("backend-2", "http://localhost:3161/")
      add_route("/foo", "backend-1", :prefix => true)
      add_route("/bar", "backend-2", :prefix => true)
      add_route("/baz", "backend-1", :prefix => true)
      reload_routes
    end

    it "should route requests for the prefix to the backend" do
      response = router_request("/foo")
      expect(response).to have_response_body("backend 1")

      response = router_request("/bar")
      expect(response).to have_response_body("backend 2")

      response = router_request("/baz")
      expect(response).to have_response_body("backend 1")
    end

    it "should route requests for children of the prefix to the backend" do
      response = router_request("/foo/bar")
      expect(response).to have_response_body("backend 1")

      response = router_request("/bar/foo.json")
      expect(response).to have_response_body("backend 2")

      response = router_request("/baz/fooey/kablooie")
      expect(response).to have_response_body("backend 1")
    end

    it "should 404 for non-matching requests" do
      response = router_request("/wibble")
      expect(response.code).to eq("404")

      response = router_request("/")
      expect(response.code).to eq("404")

      response = router_request("/foo.json")
      expect(response.code).to eq("404")
    end
  end

  describe "prefix route with children" do
    start_backend_around_all :port => 3160, :identifier => "outer"
    start_backend_around_all :port => 3161, :identifier => "inner"

    before :each do
      add_backend("outer-backend", "http://localhost:3160/")
      add_backend("inner-backend", "http://localhost:3161/")
      add_route("/foo", "outer-backend", :prefix => true)
    end

    describe "with an exact child" do
      before :each do
        add_route("/foo/bar", "inner-backend")
        reload_routes
      end

      it "should route the prefix to the outer backend" do
        response = router_request("/foo")
        expect(response).to have_response_body("outer")
      end

      it "should route the exact child to the inner backend" do
        response = router_request("/foo/bar")
        expect(response).to have_response_body("inner")
      end

      it "should route children of the exact child to the outer backend" do
        response = router_request("/foo/bar/baz")
        expect(response).to have_response_body("outer")
      end
    end

    describe "with a prefix child" do
      before :each do
        add_route("/foo/bar", "inner-backend", :prefix => true)
        reload_routes
      end

      it "should route the outer prefix to the outer backend" do
        response = router_request("/foo")
        expect(response).to have_response_body("outer")
      end

      it "should route the inner prefix to the inner backend" do
        response = router_request("/foo/bar")
        expect(response).to have_response_body("inner")
      end

      it "should route children of the inner prefix to the inner backend" do
        response = router_request("/foo/bar/baz")
        expect(response).to have_response_body("inner")
      end

      it "should route other children of the outer prefix to the outer backend" do
        response = router_request("/foo/baz")
        expect(response).to have_response_body("outer")

        response = router_request("/foo/bar.json")
        expect(response).to have_response_body("outer")
      end
    end

    describe "with an exact child and a deeper prefix child" do
      start_backend_around_all :port => 3162, :identifier => "innerer"

      before :each do
        add_backend("innerer-backend", "http://localhost:3162/")
        add_route("/foo/bar", "inner-backend")
        add_route("/foo/bar/baz", "innerer-backend", :prefix => true)
        reload_routes
      end

      it "should route the outer prefix route to the outer backend" do
        response = router_request("/foo")
        expect(response).to have_response_body("outer")

        response = router_request("/foo/baz")
        expect(response).to have_response_body("outer")

        response = router_request("/foo/bar.json")
        expect(response).to have_response_body("outer")
      end

      it "should route the exact route to the inner backend" do
        response = router_request("/foo/bar")
        expect(response).to have_response_body("inner")
      end

      it "should route other children of the outer prefix route to the outer backend" do
        response = router_request("/foo/bar/wibble")
        expect(response).to have_response_body("outer")

        response = router_request("/foo/bar/baz.json")
        expect(response).to have_response_body("outer")
      end

      it "should route the inner prefix route to the innerer backend" do
        response = router_request("/foo/bar/baz")
        expect(response).to have_response_body("innerer")
      end

      it "should route children of the inner prefix route to the innerer backend" do
        response = router_request("/foo/bar/baz/wibble")
        expect(response).to have_response_body("innerer")
      end
    end
  end

  describe "prefix and exact route at same level" do
    start_backend_around_all :port => 3160, :identifier => "backend 1"
    start_backend_around_all :port => 3161, :identifier => "backend 2"

    before :each do
      add_backend("backend-1", "http://localhost:3160/")
      add_backend("backend-2", "http://localhost:3161/")
      add_route("/foo", "backend-1", :prefix => true)
      add_route("/foo", "backend-2")
      reload_routes
    end

    it "should route the exact route to the exact backend" do
      response = router_request("/foo")
      expect(response).to have_response_body("backend 2")
    end

    it "should route children of the route to the prefix backend" do
      response = router_request("/foo/bar")
      expect(response).to have_response_body("backend 1")
    end
  end

  describe "routes at the root level" do
    start_backend_around_all :port => 3160, :identifier => "root backend"
    start_backend_around_all :port => 3161, :identifier => "other backend"

    before :each do
      add_backend("root", "http://localhost:3160/")
      add_backend("other", "http://localhost:3161/")
      add_route("/foo", "other")
    end

    it "should handle an exact route at the root level" do
      add_route("/", "root")
      reload_routes

      response = router_request("/")
      expect(response).to have_response_body("root backend")

      response = router_request("/foo")
      expect(response).to have_response_body("other backend")

      response = router_request("/bar")
      expect(response.code).to eq("404")
    end

    it "should handle a prefix route at the root level" do
      add_route("/", "root", :prefix => true)
      reload_routes

      response = router_request("/")
      expect(response).to have_response_body("root backend")

      response = router_request("/foo")
      expect(response).to have_response_body("other backend")

      response = router_request("/bar")
      expect(response).to have_response_body("root backend")
    end
  end
end
