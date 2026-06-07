#!/usr/bin/env bash
set -euo pipefail

targets="${1:-all-steam}"

: "${SCROLL_REGISTRY_HOST:?SCROLL_REGISTRY_HOST is required}"
: "${SCROLL_REGISTRY_USER:?SCROLL_REGISTRY_USER is required}"
: "${SCROLL_REGISTRY_PASSWORD:?SCROLL_REGISTRY_PASSWORD is required}"

export DRUID_RUNTIME_IMAGE="${DRUID_RUNTIME_IMAGE:-artifacts.druid.gg/druid-team/druid:v0.1.248}"
export DRUID_STEAM_RUNTIME_IMAGE="${DRUID_STEAM_RUNTIME_IMAGE:-artifacts.druid.gg/druid-team/druid:v0.1.248-steamcmd}"
export PREBUILD_DOCKER_PLATFORM="${PREBUILD_DOCKER_PLATFORM:-linux/amd64}"

echo "Targets: ${targets}"
echo "Runtime image: ${DRUID_RUNTIME_IMAGE}"
echo "Steam runtime image: ${DRUID_STEAM_RUNTIME_IMAGE}"

go run ./scripts/prebuild --targets "${targets}"
