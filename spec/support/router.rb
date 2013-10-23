require 'httpclient'

module RouterHelpers

  def api_url(path, api_port = nil)
    api_port ||= 3168
    "http://127.0.0.1:#{api_port}#{path}"
  end

  def reload_routes(api_port = nil)
    HTTPClient.post(api_url("/reload", api_port))
  end

  def router_url(path, port = nil)
    port ||= 3169
    "http://127.0.0.1:#{port}#{path}"
  end

  def router_request(path, options = {})
    HTTPClient.get(router_url(path, options[:port]))
  end

  class << self
    def init
      @running_routers = []
      at_exit do
        @running_routers.dup.each do |pid|
          puts "Stopping router #{pid}"
          RouterHelpers.stop_router(pid)
        end
      end
    end

    def start_router(options = {})
      port = options[:port] || 3169
      api_port = options[:api_port] || 3168
      puts "Starting router on port: #{port}, api_port: #{api_port}" if options[:verbose]

      repo_root = File.expand_path("../../..", __FILE__)

      extra_env = options[:extra_env] || {}
      env = {
        "ROUTER_PUBADDR"  => ":#{port}",
        "ROUTER_APIADDR"  => ":#{api_port}",
        "ROUTER_MONGO_DB" => "router_test",
      }.merge(extra_env)

      if ENV['USE_COMPILED_ROUTER']
        command = %w(./router)
      else
        print `#{repo_root}/build_gopath.sh`
        command = %w(go run main.go router.go)
        env["GOPATH"] = "#{repo_root}/gopath.tmp"
      end

      pid = spawn(env, *command, :chdir => repo_root, :pgroup => true, :out => "/dev/null", :err => "/dev/null")

      retries = 0
      begin
        s = TCPSocket.new("localhost", api_port)
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
      @running_routers << pid
      pid
    end

    def stop_router(router_pid)
      Process.kill("-INT", router_pid)
      Process.wait(router_pid)
      @running_routers.delete(router_pid)
    end

    def included(base)
      base.extend(ExampleGroupMethods)
    end
  end

  module ExampleGroupMethods
    def start_router_around_all(options = {})
      router = nil
      before :all do
        router = RouterHelpers.start_router(options)
      end
      after :all do
        RouterHelpers.stop_router(router)
      end
    end
  end
end

RSpec.configuration.include(RouterHelpers)
RSpec.configuration.before(:suite) do
  RouterHelpers.init
  RouterHelpers.start_router(:verbose => true)
end
