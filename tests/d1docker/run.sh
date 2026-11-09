#!/usr/bin/env bash
# Runs the D1 conformance suites against a real local D1 (Miniflare/workerd) inside Docker.
# usage: tests/d1docker/run.sh [go test flags]   (default: -test.v)
set -euo pipefail
here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT

cd "$here"
GOOS=js GOARCH=wasm go test -c -o "$work/d1.wasm" .
cp "$(go env GOROOT)/lib/wasm/wasm_exec_node.js" "$(go env GOROOT)/lib/wasm/wasm_exec.js" "$work/"
docker build -q -t webtyp-d1-conformance "$here" >/dev/null
docker run --rm -v "$work:/work:ro" webtyp-d1-conformance /work/wasm_exec_node.js /work/d1.wasm "${@:--test.v}"
