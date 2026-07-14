# LiteLLM Integration

LiteLLM is AetherGate's model data plane. This integration will manage supported administrative workflows such as models, virtual keys, limits, budgets, and usage synchronization.

Rules:

- Use supported LiteLLM APIs; do not depend on LiteLLM internal database tables.
- Keep the LiteLLM master key server-side and out of browser-delivered configuration.
- Map each LiteLLM identifier to an AetherGate organization, workspace, project, application, or member through AetherGate-owned records.
- Normalize LiteLLM responses and errors at the adapter boundary.
- Test compatibility before changing the pinned LiteLLM image version.

[LiteLLM documentation](https://docs.litellm.ai/)

