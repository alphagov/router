require 'spec_helper'

describe "performance" do
  describe "two healthy backends" do
    start_backend_around_all :port => 3160, :identifier => "backend 1"
    start_backend_around_all :port => 3161, :identifier => "backend 2"

    before :each do
      add_backend("backend-1", "http://localhost:3160/")
      add_backend("backend-2", "http://localhost:3161/")
      add_backend_route("/one", "backend-1")
      add_backend_route("/two", "backend-2")
      reload_routes
    end

    it "should not increase latency by more than twofold" do
      results_direct = vegeta_request_latency(["http://localhost:3160/one", "http://localhost:3161/two"])
      results_router = vegeta_request_latency([router_url("/one"), router_url("/two")])

      expect(results_router["max"]).to  be_within(200).percent_of(results_direct["max"])
      expect(results_router["mean"]).to be_within(200).percent_of(results_direct["mean"])
      expect(results_router["95th"]).to be_within(200).percent_of(results_direct["95th"])
      expect(results_router["99th"]).to be_within(200).percent_of(results_direct["99th"])
    end
  end

  describe "one slow backend" do
    start_backend_around_all :port => 3160, :identifier => "backend 1"
    start_backend_around_all :port => 3161, :identifier => "backend 2"
    start_backend_around_all :port => 3162, :type => :tarpit, "response-delay" => "1s"

    before :each do
      add_backend("backend-1", "http://localhost:3160/")
      add_backend("backend-2", "http://localhost:3161/")
      add_backend("backend-slow", "http://localhost:3162/")
      add_backend_route("/one", "backend-1")
      add_backend_route("/two", "backend-2")
      add_backend_route("/slow", "backend-slow")
      reload_routes
    end

    start_vegeta_load_around_all("/slow")

    it "should not impact other backends" do
      opts = {:duration => "11s"}

      results_direct = vegeta_request_latency(
        ["http://localhost:3160/one", "http://localhost:3161/two"],
        opts
      )
      results_router = vegeta_request_latency(
        [router_url("/one"), router_url("/two")],
        opts
      )

      expect(results_router["max"]).to  be_within(200).percent_of(results_direct["max"])
      expect(results_router["mean"]).to be_within(200).percent_of(results_direct["mean"])
      expect(results_router["95th"]).to be_within(200).percent_of(results_direct["95th"])
      expect(results_router["99th"]).to be_within(200).percent_of(results_direct["99th"])
    end
  end

  describe "one downed backend" do
    start_backend_around_all :port => 3160, :identifier => "backend 1"
    start_backend_around_all :port => 3161, :identifier => "backend 2"

    before :each do
      add_backend("backend-1", "http://localhost:3160/")
      add_backend("backend-2", "http://localhost:3161/")
      add_backend("backend-down", "http://localhost:3162/")
      add_backend_route("/one", "backend-1")
      add_backend_route("/two", "backend-2")
      add_backend_route("/down", "backend-down")
      reload_routes
    end

    start_vegeta_load_around_all("/down")

    it "should not be impacted by a missing backend" do
      opts = {:duration => "11s"}

      results_direct = vegeta_request_latency(
        ["http://localhost:3160/one", "http://localhost:3161/two"],
        opts
      )
      results_router = vegeta_request_latency(
        [router_url("/one"), router_url("/two")],
        opts
      )

      expect(results_router["max"]).to  be_within(200).percent_of(results_direct["max"])
      expect(results_router["mean"]).to be_within(200).percent_of(results_direct["mean"])
      expect(results_router["95th"]).to be_within(200).percent_of(results_direct["95th"])
      expect(results_router["99th"]).to be_within(200).percent_of(results_direct["99th"])
    end
  end
end
