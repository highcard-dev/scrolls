#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

test -s ui/dist/app.wasm
test -s ui/dist/app.openui
test -s ui-minecraft/dist/app.wasm
test -s ui-minecraft/dist/app.openui
test -s ui-minecraft/public/dist/app.wasm
test -s ui-minecraft/public/dist/app.openui
go test ./scripts/stage-scroll-ui

echo "Scroll UI bundle and all released family mappings are valid."
