# PgBouncer

Reusable PgBouncer configuration and operational notes.

Initial policy:

- AetherGate runtime traffic uses transaction pooling.
- AetherGate migrations use a direct PostgreSQL connection.
- LiteLLM initially connects directly to PostgreSQL for startup and schema operations.
- A LiteLLM session pool may be evaluated only after compatibility testing.

