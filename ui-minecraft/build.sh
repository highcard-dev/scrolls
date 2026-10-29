#!/bin/sh
# Standalone Rust guest builder. Requires Cargo and wasm-tools, not Node/npm.
set -eu

if [ "$#" -lt 2 ]; then
  echo "Usage: $0 <Cargo.toml> <output-dir> [--name=app] [--document=view.openui] [--watch]" >&2
  exit 1
fi
script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
manifest=$(CDPATH= cd -- "$(dirname -- "$1")" && pwd)/$(basename -- "$1")
output=$2
shift 2
name=app
document=
watch=0
for option do
  case "$option" in
    --name=*) name=${option#*=} ;;
    --document=*) document=${option#*=} ;;
    --watch) watch=1 ;;
    *) echo "Unknown option: $option" >&2; exit 1 ;;
  esac
done
case "$name" in
  ''|*[!a-zA-Z0-9_-]*) echo "Artifact name must be a filename without a path" >&2; exit 1 ;;
esac
command -v cargo >/dev/null 2>&1 || {
  echo "Cargo is required; install Rust and rustup target add wasm32-unknown-unknown" >&2
  exit 1
}
command -v wasm-tools >/dev/null 2>&1 || {
  echo "wasm-tools is required; cargo install --locked wasm-tools --version 1.262.0" >&2
  exit 1
}
wit=$(CDPATH= cd -- "${DRUID_WIT_PATH:-$script_dir/wit}" && pwd)
if [ -n "$document" ]; then
  test -s "$document" || { echo "OpenUI document is missing or empty: $document" >&2; exit 1; }
  document=$(CDPATH= cd -- "$(dirname -- "$document")" && pwd)/$(basename -- "$document")
fi
mkdir -p -- "$output"
output=$(CDPATH= cd -- "$output" && pwd)
if [ "$watch" = 1 ]; then
  fingerprint() {
    {
      find "$(dirname -- "$manifest")" -type d \( -name target -o -name .git \) -prune -o \
        -type f \( -name '*.rs' -o -name '*.toml' -o -name '*.lock' \) -exec cksum {} +
      find "$wit" -type f -name '*.wit' -exec cksum {} +
      if [ -n "$document" ]; then cksum "$document"; fi
    } | cksum
  }
  previous=
  trap 'exit 0' HUP INT TERM
  while :; do
    current=$(fingerprint)
    if [ "$current" != "$previous" ]; then
      previous=$current
      if [ -n "$document" ]; then
        "$0" "$manifest" "$output" "--name=$name" "--document=$document" || true
      else
        "$0" "$manifest" "$output" "--name=$name" || true
      fi
    fi
    sleep 1
  done
fi
staging=$(mktemp -d "$output/.build-XXXXXX")
trap 'rm -rf -- "$staging"' EXIT
trap 'exit 1' HUP INT TERM

# Explicit rustc output avoids assumptions about crate names, workspaces or target-dir.
(
  cd -- "$(dirname -- "$manifest")"
  DRUID_WIT_PATH="$wit" cargo rustc --manifest-path "$manifest" \
    --target wasm32-unknown-unknown --release --locked --lib --crate-type cdylib \
    -- --emit="link=$staging/core.wasm"
)
wasm-tools component new "$staging/core.wasm" -o "$staging/$name.wasm"
wasm-tools component wit "$staging/$name.wasm" > "$staging/component.wit"
if ! grep -Eq '^[[:space:]]*export druid:openui/client;' "$staging/component.wit"; then
  echo "The guest must export druid:openui/client" >&2
  exit 1
fi
if [ -n "$document" ]; then
  cp -- "$document" "$staging/$name.openui"
  mv -- "$staging/$name.openui" "$output/$name.openui"
fi
mv -- "$staging/$name.wasm" "$output/$name.wasm"
echo "Built $output/$name.wasm"
