require 'mongo'
require 'net/http'

module RoutesHelpers
  def add_backend(id, url)
    RoutesHelpers.db["applications"].insert({"application_id" => id, "backend_url" => url})
  end

  def add_route(path, backend_id, options = {})
    route_type = options[:prefix] ? 'prefix' : 'exact'
    RoutesHelpers.db["routes"].insert({"application_id" => backend_id, "incoming_path" => path, "route_type" => route_type})
  end

  def reload_routes
    Net::HTTP.post_form(URI.parse("http://localhost:3168/"), {})
  end

  def clear_routes
    RoutesHelpers.db["applications"].remove
    RoutesHelpers.db["routes"].remove
  end

  def self.db
    @db ||= Mongo::MongoClient.new("localhost").db("router_test")
  end
end

RSpec.configuration.include(RoutesHelpers)
RSpec.configuration.after(:each) do
  clear_routes
end
