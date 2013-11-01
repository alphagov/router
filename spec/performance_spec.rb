require 'spec_helper'
require 'open3'

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

    puts "95th percentile results."
    direct_95th = direct['latencies']['95th'].to_f
    router_95th = router['latencies']['95th'].to_f

    puts "direct: %.2f us" % [direct_95th / 1000]
    puts "router: %.2f us" % [router_95th / 1000]

    diff = router_95th - direct_95th
    perc = (router_95th / direct_95th) * 100
    puts "difference: %.2f us (%d%%)" % [diff / 1000, perc]
  end
end
