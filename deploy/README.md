# Deployment

Reviewed, version-controlled deployment sources live here. Runtime credentials, generated environment files, database volumes, logs, and backups never belong in Git.

- `compose/core`: single-server foundation stack. Copy the safe source files from the existing `aethergate-litellm-stack` here.
- `compose/analytics`: later ClickHouse and OpenMeter composition.
- `postgres/init`: reusable database initialization source if it is separated from the core stack.
- `pgbouncer`: reusable connection-pool configuration.
- `litellm`: reusable gateway configuration.
- `monitoring`: later dashboards, alerts, and telemetry configuration.

Operational instructions are in [`docs/deployment`](../docs/deployment/).

