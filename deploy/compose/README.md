# Docker Compose Deployments

## Core stack

`core/` contains the source-controlled single-server foundation:

- LiteLLM Proxy;
- PostgreSQL with isolated `litellm` and `aethergate` databases;
- PgBouncer;
- Redis;
- initialization, verification, and backup scripts.

The already downloaded `aethergate-litellm-stack` should be copied into `core/` after removing secrets and runtime data. Follow [the import guide](../../docs/deployment/stack-import.md).

## Analytics stack

`analytics/` is reserved for the later ClickHouse and OpenMeter deployment. Keeping it separate prevents the initial development environment from requiring the complete production data platform.

