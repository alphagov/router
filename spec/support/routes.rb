require 'mongo'

module RoutesHelpers
  def add_backend(id, url)
    RoutesHelpers.db["backends"].insert({"backend_id" => id, "backend_url" => url})
  end

  def add_backend_route(path, backend_id, options = {})
    add_route path, options.merge(:handler => "backend", :backend_id => backend_id)
  end

  def add_redirect_route(path, redirect_to, options = {})
    add_route path, options.merge(:handler => "redirect", :redirect_to => redirect_to)
  end

  def add_gone_route(path, options = {})
    add_route path, options.merge(:handler => "gone")
  end

  def add_route(path, attrs = {})
    route_type = attrs.delete(:prefix) ? 'prefix' : 'exact'
    RoutesHelpers.db["routes"].insert(attrs.merge({
      "incoming_path" => path,
      "route_type" => route_type,
    }))
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
