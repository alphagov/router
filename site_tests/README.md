# Site-wide Routing Tests

These Playwright tests are for ensuring Router is routing traffic correctly.
They test all of Router's route types and hit various different backends and redirects.

## Usage

1. Install Playwright: `npm install`
2. Install Playwright browsers: `npx playwright install`
3. Run tests: `npm test`

## Configuration

Tests can be configured via environment variables

* `BASE_URL` - Site hostname (default `https://www.gov.uk`)
* `BASIC_AUTH` - Basic auth credentials in `username:password` format
* `CACHE_BUST` - Appends random cache busting string to requests (default `true`)

## Examples

Run tests with basic auth against a different base URL:

```sh
BASIC_AUTH=myusername:mypassword BASE_URL=https://www.integration.publishing.service.gov.uk npm test
```

Run tests with cache bust disabled:

```sh
CACHE_BUST=false npm test
```
