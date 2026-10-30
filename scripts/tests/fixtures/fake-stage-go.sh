#!/usr/bin/env bash
set -euo pipefail
[[ "$1" == run && "$2" == ./scripts/stage-scroll-ui ]]
case "$3" in
  */minecraft/*) [[ "$5" == */ui-minecraft/dist/app.wasm ]] ;;
  *) [[ "$5" == */ui/dist/app.wasm ]] ;;
esac
printf '%s|%s\n' "$3" "$5" >> "$FAKE_DRUID_STATE_DIR/staging.log"
mkdir -p "$4"
