# Changelog

## Unreleased

- Added native OpenAPI 3.x YAML/JSON loading with local and opt-in external `$ref` resolution.
- Added deterministic schema-derived TypeScript types, typed operation parameters, request bodies, success responses, and common typed errors.
- Added `sdkgen.yaml` configuration loading and a claims-shaped OpenAPI 3.1 fixture/test.
- Legacy Swagger/OpenAPI 2 generation remains supported.

Consumers should pin the commit containing this change (or a later tagged release) and replace the placeholder `generatorVersion` value in their configuration with that commit after it is created.
