require "spec_helper"

describe "loading routes from the db" do
  start_backend_around_all :port => 3160, :identifier => "backend 1"
  start_backend_around_all :port => 3161, :identifier => "backend 2"

  before :each do
    add_backend("backend-1", "http://localhost:3160/")
    add_backend("backend-2", "http://localhost:3161/")
  end

  context "a route with a non-existent backend" do
    before :each do
      add_route("/foo", "backend-1")
      add_route("/bar", "backend-bar")
      add_route("/baz", "backend-2")
      add_route("/qux", "backend-1")
      reload_routes
    end

    it "should skip the invalid route" do
      response = router_request("/bar")
      expect(response.code).to eq(404)
    end

    it "should continue to load other routes" do
      response = router_request("/foo")
      expect(response).to have_response_body("backend 1")

      response = router_request("/baz")
      expect(response).to have_response_body("backend 2")

      response = router_request("/qux")
      expect(response).to have_response_body("backend 1")
    end
  end
end
