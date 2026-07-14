# Security Policy

AetherGate handles API credentials, enterprise identities, usage records, and billing-related data. Treat security issues as confidential until a fix and disclosure plan are available.

## Reporting a vulnerability

Do not open a public issue containing exploit details, credentials, customer information, or proof-of-concept data.

Use GitHub private vulnerability reporting when it is enabled for this repository. Otherwise contact the maintainers privately through the organization contact shown on the repository profile. Include:

- affected component and version or commit;
- impact and required access level;
- reproducible steps or a minimal proof of concept;
- suggested mitigation, if known.

## Deployment baseline

- Never commit `.env`, generated backend environment files, secrets, backups, logs, or database volumes.
- Keep PostgreSQL, PgBouncer, and Redis off the public network.
- Restrict any temporary public LiteLLM port to trusted source IPs.
- Use TLS and a reverse proxy before production exposure.
- Rotate credentials after accidental disclosure and invalidate affected API keys.

