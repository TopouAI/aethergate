# Database Package

Database conventions shared across the repository. Application migrations stay with the service that owns the schema.

The single PostgreSQL instance initially contains two isolated databases and users:

```text
PostgreSQL
├── litellm     owned by litellm_user
└── aethergate  owned by aethergate_user
```

AetherGate must use supported LiteLLM APIs rather than querying or modifying LiteLLM internal tables.

