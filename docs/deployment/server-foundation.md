# Server Foundation Deployment

This guide consolidates the foundation deployment decisions for AetherGate development. It assumes the actual stack files have been imported into `deploy/compose/core`.

## Foundation services

```text
Public or trusted network
          │
          ▼
LiteLLM Proxy :4000
├── PostgreSQL / litellm
└── Redis

AetherGate API
          │
          ▼
PgBouncer
          │
          ▼
PostgreSQL / aethergate
```

The foundation stack contains:

- LiteLLM Proxy for model routing, virtual keys, gateway limits, and upstream failover;
- PostgreSQL for LiteLLM and AetherGate transactional data;
- PgBouncer for AetherGate runtime connection pooling;
- Redis for LiteLLM and later bounded AetherGate coordination needs.

ClickHouse and OpenMeter are intentionally deferred until the analytics and billing phases.

## Database ownership

One PostgreSQL instance initially hosts two isolated databases:

```text
litellm
└── owner: litellm_user

aethergate
└── owner: aethergate_user
```

LiteLLM owns its schema. AetherGate must never write LiteLLM internal tables. The Go API communicates with LiteLLM through supported APIs and keeps external identifier mappings in the `aethergate` database.

## Connection policy

The generated backend environment is expected to contain values with this meaning:

```env
DATABASE_URL=postgresql://aethergate_user:<password>@pgbouncer:5432/aethergate
DIRECT_URL=postgresql://aethergate_user:<password>@postgres:5432/aethergate
LITELLM_BASE_URL=http://litellm:4000
LITELLM_MASTER_KEY=<server-side-master-key>
```

- `DATABASE_URL`: normal AetherGate runtime traffic through PgBouncer transaction pooling.
- `DIRECT_URL`: schema migrations and operations requiring a direct PostgreSQL session.
- `LITELLM_BASE_URL`: internal Compose-network address.
- `LITELLM_MASTER_KEY`: privileged server-side integration credential; never expose it to the Console browser.

LiteLLM should initially connect directly to PostgreSQL because startup and schema operations may need session behavior. Evaluate a PgBouncer session pool only after testing the exact pinned LiteLLM version. Do not route migrations through a transaction pool.

## Network exposure

The earlier development stack used these host bindings:

| Service | Development binding | Rule |
|---|---|---|
| LiteLLM | `0.0.0.0:4000` | Temporary public development access; restrict source IPs |
| PostgreSQL | `127.0.0.1:5433` | Host-local access only |
| PgBouncer | `127.0.0.1:6432` | Host-local access only |
| Redis | no host port | Compose network only |

Inside the Compose network, services use container addresses such as `postgres:5432`, `pgbouncer:5432`, `redis:6379`, and `litellm:4000`. Host port numbers are not used for service-to-service traffic.

For temporary remote development, allow TCP 4000 only from the developers' current public IP addresses in the cloud security group and host firewall. Do not leave `0.0.0.0/0` enabled.

Before production:

- place LiteLLM and the AetherGate API behind a reverse proxy;
- use trusted TLS certificates and HTTPS;
- stop publishing the application containers directly where practical;
- add authentication, rate limits, request-size limits, and appropriate timeouts at the edge;
- keep PostgreSQL, PgBouncer, and Redis private.

## Server preparation

Recommended runtime location:

```bash
sudo install -d -m 0750 /opt/aethergate
sudo chown -R "$USER":"$USER" /opt/aethergate
```

Required tools:

```bash
docker --version
docker compose version
openssl version
```

The stack scripts may also require `bash`, `curl`, and standard PostgreSQL/Docker utilities available inside containers. Review each script before running it with elevated privileges.

## First deployment

Copy the reviewed contents of `deploy/compose/core` into `/opt/aethergate`. Keep server-only `.env`, generated backend environment, secrets, and backups in `/opt/aethergate`, not in Git.

From the runtime directory:

```bash
cd /opt/aethergate
chmod +x init-env.sh backup.sh verify.sh
./init-env.sh
```

The initialization script should create strong random values for:

- PostgreSQL administrative and application users;
- LiteLLM master and salt keys;
- LiteLLM UI credentials;
- the generated AetherGate backend environment.

The LiteLLM salt key protects encrypted provider credentials. Do not casually regenerate or change it after credentials are stored.

Validate the resolved Compose configuration before pulling or starting services:

```bash
docker compose config --quiet
docker compose pull
docker compose up -d
```

Check status and logs:

```bash
docker compose ps
docker compose logs --tail=200 litellm
docker compose logs --tail=200 postgres
docker compose logs --tail=200 pgbouncer
docker compose logs --tail=200 redis
```

Use the service names from the imported `compose.yaml` if they differ.

## Verification

Run the stack verification script:

```bash
./verify.sh
```

Also verify these outcomes:

1. Every service is running and health checks pass.
2. LiteLLM readiness responds on the configured endpoint.
3. The LiteLLM UI is reachable only from an allowed source.
4. Both `litellm` and `aethergate` databases exist with separate owners.
5. PgBouncer accepts the AetherGate runtime connection.
6. PostgreSQL and PgBouncer are not reachable from an untrusted external host.
7. A test LiteLLM virtual key can list or call only its permitted models.

The earlier stack used:

```text
http://<server-public-ip>:4000/ui
http://<server-public-ip>:4000/health/readiness
```

Confirm endpoints against the imported LiteLLM version and `verify.sh`; do not treat a working UI alone as a complete health check.

## Connecting the Go API

When AetherGate API joins the same Compose network, use the internal values generated in `aethergate-backend.env`. Load them into the API container through Compose `env_file` or secret handling; do not copy them into frontend configuration.

When the Go API runs on a developer workstation, prefer one of these patterns:

1. run the API in the server Compose network;
2. use an SSH tunnel to the host-local PgBouncer port;
3. use a separate development database reachable only through a VPN or trusted network.

Do not publish PostgreSQL to the entire internet for convenience.

## Backup and restore readiness

Before changes to images, schema, or configuration:

```bash
cd /opt/aethergate
./backup.sh
```

After the command:

- confirm a new backup exists;
- record which databases and configuration files it contains;
- ensure the backup is encrypted or stored in a protected location;
- copy important backups off the server according to retention policy;
- periodically restore into an isolated environment.

A backup without a tested restore procedure is not sufficient. Document the exact restore commands after reviewing the imported `backup.sh` output format.

## Updating the stack

1. Review and commit changes under `deploy/compose/core`.
2. Read release notes for every changed image.
3. Back up the current server deployment.
4. Sync reviewed source while preserving server-only `.env`, secrets, backups, and persistent volumes.
5. Run `docker compose config --quiet`.
6. Pull images and recreate only the intended services.
7. Run `./verify.sh` and inspect logs.
8. Keep the previous configuration and image tags available for rollback.

Use pinned image versions or explicit version variables. Do not introduce a moving `latest` tag into a working stack without an intentional upgrade policy.

## Troubleshooting order

1. `docker compose config` — missing variables, files, or invalid YAML.
2. `docker compose ps` — container state and health.
3. service logs — startup, authentication, migration, or connection errors.
4. `docker compose exec` network checks — internal DNS and ports.
5. host listeners and firewall — expected bindings only.
6. cloud security groups — TCP 4000 restricted to trusted IPs.
7. database roles and URLs — correct database, user, host, port, and password encoding.

Do not weaken database exposure or disable authentication as a troubleshooting shortcut.

