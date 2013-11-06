require 'spec_helper'

describe "performance" do
  ROUTER_LATENCY_THRESHOLD = 20_000_000 # 20 miliseconds in nanoseconds

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
      puts "\ntwo healthy backends latencies"
      pp res

      expect(res[:router]["max"]).to  be_within(ROUTER_LATENCY_THRESHOLD).of(res[:direct]["max"])
      expect(res[:router]["mean"]).to be_within(ROUTER_LATENCY_THRESHOLD).of(res[:direct]["mean"])
      expect(res[:router]["95th"]).to be_within(ROUTER_LATENCY_THRESHOLD).of(res[:direct]["95th"])
      expect(res[:router]["99th"]).to be_within(ROUTER_LATENCY_THRESHOLD).of(res[:direct]["99th"])
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

      results_direct = results_router = nil
      t1 = Thread.new do
        results_direct = vegeta_request_stats(
          ["http://localhost:3160/one", "http://localhost:3161/two"],
          opts
        )
      end
      t2 = Thread.new do
        results_router = vegeta_request_stats(
          [router_url("/one"), router_url("/two")],
          opts
        )
      end
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
      puts "\none slow backend latencies"
      pp res

      expect(res[:router]["max"]).to  be_within(ROUTER_LATENCY_THRESHOLD).of(res[:direct]["max"])
      expect(res[:router]["mean"]).to be_within(ROUTER_LATENCY_THRESHOLD).of(res[:direct]["mean"])
      expect(res[:router]["95th"]).to be_within(ROUTER_LATENCY_THRESHOLD).of(res[:direct]["95th"])
      expect(res[:router]["99th"]).to be_within(ROUTER_LATENCY_THRESHOLD).of(res[:direct]["99th"])
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

      results_direct = results_router = nil
      t1 = Thread.new do
        results_direct = vegeta_request_stats(
          ["http://localhost:3160/one", "http://localhost:3161/two"],
          opts
        )
      end
      t2 = Thread.new do
        results_router = vegeta_request_stats(
          [router_url("/one"), router_url("/two")],
          opts
        )
      end
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
      puts "\none downed backend latencies"
      pp res

      expect(res[:router]["max"]).to  be_within(ROUTER_LATENCY_THRESHOLD).of(res[:direct]["max"])
      expect(res[:router]["mean"]).to be_within(ROUTER_LATENCY_THRESHOLD).of(res[:direct]["mean"])
      expect(res[:router]["95th"]).to be_within(ROUTER_LATENCY_THRESHOLD).of(res[:direct]["95th"])
      expect(res[:router]["99th"]).to be_within(ROUTER_LATENCY_THRESHOLD).of(res[:direct]["99th"])
    end
  end
end
