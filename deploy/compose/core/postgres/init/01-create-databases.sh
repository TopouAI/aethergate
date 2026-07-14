#!/usr/bin/env bash
set -euo pipefail

required_vars=(
  LITELLM_DB_USER
  LITELLM_DB_PASSWORD
  LITELLM_DB_NAME
  AETHERGATE_DB_USER
  AETHERGATE_DB_PASSWORD
  AETHERGATE_DB_NAME
)

for variable in "${required_vars[@]}"; do
  if [[ -z "${!variable:-}" ]]; then
    echo "Missing required environment variable: ${variable}" >&2
    exit 1
  fi
done

echo "Creating LiteLLM and AetherGate database roles and databases..."

psql \
  --username "${POSTGRES_USER}" \
  --dbname postgres \
  --set=ON_ERROR_STOP=1 \
  --set=litellm_user="${LITELLM_DB_USER}" \
  --set=litellm_password="${LITELLM_DB_PASSWORD}" \
  --set=litellm_db="${LITELLM_DB_NAME}" \
  --set=aethergate_user="${AETHERGATE_DB_USER}" \
  --set=aethergate_password="${AETHERGATE_DB_PASSWORD}" \
  --set=aethergate_db="${AETHERGATE_DB_NAME}" <<'EOSQL'
SELECT format('CREATE ROLE %I LOGIN PASSWORD %L', :'litellm_user', :'litellm_password')
WHERE NOT EXISTS (
  SELECT 1 FROM pg_roles WHERE rolname = :'litellm_user'
)\gexec

SELECT format('ALTER ROLE %I WITH LOGIN PASSWORD %L', :'litellm_user', :'litellm_password')
\gexec

SELECT format('CREATE ROLE %I LOGIN PASSWORD %L', :'aethergate_user', :'aethergate_password')
WHERE NOT EXISTS (
  SELECT 1 FROM pg_roles WHERE rolname = :'aethergate_user'
)\gexec

SELECT format('ALTER ROLE %I WITH LOGIN PASSWORD %L', :'aethergate_user', :'aethergate_password')
\gexec

SELECT format('CREATE DATABASE %I OWNER %I', :'litellm_db', :'litellm_user')
WHERE NOT EXISTS (
  SELECT 1 FROM pg_database WHERE datname = :'litellm_db'
)\gexec

SELECT format('ALTER DATABASE %I OWNER TO %I', :'litellm_db', :'litellm_user')
\gexec

SELECT format('CREATE DATABASE %I OWNER %I', :'aethergate_db', :'aethergate_user')
WHERE NOT EXISTS (
  SELECT 1 FROM pg_database WHERE datname = :'aethergate_db'
)\gexec

SELECT format('ALTER DATABASE %I OWNER TO %I', :'aethergate_db', :'aethergate_user')
\gexec
EOSQL

psql --username "${POSTGRES_USER}" --dbname "${LITELLM_DB_NAME}" \
  --set=ON_ERROR_STOP=1 \
  -c "CREATE EXTENSION IF NOT EXISTS pgcrypto;"

psql --username "${POSTGRES_USER}" --dbname "${AETHERGATE_DB_NAME}" \
  --set=ON_ERROR_STOP=1 \
  -c "CREATE EXTENSION IF NOT EXISTS pgcrypto;"

echo "Database initialization completed."
