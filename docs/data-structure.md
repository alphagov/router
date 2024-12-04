# Data structure

## Routes

The `routes` uses the following data structure:

```json
{
  "route_type": ["prefix", "exact"],
  "incoming_path": "/url-path/here",
  "handler": ["backend", "redirect", "gone"]
}
```

Incoming paths with special characters must be in their % encoded form in the
database (eg spaces must be stored as `%20`).

The behaviour of an enabled route is determined by `handler`. See below for
extra fields corresponding to `handler` types.

### `backend` handler

The `backend` handler causes the Router to reverse proxy to a named
`backend`. The following extra fields are supported:

```json
{
  "backend_id": "backend-id-corresponding-to-backends-collection"
}
```

### `redirect` handler

The `redirect` handler causes the Router to redirect the given
`incoming_path` to the path stored in `redirect_to`. The following
extra fields are supported:

```json
{
  "redirect_to": "/target-of-redirect"
}
```

### `gone` handler

The `gone` handler causes the Router to return a 410 response.
