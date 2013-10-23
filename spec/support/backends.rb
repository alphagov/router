module BackendHelpers
  class << self
    def init
      @running_backends = []
      at_exit do
        @running_backends.dup.each do |pid|
          puts "Stopping backend #{pid}"
          stop_backend(pid)
        end
      end
    end

    def start_backend(options = {})
      port = options.delete(:port) || 3160
      command = %w(go run)
      command << test_backend_path(options.delete(:type))
      command << "-port=#{port}"
      command += options.map {|k, v| "-#{k}=#{v}" }

      pid = spawn(*command, :pgroup => true, :out => "/dev/null", :err => "/dev/null")

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

      @running_backends << pid
      pid
    end

    def stop_backend(pid)
      Process.kill("-INT", pid)
      Process.wait(pid)
      @running_backends.delete(pid)
    end

    def test_backend_path(type)
      type ||= "simple"
      File.expand_path("../../test_backends/#{type}_backend.go", __FILE__)
    end

    def included(base)
      base.extend(ExampleGroupMethods)
    end
  end

  module ExampleGroupMethods
    def start_backend_around_all(options = {})
      backend = nil
      before :all do
        backend = BackendHelpers.start_backend(options)
      end
      after :all do
        BackendHelpers.stop_backend(backend)
      end
    end
  end
end

RSpec.configuration.include(BackendHelpers)
BackendHelpers.init
