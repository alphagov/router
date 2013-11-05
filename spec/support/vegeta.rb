require 'json'
require 'open3'

module VegetaHelpers

  def vegeta_request_stats(urls, options = {})
    urls = urls.map { |u| "GET #{u}" }.join("\n")

    attack_cmd = %w(vegeta attack)
    attack_cmd += options.map {|k, v| "-#{k}=#{v}" }

    report_cmd = %w(vegeta report --reporter json)

    json = nil
    Open3.popen3(*attack_cmd) {|attack_in, attack_out, attack_err|
      attack_in.puts urls
      attack_in.close
      Open3.popen2(*report_cmd) {|report_in, report_out|
        report_in.puts attack_out.read
        report_in.close
        json = JSON.parse(report_out.read)
      }
    }

    return json
  end

  class << self
    def init
      @background_procs = []
      at_exit do
        @background_procs.dup.each do |pid|
          puts "Stopping background vegeta #{pid}"
          stop_vegeta(pid)
        end
      end
    end

    def start_vegeta_bg(urls, options = {})
      urls = urls.map { |u| "GET #{u}" }.join("\n")

      cmd = %w(vegeta attack)
      cmd += options.map { |k, v| "-#{k}=#{v}" }

      r, w = IO.pipe
      w.write urls
      w.close
      pid = spawn(*cmd, :pgroup => true, :in => r, :out => "/dev/null", :err => "/dev/null")

      raise "vegeta background failed" unless pid
      @background_procs << pid

      pid
    end

    def stop_vegeta(pid)
      if Process.waitpid(pid, Process::WNOHANG).nil?
        Process.kill("-INT", pid)
        Process.wait(pid)
      end
      @background_procs.delete(pid)
    end

    def included(base)
      base.extend(ExampleGroupMethods)
    end
  end

  module ExampleGroupMethods
    def start_vegeta_load_around_all(path)
      pid = nil
      before :all do
        urls = [router_url(path)]
        opts = {:duration => "10m"}
        pid = VegetaHelpers.start_vegeta_bg(urls, opts)
      end
      after :all do
        VegetaHelpers.stop_vegeta(pid)
      end
    end
  end
end

RSpec.configuration.include(VegetaHelpers)
VegetaHelpers.init
