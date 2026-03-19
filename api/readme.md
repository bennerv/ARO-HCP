# RedHatOpenShift HCP Clusters — Server-Side Models

This document describes how server-side Go models are generated from the
TypeSpec API definitions.

## Generation

Go models (constants, model structs, and serialization helpers) are generated
directly from TypeSpec using the `@azure-tools/typespec-go` emitter. The
generated code lives in `internal/api/{VERSION}/generated/` with
`package generated`.

To regenerate models for the default (latest) API version:

```bash
cd api
make models
```

To regenerate for a specific API version:

```bash
cd api
make models VERSION=v20240610preview
```

The `make models` target:
1. Runs `tsp compile` with `--emit @azure-tools/typespec-go` and appropriate options
2. Formats imports with `goimports`
3. Removes client/SDK files (client_factory, *_client, options, responses) — only model files are kept

### Supported API versions

| VERSION tag | TypeSpec API version |
|---|---|
| `v20251223preview` | `2025-12-23-preview` |
| `v20240610preview` | `2024-06-10-preview` |
