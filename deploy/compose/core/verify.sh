#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")"
source .env

echo "== Docker services =="
docker compose ps

echo
echo "== LiteLLM health =="
curl --fail --silent --show-error \
  "http://127.0.0.1:${LITELLM_HOST_PORT}/health/liveliness"
echo

echo
echo "== LiteLLM database through PgBouncer =="
docker exec \
  -e PGPASSWORD="${LITELLM_DB_PASSWORD}" \
  aethergate-pgbouncer \
  psql -h 127.0.0.1 -p 5432 -U litellm_user -d litellm \
  -c "SELECT current_database(), current_user, now();"

echo
echo "== AetherGate database through PgBouncer =="
docker exec \
  -e PGPASSWORD="${AETHERGATE_DB_PASSWORD}" \
  aethergate-pgbouncer \
  psql -h 127.0.0.1 -p 5432 -U aethergate_user -d aethergate \
  -c "SELECT current_database(), current_user, now();"

echo
echo "All basic checks passed."
