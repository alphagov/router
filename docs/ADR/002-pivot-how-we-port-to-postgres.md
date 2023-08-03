# ADR 002 - Pivot on how we port to PostgreSQL

## Context

This partially reverses the approach outlined in [ADR 001](001-porting-to-postgresql.md). We were going to piggy back on work by the Publishing Team who are building a proxy app to split requests between old and new router/router-api and content-store. However they've hit several issues and can't get it working.

On recommendation from the Tech Lead of the Platform Engineering team we are going to just use feature flags to manage the roll out.

For feature flagging we will check for the presence of an environment variable. The default behaviour will be to use MongoDB if the variable isn't set or does not have the value "postgres". The variable should be called "ROUTER_USE_DB" and the values "mongo" or "postgres" to be absolutely clear.

If the value is not set or is "mongo" router and router-api will use MongoDB. If the value is set to "postgres" router and router-api will use PostgreDB.

On roll out we can set whichever value we want to test on each environment.

When we are happy with the running of things, we can work to make the switch permanent (for example hardcoding and not reading for the environment variable) and remove old code that uses MongoDB.

## Status

Accepted