#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")"
source .env

timestamp="$(date +%Y%m%d-%H%M%S)"
mkdir -p backups

docker exec \
  -e PGPASSWORD="${POSTGRES_PASSWORD}" \
  aethergate-postgres \
  pg_dump -U postgres -d litellm -Fc \
  > "backups/litellm-${timestamp}.dump"

docker exec \
  -e PGPASSWORD="${POSTGRES_PASSWORD}" \
  aethergate-postgres \
  pg_dump -U postgres -d aethergate -Fc \
  > "backups/aethergate-${timestamp}.dump"

echo "Created:"
echo "  backups/litellm-${timestamp}.dump"
echo "  backups/aethergate-${timestamp}.dump"
