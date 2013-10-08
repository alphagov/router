require 'net/http'

module RouterHelpers

  def reload_routes
    Net::HTTP.post_form(URI.parse("http://localhost:3168/"), {})
  end

  def router_request(path, options = {})
    Net::HTTP.get_response(URI.parse("http://localhost:3169#{path}"))
  end

  class << self
    def start_router
      at_exit do
        stop_router
      end

      port = 3169
      puts "Starting router on port #{port}"

      repo_root = File.expand_path("../../..", __FILE__)

      if ENV['USE_COMPILED_ROUTER']
        command = %w(./router)
        env = {}
      else
        puts `#{repo_root}/build_gopath.sh`
        command = %w(go run main.go router.go)
        env = {"GOPATH" => "#{repo_root}/gopath.tmp"}
      end
      command += ["-pubAddr=:#{port}", "-apiAddr=:3168", "-mongoDbName=router_test"]

      @router_pid = spawn(env, *command, :chdir => repo_root, :pgroup => true, :out => "/dev/null", :err => "/dev/null")

      retries = 0
      begin
        s = TCPSocket.new("localhost", port)
      rescue Errno::ECONNREFUSED
        if retries < 10
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
