DROP DATABASE IF EXISTS router_test;
CREATE DATABASE router_test;
\connect router_test;

CREATE TABLE backends (
  id SERIAL PRIMARY KEY,
  backend_id VARCHAR,
  backend_url VARCHAR,
  created_at TIMESTAMP NOT NULL,
  updated_at TIMESTAMP NOT NULL
);
CREATE UNIQUE INDEX index_backends_on_backend_id ON backends (backend_id);

CREATE TABLE routes (
  id SERIAL PRIMARY KEY,
  incoming_path VARCHAR,
  route_type VARCHAR,
  handler VARCHAR,
  disabled BOOLEAN DEFAULT false,
  backend_id VARCHAR,
  redirect_to VARCHAR,
  redirect_type VARCHAR,
  segments_mode VARCHAR,
  created_at TIMESTAMP NOT NULL,
  updated_at TIMESTAMP NOT NULL
);
CREATE UNIQUE INDEX index_unique_routes ON routes (incoming_path, route_type);

CREATE TABLE users (
  id SERIAL PRIMARY KEY,
  name VARCHAR,
  email VARCHAR,
  uid VARCHAR,
  organisation_slug VARCHAR,
  organisation_content_id VARCHAR,
  app_name VARCHAR,
  permissions TEXT,
  remotely_signed_out BOOLEAN DEFAULT false,
  disabled BOOLEAN DEFAULT false,
  created_at TIMESTAMP NOT NULL,
  updated_at TIMESTAMP NOT NULL
);

CREATE OR REPLACE FUNCTION notify_listeners() RETURNS TRIGGER AS $$
  BEGIN
  PERFORM pg_notify('notify', 'notification from test database');
  RETURN NULL;
  END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER backends_notify_trigger 
AFTER INSERT OR UPDATE OR DELETE 
ON routes 
FOR EACH ROW
EXECUTE PROCEDURE notify_listeners();

CREATE TRIGGER routes_notify_trigger 
AFTER INSERT OR UPDATE OR DELETE 
ON backends 
FOR EACH ROW
EXECUTE PROCEDURE notify_listeners();

CREATE TRIGGER users_notify_trigger 
AFTER INSERT OR UPDATE OR DELETE 
ON users 
FOR EACH ROW
EXECUTE PROCEDURE notify_listeners();