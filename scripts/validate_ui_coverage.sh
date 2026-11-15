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
test -s scrolls/spacedruids/1.0.0/private/dist/app.wasm
test -s scrolls/spacedruids/1.0.0/private/dist/app.openui
test -s scrolls/spacedruids/1.0.0/public/dist/app.wasm
test -s scrolls/spacedruids/1.0.0/public/dist/app.openui
cmp scrolls/spacedruids/1.0.0/private/dist/app.wasm scrolls/spacedruids/1.0.0/public/dist/app.wasm
cmp scrolls/spacedruids/1.0.0/private/dist/app.openui scrolls/spacedruids/1.0.0/public/dist/app.openui
for scope in private public; do
  test -s "scrolls/spacedruids-multiplayer/1.0.0/$scope/dist/app.wasm"
  test -s "scrolls/spacedruids-multiplayer/1.0.0/$scope/dist/app.openui"
done
go test ./scripts/stage-scroll-ui

echo "Scroll UI bundle and all released family mappings are valid."
