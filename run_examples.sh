#!/usr/bin/env bash
set -euo pipefail

OUT_DIR="./sdk-example-output"
EXAMPLES_DIR="./examples"

echo "== swagger-sdkgen: running example generations =="

# Clean output directory
if [ -d "${OUT_DIR}" ]; then
  echo "Removing existing ${OUT_DIR}"
  rm -rf "${OUT_DIR}"
fi

mkdir -p "${OUT_DIR}"

# Ensure examples exist
if [ ! -d "${EXAMPLES_DIR}" ]; then
  echo "ERROR: ${EXAMPLES_DIR} directory not found"
  exit 1
fi

if [ ! -f "${EXAMPLES_DIR}/swagger_jobs.json" ]; then
  echo "ERROR: missing examples/swagger_jobs.json"
  exit 1
fi

if [ ! -f "${EXAMPLES_DIR}/swagger_misc.json" ]; then
  echo "ERROR: missing examples/swagger_misc.json"
  exit 1
fi

echo "Building sdkgen binary..."
go build -o ./sdkgen ./cmd/sdkgen

echo "Running generator on swagger_jobs.json (TS + JS)"
./sdkgen \
  --input "${EXAMPLES_DIR}/swagger_jobs.json" \
  --out "${OUT_DIR}/jobs-ts" \
  --lang ts \
  --name JobEngineSDK

./sdkgen \
  --input "${EXAMPLES_DIR}/swagger_jobs.json" \
  --out "${OUT_DIR}/jobs-js" \
  --lang js \
  --name JobEngineSDK

echo "Running generator on swagger_misc.json (TS + JS)"
./sdkgen \
  --input "${EXAMPLES_DIR}/swagger_misc.json" \
  --out "${OUT_DIR}/misc-ts" \
  --lang ts \
  --name MiscSDK

./sdkgen \
  --input "${EXAMPLES_DIR}/swagger_misc.json" \
  --out "${OUT_DIR}/misc-js" \
  --lang js \
  --name MiscSDK

echo
echo "✅ Example SDKs generated successfully."
echo "Output directory:"
echo "  ${OUT_DIR}"
echo
echo "Structure:"
echo "  ${OUT_DIR}/jobs-ts"
echo "  ${OUT_DIR}/jobs-js"
echo "  ${OUT_DIR}/misc-ts"
echo "  ${OUT_DIR}/misc-js"