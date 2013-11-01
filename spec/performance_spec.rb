require 'spec_helper'
require 'open3'
require 'pp'

def vegeta(requests)
  rate = 1000
  puts "Rate: #{rate} rps"
  cmd = "./vegeta attack -rate #{rate}"
  Open3.popen2(cmd) {|i,o,t|
    i.puts requests
    i.close
    Open3.popen2("./vegeta report -reporter json") {|i2,o2,t2|
      i2.puts o.read
      i2.close
      return JSON.parse(o2.read)
    }
  }
end

describe "perf" do

  start_backend_around_all :port => 3160, :identifier => "backend 1"
  start_backend_around_all :port => 3161, :identifier => "backend 2"

  before :each do
    add_backend("backend-1", "http://localhost:3160/")
    add_backend("backend-2", "http://localhost:3161/")
    add_backend_route("/one", "backend-1")
    add_backend_route("/two", "backend-2")
    reload_routes
  end

  it "should be fast as LIGHTNING" do
    direct = vegeta <<EOF
GET http://localhost:3160/one
GET http://localhost:3161/two
EOF
    router = vegeta <<EOF
GET #{router_url("/one")}
GET #{router_url("/two")}
EOF

    results = {}
    direct['latencies'].each_key do |key|
      d = direct['latencies'][key].to_f
      r = router['latencies'][key].to_f
      results[key] = {
        :direct => "%.2f us" % [d / 1000],
        :router => "%.2f us" % [r / 1000],
        :difference => "%.2f us" % [(r - d) / 1000],
        :percentage => "%d%%" % [(r / d) * 100],
      }
    end

    pp results
  end
end
