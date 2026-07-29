# Theme API v1

Theme API v1 is the presentation boundary between thisM and complete theme plugins. A plugin supplies every required primitive, application shell, registered slot, approved portal, light/dark semantic token set, and declarative setting. Partial registries are invalid and are never filled from `default-shadcn`.

The host always owns the React root, routing, authentication and authorization, localization, API transport, data loading, mutations, commands, and error semantics. Plugins must not create a second root or provide business behavior.

## Public validation

- `validateThemeRegistry(registry)` checks Theme API compatibility and all required presentation exports.
- `validateThemeBuild(report)` checks the build graph, entry source, and CSS boundary report before packaging.
- `runThemeConformanceChecks(adapter)` from the testing entry registers the shared rendered-behavior suite for plugin projects.

The only allowed runtime externals are `react`, `react-dom`, `react/jsx-runtime`, and `@thism/theme-sdk`. All other dependencies must be bundled. Theme CSS must use the `thism-theme` cascade layer, stay beneath `[data-thism-theme-root]` or an approved `[data-thism-theme-portal]`, define visible focus, support reduced motion, and include responsive navigation rules.

A compatible API version has major version `1`. Minor and patch releases within v1 are accepted; malformed versions and other major versions are rejected before plugin execution.
