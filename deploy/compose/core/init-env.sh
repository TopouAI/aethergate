#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")"

if [[ -f .env ]]; then
  echo ".env already exists; refusing to overwrite it." >&2
  echo "Move or delete .env only if you intentionally want new credentials." >&2
  exit 1
fi

if ! command -v openssl >/dev/null 2>&1; then
  echo "openssl is required. Install it with:" >&2
  echo "  sudo apt-get update && sudo apt-get install -y openssl" >&2
  exit 1
fi

mkdir -p secrets backups

POSTGRES_PASSWORD="$(openssl rand -hex 32)"
LITELLM_DB_PASSWORD="$(openssl rand -hex 32)"
AETHERGATE_DB_PASSWORD="$(openssl rand -hex 32)"
PGBOUNCER_ADMIN_PASSWORD="$(openssl rand -hex 32)"
REDIS_PASSWORD="$(openssl rand -hex 32)"
LITELLM_MASTER_KEY="sk-$(openssl rand -hex 32)"
LITELLM_SALT_KEY="sk-salt-$(openssl rand -hex 32)"
UI_PASSWORD="$(openssl rand -hex 24)"

cat > .env <<EOF
TZ=Asia/Shanghai

POSTGRES_IMAGE=postgres:17-alpine
POSTGRES_PASSWORD=${POSTGRES_PASSWORD}
POSTGRES_BIND_IP=127.0.0.1
POSTGRES_HOST_PORT=5433
POSTGRES_MAX_CONNECTIONS=300
POSTGRES_SHARED_BUFFERS=512MB
POSTGRES_EFFECTIVE_CACHE_SIZE=2GB
POSTGRES_WORK_MEM=8MB
POSTGRES_MAINTENANCE_WORK_MEM=256MB
POSTGRES_MAX_WAL_SIZE=2GB
POSTGRES_STATEMENT_TIMEOUT=120s
POSTGRES_IDLE_TX_TIMEOUT=60s
POSTGRES_SLOW_QUERY_MS=2000
POSTGRES_SHM_SIZE=512mb

LITELLM_DB_PASSWORD=${LITELLM_DB_PASSWORD}
AETHERGATE_DB_PASSWORD=${AETHERGATE_DB_PASSWORD}

PGBOUNCER_IMAGE=edoburu/pgbouncer:v1.25.2-p0
PGBOUNCER_ADMIN_PASSWORD=${PGBOUNCER_ADMIN_PASSWORD}
PGBOUNCER_BIND_IP=127.0.0.1
PGBOUNCER_HOST_PORT=6432

REDIS_IMAGE=redis:7.4-alpine
REDIS_PASSWORD=${REDIS_PASSWORD}

LITELLM_IMAGE=ghcr.io/berriai/litellm-non_root:v1.92.0
LITELLM_MASTER_KEY=${LITELLM_MASTER_KEY}
LITELLM_SALT_KEY=${LITELLM_SALT_KEY}
LITELLM_BIND_IP=0.0.0.0
LITELLM_HOST_PORT=4000
LITELLM_NUM_WORKERS=1
LITELLM_DB_HOST=postgres
LITELLM_DB_PORT=5432
LITELLM_DB_CONNECTION_LIMIT=20

UI_USERNAME=admin
UI_PASSWORD=${UI_PASSWORD}
EOF

cat > secrets/pgbouncer_users.txt <<EOF
"litellm_user" "${LITELLM_DB_PASSWORD}"
"aethergate_user" "${AETHERGATE_DB_PASSWORD}"
"pgbouncer_admin" "${PGBOUNCER_ADMIN_PASSWORD}"
EOF

cat > aethergate-backend.env <<EOF
# Use these values when the AetherGate API is added to the same Docker network.
DATABASE_URL=postgresql://aethergate_user:${AETHERGATE_DB_PASSWORD}@pgbouncer:5432/aethergate
DIRECT_URL=postgresql://aethergate_user:${AETHERGATE_DB_PASSWORD}@postgres:5432/aethergate
LITELLM_BASE_URL=http://litellm:4000
LITELLM_MASTER_KEY=${LITELLM_MASTER_KEY}
EOF

chmod 600 .env aethergate-backend.env secrets/pgbouncer_users.txt

echo "Generated:"
echo "  .env"
echo "  aethergate-backend.env"
echo "  secrets/pgbouncer_users.txt"
echo
echo "LiteLLM UI username: admin"
echo "LiteLLM UI password: ${UI_PASSWORD}"
echo
echo "Save the UI password securely."
echo "Do not change LITELLM_SALT_KEY after provider credentials are stored."
