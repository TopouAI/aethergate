# AetherGate Product Overview

AetherGate is an open-source enterprise AI gateway and usage intelligence platform initiated and maintained by TopoAI.

## Product definition

AetherGate is not only an API relay or prepaid token storefront. It is an enterprise control plane that answers:

- Which company, department, project, application, or engineer is using AI?
- Which models and providers may each scope access?
- How much was used, what did it cost, and how did usage change?
- Are budgets, limits, security policies, and contracts being followed?
- Which projects actively adopt AI, and which need support?

## Primary users

- Platform administrators operating models and upstream providers
- Enterprise administrators managing organizations, permissions, and budgets
- Engineering leaders reviewing adoption, reliability, and cost
- Finance and operations teams reconciling usage and contracts
- Developers creating keys, inspecting requests, and testing models

## Core resource model

```text
Organization
├── Workspace
│   ├── Department
│   ├── Project
│   │   ├── Application
│   │   └── API Key
│   └── Members and roles
├── Model access policies
├── Budgets and quotas
└── Usage, costs, and audit records
```

An exact hierarchy will be validated during domain modeling. External gateway identifiers remain mappings rather than becoming the primary AetherGate identity model.

## Product boundary

LiteLLM provides gateway execution. AetherGate provides enterprise workflows and governance around it. Helicone is a product and interaction reference, not a backend dependency or long-term fork. ClickHouse and OpenMeter are introduced only when their scale and billing responsibilities are required.

## Initial success criteria

The foundation phase is successful when an operator can:

1. create an organization, workspace, project, application, and member;
2. assign allowed models, limits, and a budget;
3. issue and revoke a usable LiteLLM-backed API key;
4. attribute usage to the correct enterprise scopes;
5. view reliable basic usage and cost summaries;
6. deploy the complete foundation on a documented single server without exposing its databases publicly.

