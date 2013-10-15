require 'mongo'

module RoutesHelpers
  def add_backend(id, url)
    RoutesHelpers.db["backends"].insert({"backend_id" => id, "backend_url" => url})
  end

  def add_route(path, backend_id, options = {})
    route_type = options[:prefix] ? 'prefix' : 'exact'
    RoutesHelpers.db["routes"].insert({"backend_id" => backend_id, "incoming_path" => path, "route_type" => route_type})
  end

  def add_redirect(path, options = {})
    RoutesHelpers.db["routes"].insert ({"incoming_path" => path}).merge(options)
  end

  def clear_routes
    RoutesHelpers.db["backends"].remove
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
