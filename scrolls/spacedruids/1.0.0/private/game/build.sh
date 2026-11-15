#!/bin/sh
set -eu
cd -- "$(dirname -- "$0")"
sh ./compile.sh Cargo.toml ../dist --document=src/view.openui
mkdir -p ../../public/dist
cp ../dist/app.wasm ../dist/app.openui ../../public/dist/
