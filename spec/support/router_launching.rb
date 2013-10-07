module RouterLaunchingHelpers
  class << self
    def init
      at_exit do
        stop_router
      end
    end

    def start_router
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

RouterLaunchingHelpers.init
RSpec.configuration.before(:suite) do
  RouterLaunchingHelpers.start_router
end
