#!/usr/bin/env bash
#
# Builds the notification template preview WASM bundle into docs/assets/.

set -euo pipefail

cd "$(git rev-parse --show-toplevel)"

goroot=$(go env GOROOT)
# wasm_exec.js moved from misc/wasm to lib/wasm in Go 1.24.
for candidate in "$goroot/lib/wasm/wasm_exec.js" "$goroot/misc/wasm/wasm_exec.js"; do
  if [ -f "$candidate" ]; then
    cp "$candidate" ./docs/assets/
    break
  fi
done

if [ ! -f ./docs/assets/wasm_exec.js ]; then
  echo "wasm_exec.js not found under $goroot" >&2
  exit 1
fi

GOARCH=wasm GOOS=js go build -o ./docs/assets/tplprev.wasm ./tplprev