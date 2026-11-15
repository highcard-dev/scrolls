#!/usr/bin/env bash
set -euo pipefail
[[ "$1" == run && "$2" == ./scripts/stage-scroll-ui ]]
if [[ "$3" == --preserve-ui ]]; then
  [[ "$4" == */spacedruids/* || "$4" == */spacedruids-multiplayer/* ]]
  printf '%s|--preserve-ui\n' "$4" >> "$FAKE_DRUID_STATE_DIR/staging.log"
  mkdir -p "$5"
  exit
fi
case "$3" in
  */minecraft/*) [[ "$5" == */ui-minecraft/dist/app.wasm ]] ;;
  *) [[ "$5" == */ui/dist/app.wasm ]] ;;
esac
printf '%s|%s\n' "$3" "$5" >> "$FAKE_DRUID_STATE_DIR/staging.log"
mkdir -p "$4"
