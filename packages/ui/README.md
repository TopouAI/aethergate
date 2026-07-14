# UI Package

Shared AetherGate interface elements built on HeroUI v3 and Tailwind CSS v4.

Own design tokens, navigation shells, charts, filters, empty states, permission-aware controls, and the vendor-neutral `DataGrid` interface here. Feature pages remain in `apps/console`.

Do not re-export every HeroUI component. Wrap only components where AetherGate needs stable behavior, branding, analytics, authorization context, or a replaceable integration boundary.

