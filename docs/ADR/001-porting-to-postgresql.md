# ADR 001 - How we migrate Router/Router-API from Mongo to Postgres

## Context

We need to migrate router (and [router-api](https://github.com/alphagov/router-api)) off of MongoDB as its datastore. The majority of GOV.UK platform is on PosgreSQL and there is no longer a strong enough case for being on MongoDB for router data.

Even the [content-store is migrating from MongoDB to PostgreSQL](https://github.com/alphagov/govuk-rfcs/pull/158).

This ADR is to outline our decisions and rationale on the steps we take for that migration:

## Decision
* Spin up a completely separate stack of router and router-api (govuk-docker first) that will talk to a PostgreSQL db only (as forks)
* Provision in EKS a new PostgreSQL db
* Provision the new p-router/p-router-api apps
* Amend messages to router-api to be duplicated to p-router-api within content-store

### Benefits of doing it this way
* As updates are being made to the two databases, we can backfill data previous to that point in time without loss of service or downtime to the live (old) service
* We can test traffic going to the new p-router stack insolation
* Switchover would mean a change in DNS
* We can rename p-router once we decommission old router

### Risks

* We never rename p-router
* DNS messes up and caching gets in the way, causing downtime to requests trying to reach origin

## Alternatives

* We update router and router-api and deploy the changes to the existing version

### Risks

* Big bang
* Running 2 databases under 1 app
* Double code/logic to handle that
* Harder to clean up

## Status

Accepted