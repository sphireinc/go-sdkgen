#!/usr/bin/env bash
set -euo pipefail

OUT_DIR="./sdk-example-output"
EXAMPLES_DIR="./examples"

echo "== go-sdkgen: running example generations =="

if [ -d "${OUT_DIR}" ]; then
  echo "Removing existing ${OUT_DIR}"
  rm -rf "${OUT_DIR}"
fi

mkdir -p "${OUT_DIR}"

for f in swagger_telephone.json swagger_dog_parlor.json swagger_customer_booking.json; do
  if [ ! -f "${EXAMPLES_DIR}/${f}" ]; then
    echo "ERROR: missing ${EXAMPLES_DIR}/${f}"
    exit 1
  fi
done

echo "Building sdkgen binary..."
go build -o ./sdkgen ./cmd/sdkgen

gen_pair () {
  local input="$1"
  local name="$2"
  local outbase="$3"

  echo "Generating ${name} (TS + JS) from ${input}"
  ./sdkgen --input "${input}" --out "${OUT_DIR}/${outbase}-ts" --lang ts --name "${name}"
  ./sdkgen --input "${input}" --out "${OUT_DIR}/${outbase}-js" --lang js --name "${name}"
}

gen_pair "${EXAMPLES_DIR}/swagger_telephone.json" "TelephoneSDK" "telephone"
gen_pair "${EXAMPLES_DIR}/swagger_dog_parlor.json" "DogParlorSDK" "dog-parlor"
gen_pair "${EXAMPLES_DIR}/swagger_customer_booking.json" "CustomerBookingSDK" "customer-booking"

echo
echo "✅ Example SDKs generated successfully at ${OUT_DIR}"