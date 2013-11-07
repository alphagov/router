require 'spec_helper'

describe "performance" do
  ROUTER_LATENCY_THRESHOLD = 20_000_000 # 20 miliseconds in nanoseconds

  shared_examples "a performant router" do |vegeta_opts|
    vegeta_opts ||= {}

    it "should not significantly increase latency" do
      results_direct = results_router = nil
      t1 = Thread.new { results_direct = vegeta_request_stats(["http://localhost:3160/one", "http://localhost:3161/two"], vegeta_opts) }
      t2 = Thread.new { results_router = vegeta_request_stats([router_url("/one"), router_url("/two")], vegeta_opts) }
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

  context "two healthy backends" do
    start_backend_around_all :port => 3160, :identifier => "backend 1"
    start_backend_around_all :port => 3161, :identifier => "backend 2"

    before :each do
      add_backend("backend-1", "http://localhost:3160/")
      add_backend("backend-2", "http://localhost:3161/")
      add_backend_route("/one", "backend-1")
      add_backend_route("/two", "backend-2")
      reload_routes
    end

    it_behaves_like "a performant router"

    context "when the routes are being reloaded repeatedly" do
      before :each do
        @stop_reloading = false
        @reload_thread = Thread.new do
          until @stop_reloading
            reload_routes
            sleep 0.1
          end
        end
      end

      after :each do
        @stop_reloading = true
        @reload_thread.join
      end

      it_behaves_like "a performant router"
    end

    describe "one slow backend hit separately" do
      start_backend_around_all :port => 3162, :type => :tarpit, "response-delay" => "1s"

      before :each do
        add_backend("backend-slow", "http://localhost:3162/")
        add_backend_route("/slow", "backend-slow")
        reload_routes
      end

      start_vegeta_load_around_all("/slow")

      it_behaves_like "a performant router"
    end

    describe "one downed backend hit seperately" do
      before :each do
        add_backend("backend-down", "http://localhost:3162/")
        add_backend_route("/down", "backend-down")
        reload_routes
      end

      start_vegeta_load_around_all("/down")

      it_behaves_like "a performant router"
    end

    describe "high request throughput", :ulimits => true do
      it_behaves_like "a performant router", :rate => 3000
    end
  end

  context "many concurrent (slow) connections", :ulimits => true do
    start_backend_around_all :port => 3160, :type => :tarpit, "response-delay" => "1s"
    start_backend_around_all :port => 3161, :type => :tarpit, "response-delay" => "1s"

    before :each do
      add_backend("backend-1", "http://localhost:3160/")
      add_backend("backend-2", "http://localhost:3161/")
      add_backend_route("/one", "backend-1")
      add_backend_route("/two", "backend-1")
      reload_routes
    end

    it_behaves_like "a performant router", :rate => 1000
  end
end
