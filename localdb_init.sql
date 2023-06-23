CREATE DATABASE router;
\c router;

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
CREATE UNIQUE INDEX index_routes_on_incoming_path ON routes (incoming_path);

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
