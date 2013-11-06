require 'spec_helper'

describe "performance" do
  ROUTER_LATENCY_THRESHOLD = 20_000_000 # 20 miliseconds in nanoseconds

  shared_examples "a performant router" do
    it "should not significantly increase latency" do
      results_direct = results_router = nil
      t1 = Thread.new { results_direct = vegeta_request_stats(["http://localhost:3160/one", "http://localhost:3161/two"]) }
      t2 = Thread.new { results_router = vegeta_request_stats([router_url("/one"), router_url("/two")]) }
      t1.join
      t2.join

      res = {
        :router => results_router["status_codes"]["200"] || 0,
        :direct => results_direct["status_codes"]["200"] || 0,
      }
      expect(results_router["requests"]).to eq(results_direct["requests"])
      expect(res[:router]).to eq(results_router["requests"])
      expect(res[:direct]).to eq(results_direct["requests"])

      res = {
        :direct => results_direct["latencies"],
        :router => results_router["latencies"],
      }
      puts "\n#{example.metadata[:example_group][:example_group][:description_args].first} latencies"
      pp res

      expect(res[:router]["mean"]).to be_within(ROUTER_LATENCY_THRESHOLD).of(res[:direct]["mean"])
      expect(res[:router]["95th"]).to be_within(ROUTER_LATENCY_THRESHOLD).of(res[:direct]["95th"])
      expect(res[:router]["99th"]).to be_within(ROUTER_LATENCY_THRESHOLD * 2).of(res[:direct]["99th"])
      expect(res[:router]["max"]).to  be_within(ROUTER_LATENCY_THRESHOLD * 2).of(res[:direct]["max"])
    end
  end

  start_backend_around_all :port => 3160, :identifier => "backend 1"
  start_backend_around_all :port => 3161, :identifier => "backend 2"

  before :each do
    add_backend("backend-1", "http://localhost:3160/")
    add_backend("backend-2", "http://localhost:3161/")
    add_backend_route("/one", "backend-1")
    add_backend_route("/two", "backend-2")
  end

  describe "two healthy backends" do
    before :each do
      reload_routes
    end

    it_behaves_like "a performant router"
  end

  describe "one slow backend" do
    start_backend_around_all :port => 3162, :type => :tarpit, "response-delay" => "1s"

    before :each do
      add_backend("backend-slow", "http://localhost:3162/")
      add_backend_route("/slow", "backend-slow")
      reload_routes
    end

    start_vegeta_load_around_all("/slow")

    it_behaves_like "a performant router"
  end

  describe "one downed backend" do
    before :each do
      add_backend("backend-down", "http://localhost:3162/")
      add_backend_route("/down", "backend-down")
      reload_routes
    end

    start_vegeta_load_around_all("/down")

    it_behaves_like "a performant router"
  end
end
