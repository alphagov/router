# Data structure

The Router requires two MongoDB collections: `routes` and `backends`.

## Routes

The `routes` collection uses the following data structure:

```json
{
  "_id"           : ObjectId(),
  "route_type"    : ["prefix","exact"],
  "incoming_path" : "/url-path/here",
  "handler"       : ["backend", "redirect", "gone"],
  "disabled"      : false
}
```

Incoming paths with special characters must be in their % encoded form in the
database (eg spaces must be stored as `%20`).

The behaviour of an enabled route is determined by `handler`. See below for
extra fields corresponding to `handler` types.

If a route is disabled, the router will return a 503 for all matching requests.
This is typically used if a service needs to be taken offline for maintenance
etc.

### `backend` handler

The `backend` handler causes the Router to reverse proxy to a named
`backend`. The following extra fields are supported:

```json
{
  "backend_id" : "backend-id-corresponding-to-backends-collection"
}
```

### `redirect` handler

The `redirect` handler causes the Router to redirect the given
`incoming_path` to the path stored in `redirect_to`. The following
extra fields are supported:

```json
{
  "redirect_to"   : "/target-of-redirect",
  "redirect_type" : ["permanent", "temporary"]
}
```

### `gone` handler

The `gone` handler causes the Router to return a 410 response.

## Backends

The `backends` collection uses the following data structure:

```json
{
  "_id"         : ObjectId(),
  "backend_id"  : "arbitrary-slug-or-name",
  "backend_url" : "https://example.com:port/"
}
```
