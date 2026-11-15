#!/bin/sh
set -eu
cd -- "$(dirname -- "$0")"
sh ./compile.sh Cargo.toml ../dist --document=src/view.openui
url=${DRUID_PORT_HTTP_URL:-}
if [ -n "$url" ]; then
  # Runtime-generated endpoint: only a scheme, hostname and optional port.
  url=$(printf '%s' "$url" | sed 's/^http/ws/')
  sed "s|^\$serverUrl = \"\"|\$serverUrl = \"$url/ws\"|" src/view.openui > ../dist/app.openui
fi
mkdir -p ../../public/dist
cp ../dist/app.wasm ../dist/app.openui ../../public/dist/
