# go-sdkgen (Go) → TS/JS REST SDK generator

Generates a small axios-based REST SDK from a Swagger (OpenAPI v2) `swagger.json`.

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

## Options

	--input path to swagger.json
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