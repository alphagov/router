require 'httpclient'

module RouterHelpers

  def api_url(path)
    "http://127.0.0.1:3168#{path}"
  end

  def reload_routes
    HTTPClient.post(api_url("/"))
  end

  def router_url(path)
    "http://127.0.0.1:3169#{path}"
  end

  def router_request(path, options = {})
    HTTPClient.get(router_url(path))
  end

  class << self
    def start_router
      at_exit do
        stop_router
      end

      port = 3169
      puts "Starting router on port #{port}"

      repo_root = File.expand_path("../../..", __FILE__)

      env = {
        "ROUTER_PUBADDR"  => ":#{port}",
        "ROUTER_APIADDR"  => ":3168",
        "ROUTER_MONGO_DB" => "router_test",
      }

      if ENV['USE_COMPILED_ROUTER']
        command = %w(./router)
      else
        puts `#{repo_root}/build_gopath.sh`
        command = %w(go run main.go router.go)
        env["GOPATH"] = "#{repo_root}/gopath.tmp"
      end

      @router_pid = spawn(env, *command, :chdir => repo_root, :pgroup => true, :out => "/dev/null", :err => "/dev/null")

      retries = 0
      begin
        s = TCPSocket.new("localhost", port)
      rescue Errno::ECONNREFUSED
        if retries < 20
          retries += 1
          sleep 0.1
          retry
        else
          raise
        end
      ensure
        s.close if s
      end
    end

    def stop_router
      return unless @router_pid
      Process.kill("-INT", @router_pid)
      Process.wait(@router_pid)
      @router_pid = nil
    end
  end
end

RSpec.configuration.include(RouterHelpers)
RSpec.configuration.before(:suite) do
  RouterHelpers.start_router
end
