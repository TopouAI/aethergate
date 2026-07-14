# AetherGate LiteLLM Stack

This stack prepares the initial infrastructure for AetherGate:

- PostgreSQL 17
- PgBouncer 1.25
- Redis 7.4
- LiteLLM AI Gateway
- Separate `litellm` and `aethergate` databases
- Secure random credential generation
- Database backup and verification scripts

## Architecture

```text
Public Internet
      |
      | TCP 4000 (development only)
      v
LiteLLM
  |                  |
  | direct DB        | Redis
  v                  v
PostgreSQL         Redis

AetherGate API (added later)
  |
  | runtime queries
  v
PgBouncer
  |
  v
PostgreSQL
```

LiteLLM initially connects directly to PostgreSQL because its Prisma schema
initialization and upgrade process is more reliable without transaction
pooling. PgBouncer is already configured for:

- `litellm`: session pooling, optional after validation
- `aethergate`: transaction pooling for normal application traffic

AetherGate migrations should use the direct PostgreSQL URL.

## 1. Upload the stack

```bash
sudo mkdir -p /opt/aethergate-litellm-stack
sudo chown -R "$USER":"$USER" /opt/aethergate-litellm-stack
cd /opt/aethergate-litellm-stack
```

Copy all files from this package into that directory.

## 2. Generate credentials

```bash
chmod +x init-env.sh backup.sh verify.sh
./init-env.sh
```

This creates:

```text
.env
aethergate-backend.env
secrets/pgbouncer_users.txt
```

All three files contain secrets and must never be committed to Git.

## 3. Validate and start

```bash
docker compose config --quiet
docker compose pull
docker compose up -d
```

Check startup:

```bash
docker compose ps
docker compose logs -f litellm
```

The first LiteLLM startup may take longer while database tables are created.

## 4. Access LiteLLM through the public IP

The default development binding is:

```text
0.0.0.0:4000
```

Open:

```text
http://YOUR_SERVER_PUBLIC_IP:4000/ui
```

Read the UI credentials:

```bash
grep -E 'UI_USERNAME|UI_PASSWORD' .env
```

Cloud firewall/security group rule for development:

```text
Protocol: TCP
Port: 4000
Source: YOUR_CURRENT_PUBLIC_IP/32
```

Do not leave TCP/4000 open to `0.0.0.0/0`. It uses unencrypted HTTP. For
production, place Nginx or another reverse proxy with HTTPS in front of
LiteLLM and change `LITELLM_BIND_IP` back to `127.0.0.1`.

## 5. Database access for local AetherGate development

PostgreSQL and PgBouncer are deliberately not exposed publicly.

Create an SSH tunnel from Windows PowerShell:

```powershell
ssh -L 6432:127.0.0.1:6432 -L 5433:127.0.0.1:5433 root@YOUR_SERVER_IP
```

Keep the terminal open.

Pooled AetherGate runtime connection:

```text
postgresql://aethergate_user:<AETHERGATE_DB_PASSWORD>@127.0.0.1:6432/aethergate
```

Direct AetherGate migration connection:

```text
postgresql://aethergate_user:<AETHERGATE_DB_PASSWORD>@127.0.0.1:5433/aethergate
```

Read the password:

```bash
grep '^AETHERGATE_DB_PASSWORD=' .env
```

Never expose PostgreSQL 5433 or PgBouncer 6432 to the whole Internet. If you
temporarily change `PGBOUNCER_BIND_IP=0.0.0.0`, restrict the cloud security
group to your own public IP and understand that the connection is not using
TLS.

## 6. Add the AetherGate API later

When the AetherGate API container joins `aethergate_network`, use the generated:

```text
aethergate-backend.env
```

It contains:

```text
DATABASE_URL  -> PgBouncer runtime connection
DIRECT_URL    -> direct PostgreSQL migration connection
LITELLM_BASE_URL
LITELLM_MASTER_KEY
```

Do not expose `LITELLM_MASTER_KEY` to the browser. Only the AetherGate backend
may use it.

## 7. Verify the installation

```bash
./verify.sh
```

Manual health check:

```bash
curl http://127.0.0.1:4000/health/liveliness
```

## 8. Inspect PgBouncer

```bash
source .env

docker exec \
  -e PGPASSWORD="$PGBOUNCER_ADMIN_PASSWORD" \
  -it aethergate-pgbouncer \
  psql -h 127.0.0.1 -p 5432 \
  -U pgbouncer_admin -d pgbouncer \
  -c "SHOW POOLS;"
```

## 9. Optional: route LiteLLM through PgBouncer

The safe default is:

```text
LITELLM_DB_HOST=postgres
```

After LiteLLM has initialized successfully and you have tested backups, you
may try:

```bash
sed -i 's/^LITELLM_DB_HOST=postgres$/LITELLM_DB_HOST=pgbouncer/' .env
docker compose up -d --force-recreate litellm
docker compose logs -f litellm
```

The `litellm` pool uses session mode for better Prisma compatibility. Before
every LiteLLM upgrade, make a backup and consider switching the host back to
`postgres` while schema changes are applied.

## 10. Back up both databases

```bash
./backup.sh
```

Backups are written to:

```text
backups/
```

Copy them to another server or object storage. A backup stored only on the
same server is not sufficient disaster recovery.

## 11. Useful commands

```bash
docker compose ps
docker compose logs -f
docker compose restart litellm
docker compose down
docker compose pull
docker compose up -d
```

Do not run this unless you intentionally want to delete all database and Redis
data:

```bash
docker compose down -v
```

## Existing New API stack

This stack uses unique container, network and volume names. Host ports are
configurable:

- LiteLLM: `4000`
- Direct PostgreSQL: `5433`, local only
- PgBouncer: `6432`, local only

Change the values in `.env` if any host port is already occupied.
