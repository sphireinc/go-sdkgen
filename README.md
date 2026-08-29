[![CI](https://github.com/sphireinc/go-sdkgen/actions/workflows/ci.yml/badge.svg)](https://github.com/sphireinc/go-sdkgen/actions/workflows/ci.yml)

# go-sdkgen (Go) → TS/JS REST SDK generator

Generates a deterministic axios-based REST SDK from OpenAPI 3.x YAML/JSON or legacy Swagger (OpenAPI v2) JSON. OpenAPI 3 output is schema-derived TypeScript.

## Install

```bash
go install ./cmd/sdkgen
```

## Usage

```bash
sdkgen \
  --input ./swagger.json \
  --out ./sdk \
  --lang ts \
  --name MyApiSDK
```

For a checked-in configuration (recommended for CI):

```bash
sdkgen --config api/sdk/sdkgen.yaml
```

Example configuration:

```yaml
input: ./contract/openapi.yaml
output: ../apps/web/src/generated/api
language: ts
name: ClaimsSDK
auth: bearer
transport: axios
emitSchemas: true
emitOperations: true
generatorVersion: c9ca2e213df8f65029f97c7d52a6aa3886c4c875
```

Paths in a config are resolved relative to the config file. Pin the generator to the commit or tagged release used by CI; the current pre-upgrade baseline is `c9ca2e213df8f65029f97c7d52a6aa3886c4c875`. After this change is committed, the consuming project should replace `generatorVersion` with that resulting commit and pin it in CI.

OpenAPI 3.1 documents are validated before generation. Local references are resolved automatically; external references are rejected unless `allowExternalRefs: true` is set. The generated TypeScript separates `types.ts`, operation functions, and the axios runtime. Authentication remains a thin runtime hook (`setTokenProvider`); applications may wrap `req` for correlation IDs and error normalization.

## OpenAPI support boundary

The generator supports OpenAPI 3.1 documents, JSON Schema primitive/object/array schemas, enums, nullable unions, `oneOf`/`anyOf`/`allOf`, local references, external references when enabled, read/write-only and descriptive/constraint metadata during validation, bearer security configuration, and binary content as `string` values. `prefixItems` tuple schemas, callbacks, links, and non-bearer security schemes are parsed by the loader but are not emitted with specialized TypeScript behavior yet; contracts requiring those features should fail validation or use a documented wrapper until support is added. Parameters are generated into operation-specific types, and `setRequestIdProvider`/`setErrorNormalizer` provide the runtime extension points for correlation and error policy.

For migration from the old Swagger 2 path, keep the existing CLI flags unchanged or create `api/sdk/sdkgen.yaml` with the fields above, then run `sdkgen --config api/sdk/sdkgen.yaml`. Generated output belongs in `apps/web/src/generated/api/`; it must be regenerated from the authoritative contract rather than edited manually.

## Options

	--input path to swagger.json
	--config path to sdkgen.yaml (OpenAPI 3.x/legacy v2)
	--out output directory
	--lang ts or js
	--name SDK name (used in headers/comments)
	--baseUrlVar name of the exported baseUrl variable (default: baseApiUrl)
	--auth none|bearer (default: bearer)
	--tokenFn token function name used in generated requests (default: getToken)

## Examples

### TypeScript consumer example:

```typescript
import { setBaseUrl, setTokenProvider, fetchWorkflows } from "./sdk";

setBaseUrl("https://my-api.example.com");
setTokenProvider(() => localStorage.getItem("token") ?? "");

const res = await fetchWorkflows({ id: "abc" });
if (!res.success) throw new Error(res.error);
console.log(res.data);
```
