#!/usr/bin/env bash

set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

# Keep the catalog below readable: every release/local artifact is an explicit
# `run druid push ...` line. These env vars only retarget that catalog.
DRUID_BIN="${DRUID_BIN:-druid}"
SCROLL_PUSH_DRY_RUN="${SCROLL_PUSH_DRY_RUN:-0}"
SCROLL_REGISTRY_HOST="${SCROLL_REGISTRY_HOST:-artifacts.druid.gg}"
SCROLL_REGISTRY_NAMESPACE="${SCROLL_REGISTRY_NAMESPACE:-druid-team}"
SCROLL_REGISTRY_USER="${SCROLL_REGISTRY_USER:-}"
SCROLL_REGISTRY_PASSWORD="${SCROLL_REGISTRY_PASSWORD:-}"
DRUID_CLI_VERSION="${DRUID_CLI_VERSION:-v0.1.249}"
SCROLL_PUSH_CATEGORIES="${SCROLL_PUSH_CATEGORIES:-1}"
SCROLL_PUSH_ARTIFACTS="${SCROLL_PUSH_ARTIFACTS:-1}"
SCROLL_PUSH_JOBS="${SCROLL_PUSH_JOBS:-1}"

if [[ ! "$SCROLL_PUSH_JOBS" =~ ^[1-9][0-9]*$ ]]; then
  echo "SCROLL_PUSH_JOBS must be a positive integer (got: $SCROLL_PUSH_JOBS)" >&2
  exit 2
fi

# Used by PR CI to publish the same catalog as preview tags, for example:
# SCROLL_REGISTRY_NAMESPACE=druid-team-experimental SCROLL_TAG_SUFFIX=-pr123.
SCROLL_TAG_SUFFIX="${SCROLL_TAG_SUFFIX:-}"

registry_host="${SCROLL_REGISTRY_HOST#http://}"
registry_host="${registry_host#https://}"
registry_host="${registry_host%%/*}"
registry_prefix="${registry_host}/${SCROLL_REGISTRY_NAMESPACE}"
runtime_namespace="${SCROLL_REGISTRY_RUNTIME_NAMESPACE:-${SCROLL_REGISTRY_NAMESPACE}}"
runtime_prefix="${registry_host}/${runtime_namespace}"
runtime_image="${DRUID_SCROLL_RUNTIME_IMAGE:-${runtime_prefix}/druid:${DRUID_CLI_VERSION}}"
steamcmd_image="${DRUID_SCROLL_STEAMCMD_IMAGE:-${runtime_image}-steamcmd}"

push_pids=()
push_phase_failed=0

wait_for_oldest_push() {
  local pid="${push_pids[0]}"

  if ! wait "$pid"; then
    push_phase_failed=1
  fi
  push_pids=("${push_pids[@]:1}")
}

wait_for_push_phase() {
  while ((${#push_pids[@]} > 0)); do
    wait_for_oldest_push
  done

  local failed="$push_phase_failed"
  push_phase_failed=0
  return "$failed"
}

login_if_configured() {
  if [[ "$SCROLL_PUSH_DRY_RUN" = "1" ]]; then
    return 0
  fi
  if [[ -n "$SCROLL_REGISTRY_USER" && -n "$SCROLL_REGISTRY_PASSWORD" ]]; then
    "$DRUID_BIN" login --host "$registry_host" --user "$SCROLL_REGISTRY_USER" --password "$SCROLL_REGISTRY_PASSWORD"
  fi
}

run() {
  local command="$*"

  # Add the optional suffix only to artifact tags. Category pushes are metadata
  # for the stable repository and do not get PR suffixes.
  if [[ -n "$SCROLL_TAG_SUFFIX" && "$command" == druid\ push\ * && "$command" != druid\ push\ category\ * ]]; then
    local rest ref after last
    rest="${command#druid push }"
    ref="${rest%% *}"
    after="${rest#"$ref"}"
    last="${ref##*/}"
    if [[ "$last" == *:* ]]; then
      ref="${ref}${SCROLL_TAG_SUFFIX}"
    else
      ref="${ref}:${SCROLL_TAG_SUFFIX#-}"
    fi
    command="druid push ${ref}${after}"
  fi

  # The catalog intentionally uses production refs. Retarget them here for
  # local Harbor, staging, or PR namespaces without changing every push line.
  command="${command//artifacts.druid.gg\/druid-team\/druid:v0.1.249-steamcmd/$steamcmd_image}"
  command="${command//artifacts.druid.gg\/druid-team\/druid:v0.1.249/$runtime_image}"
  command="${command//artifacts.druid.gg\/druid-team/$registry_prefix}"
  command="${command/#druid /"$DRUID_BIN" }"
  echo "$command"
  if [[ "$SCROLL_PUSH_DRY_RUN" = "1" ]]; then
    return 0
  fi

  if ((SCROLL_PUSH_JOBS == 1)); then
    eval "$command"
    return
  fi

  eval "$command" &
  push_pids+=("$!")
  if ((${#push_pids[@]} >= SCROLL_PUSH_JOBS)); then
    wait_for_oldest_push
  fi
}

push_release_categories() {
  run druid push category artifacts.druid.gg/druid-team/scroll-minecraft-spigot minecraft ./scrolls/minecraft/minecraft-spigot/.meta
  run druid push category artifacts.druid.gg/druid-team/scroll-minecraft-vanilla minecraft ./scrolls/minecraft/minecraft-vanilla/.meta
  run druid push category artifacts.druid.gg/druid-team/scroll-minecraft-paper minecraft ./scrolls/minecraft/papermc/.meta
  run druid push category artifacts.druid.gg/druid-team/scroll-minecraft-forge minecraft ./scrolls/minecraft/forge/.meta
  run druid push category artifacts.druid.gg/druid-team/scroll-minecraft-cuberite minecraft ./scrolls/minecraft/cuberite/.meta
  run druid push category artifacts.druid.gg/druid-team/scroll-rust-oxide rust ./scrolls/rust/rust-oxide/.meta
  run druid push category artifacts.druid.gg/druid-team/scroll-rust-vanilla rust ./scrolls/rust/rust-vanilla/.meta
  run druid push category artifacts.druid.gg/druid-team/scroll-lgsm:pwserver palworld ./scrolls/lgsm/pwserver/.meta
  run druid push category artifacts.druid.gg/druid-team/scroll-lgsm:arkserver ark ./scrolls/lgsm/arkserver/.meta
  run druid push category artifacts.druid.gg/druid-team/scroll-lgsm:untserver unturned ./scrolls/lgsm/untserver/.meta
  run druid push category artifacts.druid.gg/druid-team/scroll-lgsm:dayzserver dayz ./scrolls/lgsm/dayzserver/.meta
  run druid push category artifacts.druid.gg/druid-team/scroll-lgsm:sdtdserver 7days ./scrolls/lgsm/sdtdserver/.meta
  run druid push category artifacts.druid.gg/druid-team/scroll-lgsm:gmodserver gmod ./scrolls/lgsm/gmodserver/.meta
  run druid push category artifacts.druid.gg/druid-team/scroll-lgsm:cs2server cs2 ./scrolls/lgsm/cs2server/.meta
  run druid push category artifacts.druid.gg/druid-team/scroll-lgsm:pzserver zomboid ./scrolls/lgsm/pzserver/.meta
  run druid push category artifacts.druid.gg/druid-team/scroll-lgsm:csgoserver csgo ./scrolls/lgsm/csgoserver/.meta
  run druid push category artifacts.druid.gg/druid-team/scroll-hytale:hytale-standalone hytale ./scrolls/hytale/hytale-standalone/.meta
  run druid push category artifacts.druid.gg/druid-team/scroll-hytale:hytale-druid-gg hytale ./scrolls/hytale/hytale-druid-gg/.meta
}

push_release_artifacts() {
  run druid push artifacts.druid.gg/druid-team/scroll-minecraft-spigot:1.17 ./scrolls/minecraft/minecraft-spigot/1.17 -p main=25565 -p rcon=25575 -i artifacts.druid.gg/druid-team/druid:v0.1.249 --min-disk 3Gi --min-ram 512Mi --min-cpu 0.25 --smart --category minecraft
  run druid push artifacts.druid.gg/druid-team/scroll-minecraft-spigot:1.17.1 ./scrolls/minecraft/minecraft-spigot/1.17.1 -p main=25565 -p rcon=25575 -i artifacts.druid.gg/druid-team/druid:v0.1.249 --min-disk 3Gi --min-ram 512Mi --min-cpu 0.25 --smart --category minecraft
  run druid push artifacts.druid.gg/druid-team/scroll-minecraft-spigot:1.18 ./scrolls/minecraft/minecraft-spigot/1.18 -p main=25565 -p rcon=25575 -i artifacts.druid.gg/druid-team/druid:v0.1.249 --min-disk 3Gi --min-ram 512Mi --min-cpu 0.25 --smart --category minecraft
  run druid push artifacts.druid.gg/druid-team/scroll-minecraft-spigot:1.18.1 ./scrolls/minecraft/minecraft-spigot/1.18.1 -p main=25565 -p rcon=25575 -i artifacts.druid.gg/druid-team/druid:v0.1.249 --min-disk 3Gi --min-ram 512Mi --min-cpu 0.25 --smart --category minecraft
  run druid push artifacts.druid.gg/druid-team/scroll-minecraft-spigot:1.18.2 ./scrolls/minecraft/minecraft-spigot/1.18.2 -p main=25565 -p rcon=25575 -i artifacts.druid.gg/druid-team/druid:v0.1.249 --min-disk 3Gi --min-ram 512Mi --min-cpu 0.25 --smart --category minecraft
  run druid push artifacts.druid.gg/druid-team/scroll-minecraft-spigot:1.19 ./scrolls/minecraft/minecraft-spigot/1.19 -p main=25565 -p rcon=25575 -i artifacts.druid.gg/druid-team/druid:v0.1.249 --min-disk 3Gi --min-ram 512Mi --min-cpu 0.25 --smart --category minecraft
  run druid push artifacts.druid.gg/druid-team/scroll-minecraft-spigot:1.19.1 ./scrolls/minecraft/minecraft-spigot/1.19.1 -p main=25565 -p rcon=25575 -i artifacts.druid.gg/druid-team/druid:v0.1.249 --min-disk 3Gi --min-ram 512Mi --min-cpu 0.25 --smart --category minecraft
  run druid push artifacts.druid.gg/druid-team/scroll-minecraft-spigot:1.19.2 ./scrolls/minecraft/minecraft-spigot/1.19.2 -p main=25565 -p rcon=25575 -i artifacts.druid.gg/druid-team/druid:v0.1.249 --min-disk 3Gi --min-ram 512Mi --min-cpu 0.25 --smart --category minecraft
  run druid push artifacts.druid.gg/druid-team/scroll-minecraft-spigot:1.19.3 ./scrolls/minecraft/minecraft-spigot/1.19.3 -p main=25565 -p rcon=25575 -i artifacts.druid.gg/druid-team/druid:v0.1.249 --min-disk 3Gi --min-ram 512Mi --min-cpu 0.25 --smart --category minecraft
  run druid push artifacts.druid.gg/druid-team/scroll-minecraft-spigot:1.19.4 ./scrolls/minecraft/minecraft-spigot/1.19.4 -p main=25565 -p rcon=25575 -i artifacts.druid.gg/druid-team/druid:v0.1.249 --min-disk 3Gi --min-ram 512Mi --min-cpu 0.25 --smart --category minecraft
  run druid push artifacts.druid.gg/druid-team/scroll-minecraft-spigot:1.20.1 ./scrolls/minecraft/minecraft-spigot/1.20.1 -p main=25565 -p rcon=25575 -i artifacts.druid.gg/druid-team/druid:v0.1.249 --min-disk 3Gi --min-ram 512Mi --min-cpu 0.25 --smart --category minecraft
  run druid push artifacts.druid.gg/druid-team/scroll-minecraft-spigot:1.20.2 ./scrolls/minecraft/minecraft-spigot/1.20.2 -p main=25565 -p rcon=25575 -i artifacts.druid.gg/druid-team/druid:v0.1.249 --min-disk 3Gi --min-ram 512Mi --min-cpu 0.25 --smart --category minecraft
  run druid push artifacts.druid.gg/druid-team/scroll-minecraft-spigot:1.20.4 ./scrolls/minecraft/minecraft-spigot/1.20.4 -p main=25565 -p rcon=25575 -i artifacts.druid.gg/druid-team/druid:v0.1.249 --min-disk 3Gi --min-ram 512Mi --min-cpu 0.25 --smart --category minecraft
  run druid push artifacts.druid.gg/druid-team/scroll-minecraft-spigot:1.20.6 ./scrolls/minecraft/minecraft-spigot/1.20.6 -p main=25565 -p rcon=25575 -i artifacts.druid.gg/druid-team/druid:v0.1.249 --min-disk 3Gi --min-ram 512Mi --min-cpu 0.25 --smart --category minecraft
  run druid push artifacts.druid.gg/druid-team/scroll-minecraft-spigot:1.21.1 ./scrolls/minecraft/minecraft-spigot/1.21.1 -p main=25565 -p rcon=25575 -i artifacts.druid.gg/druid-team/druid:v0.1.249 --min-disk 3Gi --min-ram 512Mi --min-cpu 0.25 --smart --category minecraft
  run druid push artifacts.druid.gg/druid-team/scroll-minecraft-spigot:1.21.3 ./scrolls/minecraft/minecraft-spigot/1.21.3 -p main=25565 -p rcon=25575 -i artifacts.druid.gg/druid-team/druid:v0.1.249 --min-disk 3Gi --min-ram 512Mi --min-cpu 0.25 --smart --category minecraft
  run druid push artifacts.druid.gg/druid-team/scroll-minecraft-spigot:1.21.4 ./scrolls/minecraft/minecraft-spigot/1.21.4 -p main=25565 -p rcon=25575 -i artifacts.druid.gg/druid-team/druid:v0.1.249 --min-disk 3Gi --min-ram 512Mi --min-cpu 0.25 --smart --category minecraft
  run druid push artifacts.druid.gg/druid-team/scroll-minecraft-spigot:1.21.5 ./scrolls/minecraft/minecraft-spigot/1.21.5 -p main=25565 -p rcon=25575 -i artifacts.druid.gg/druid-team/druid:v0.1.249 --min-disk 3Gi --min-ram 512Mi --min-cpu 0.25 --smart --category minecraft
  run druid push artifacts.druid.gg/druid-team/scroll-minecraft-spigot:1.21.6 ./scrolls/minecraft/minecraft-spigot/1.21.6 -p main=25565 -p rcon=25575 -i artifacts.druid.gg/druid-team/druid:v0.1.249 --min-disk 3Gi --min-ram 512Mi --min-cpu 0.25 --smart --category minecraft
  run druid push artifacts.druid.gg/druid-team/scroll-minecraft-spigot:1.21.8 ./scrolls/minecraft/minecraft-spigot/1.21.8 -p main=25565 -p rcon=25575 -i artifacts.druid.gg/druid-team/druid:v0.1.249 --min-disk 3Gi --min-ram 512Mi --min-cpu 0.25 --smart --category minecraft
  run druid push artifacts.druid.gg/druid-team/scroll-minecraft-vanilla:1.17 ./scrolls/minecraft/minecraft-vanilla/1.17 -p main=25565 -p rcon=25575 -i artifacts.druid.gg/druid-team/druid:v0.1.249 --min-disk 3Gi --min-ram 512Mi --min-cpu 0.25 --smart --category minecraft
  run druid push artifacts.druid.gg/druid-team/scroll-minecraft-vanilla:1.17.1 ./scrolls/minecraft/minecraft-vanilla/1.17.1 -p main=25565 -p rcon=25575 -i artifacts.druid.gg/druid-team/druid:v0.1.249 --min-disk 3Gi --min-ram 512Mi --min-cpu 0.25 --smart --category minecraft
  run druid push artifacts.druid.gg/druid-team/scroll-minecraft-vanilla:1.18 ./scrolls/minecraft/minecraft-vanilla/1.18 -p main=25565 -p rcon=25575 -i artifacts.druid.gg/druid-team/druid:v0.1.249 --min-disk 3Gi --min-ram 512Mi --min-cpu 0.25 --smart --category minecraft
  run druid push artifacts.druid.gg/druid-team/scroll-minecraft-vanilla:1.18.1 ./scrolls/minecraft/minecraft-vanilla/1.18.1 -p main=25565 -p rcon=25575 -i artifacts.druid.gg/druid-team/druid:v0.1.249 --min-disk 3Gi --min-ram 512Mi --min-cpu 0.25 --smart --category minecraft
  run druid push artifacts.druid.gg/druid-team/scroll-minecraft-vanilla:1.18.2 ./scrolls/minecraft/minecraft-vanilla/1.18.2 -p main=25565 -p rcon=25575 -i artifacts.druid.gg/druid-team/druid:v0.1.249 --min-disk 3Gi --min-ram 512Mi --min-cpu 0.25 --smart --category minecraft
  run druid push artifacts.druid.gg/druid-team/scroll-minecraft-vanilla:1.19 ./scrolls/minecraft/minecraft-vanilla/1.19 -p main=25565 -p rcon=25575 -i artifacts.druid.gg/druid-team/druid:v0.1.249 --min-disk 3Gi --min-ram 512Mi --min-cpu 0.25 --smart --category minecraft
  run druid push artifacts.druid.gg/druid-team/scroll-minecraft-vanilla:1.19.1 ./scrolls/minecraft/minecraft-vanilla/1.19.1 -p main=25565 -p rcon=25575 -i artifacts.druid.gg/druid-team/druid:v0.1.249 --min-disk 3Gi --min-ram 512Mi --min-cpu 0.25 --smart --category minecraft
  run druid push artifacts.druid.gg/druid-team/scroll-minecraft-vanilla:1.19.2 ./scrolls/minecraft/minecraft-vanilla/1.19.2 -p main=25565 -p rcon=25575 -i artifacts.druid.gg/druid-team/druid:v0.1.249 --min-disk 3Gi --min-ram 512Mi --min-cpu 0.25 --smart --category minecraft
  run druid push artifacts.druid.gg/druid-team/scroll-minecraft-vanilla:1.19.3 ./scrolls/minecraft/minecraft-vanilla/1.19.3 -p main=25565 -p rcon=25575 -i artifacts.druid.gg/druid-team/druid:v0.1.249 --min-disk 3Gi --min-ram 512Mi --min-cpu 0.25 --smart --category minecraft
  run druid push artifacts.druid.gg/druid-team/scroll-minecraft-vanilla:1.19.4 ./scrolls/minecraft/minecraft-vanilla/1.19.4 -p main=25565 -p rcon=25575 -i artifacts.druid.gg/druid-team/druid:v0.1.249 --min-disk 3Gi --min-ram 512Mi --min-cpu 0.25 --smart --category minecraft
  run druid push artifacts.druid.gg/druid-team/scroll-minecraft-vanilla:1.20.1 ./scrolls/minecraft/minecraft-vanilla/1.20.1 -p main=25565 -p rcon=25575 -i artifacts.druid.gg/druid-team/druid:v0.1.249 --min-disk 3Gi --min-ram 512Mi --min-cpu 0.25 --smart --category minecraft
  run druid push artifacts.druid.gg/druid-team/scroll-minecraft-vanilla:1.20.2 ./scrolls/minecraft/minecraft-vanilla/1.20.2 -p main=25565 -p rcon=25575 -i artifacts.druid.gg/druid-team/druid:v0.1.249 --min-disk 3Gi --min-ram 512Mi --min-cpu 0.25 --smart --category minecraft
  run druid push artifacts.druid.gg/druid-team/scroll-minecraft-vanilla:1.20.4 ./scrolls/minecraft/minecraft-vanilla/1.20.4 -p main=25565 -p rcon=25575 -i artifacts.druid.gg/druid-team/druid:v0.1.249 --min-disk 3Gi --min-ram 512Mi --min-cpu 0.25 --smart --category minecraft
  run druid push artifacts.druid.gg/druid-team/scroll-minecraft-vanilla:1.20.6 ./scrolls/minecraft/minecraft-vanilla/1.20.6 -p main=25565 -p rcon=25575 -i artifacts.druid.gg/druid-team/druid:v0.1.249 --min-disk 3Gi --min-ram 512Mi --min-cpu 0.25 --smart --category minecraft
  run druid push artifacts.druid.gg/druid-team/scroll-minecraft-vanilla:1.21.1 ./scrolls/minecraft/minecraft-vanilla/1.21.1 -p main=25565 -p rcon=25575 -i artifacts.druid.gg/druid-team/druid:v0.1.249 --min-disk 3Gi --min-ram 512Mi --min-cpu 0.25 --smart --category minecraft
  run druid push artifacts.druid.gg/druid-team/scroll-minecraft-vanilla:1.21.3 ./scrolls/minecraft/minecraft-vanilla/1.21.3 -p main=25565 -p rcon=25575 -i artifacts.druid.gg/druid-team/druid:v0.1.249 --min-disk 3Gi --min-ram 512Mi --min-cpu 0.25 --smart --category minecraft
  run druid push artifacts.druid.gg/druid-team/scroll-minecraft-vanilla:1.21.4 ./scrolls/minecraft/minecraft-vanilla/1.21.4 -p main=25565 -p rcon=25575 -i artifacts.druid.gg/druid-team/druid:v0.1.249 --min-disk 3Gi --min-ram 512Mi --min-cpu 0.25 --smart --category minecraft
  run druid push artifacts.druid.gg/druid-team/scroll-minecraft-vanilla:1.21.5 ./scrolls/minecraft/minecraft-vanilla/1.21.5 -p main=25565 -p rcon=25575 -i artifacts.druid.gg/druid-team/druid:v0.1.249 --min-disk 3Gi --min-ram 512Mi --min-cpu 0.25 --smart --category minecraft
  run druid push artifacts.druid.gg/druid-team/scroll-minecraft-vanilla:1.21.6 ./scrolls/minecraft/minecraft-vanilla/1.21.6 -p main=25565 -p rcon=25575 -i artifacts.druid.gg/druid-team/druid:v0.1.249 --min-disk 3Gi --min-ram 512Mi --min-cpu 0.25 --smart --category minecraft
  run druid push artifacts.druid.gg/druid-team/scroll-minecraft-vanilla:1.21.7 ./scrolls/minecraft/minecraft-vanilla/1.21.7 -p main=25565 -p rcon=25575 -i artifacts.druid.gg/druid-team/druid:v0.1.249 --min-disk 3Gi --min-ram 512Mi --min-cpu 0.25 --smart --category minecraft
  run druid push artifacts.druid.gg/druid-team/scroll-minecraft-paper:1.17 ./scrolls/minecraft/papermc/1.17 -p main=25565 -p rcon=25575 -i artifacts.druid.gg/druid-team/druid:v0.1.249 --min-disk 3Gi --min-ram 512Mi --min-cpu 0.25 --smart --category minecraft
  run druid push artifacts.druid.gg/druid-team/scroll-minecraft-paper:1.17.1 ./scrolls/minecraft/papermc/1.17.1 -p main=25565 -p rcon=25575 -i artifacts.druid.gg/druid-team/druid:v0.1.249 --min-disk 3Gi --min-ram 512Mi --min-cpu 0.25 --smart --category minecraft
  run druid push artifacts.druid.gg/druid-team/scroll-minecraft-paper:1.18.1 ./scrolls/minecraft/papermc/1.18.1 -p main=25565 -p rcon=25575 -i artifacts.druid.gg/druid-team/druid:v0.1.249 --min-disk 3Gi --min-ram 512Mi --min-cpu 0.25 --smart --category minecraft
  run druid push artifacts.druid.gg/druid-team/scroll-minecraft-paper:1.18.2 ./scrolls/minecraft/papermc/1.18.2 -p main=25565 -p rcon=25575 -i artifacts.druid.gg/druid-team/druid:v0.1.249 --min-disk 3Gi --min-ram 512Mi --min-cpu 0.25 --smart --category minecraft
  run druid push artifacts.druid.gg/druid-team/scroll-minecraft-paper:1.19 ./scrolls/minecraft/papermc/1.19 -p main=25565 -p rcon=25575 -i artifacts.druid.gg/druid-team/druid:v0.1.249 --min-disk 3Gi --min-ram 512Mi --min-cpu 0.25 --smart --category minecraft
  run druid push artifacts.druid.gg/druid-team/scroll-minecraft-paper:1.19.1 ./scrolls/minecraft/papermc/1.19.1 -p main=25565 -p rcon=25575 -i artifacts.druid.gg/druid-team/druid:v0.1.249 --min-disk 3Gi --min-ram 512Mi --min-cpu 0.25 --smart --category minecraft
  run druid push artifacts.druid.gg/druid-team/scroll-minecraft-paper:1.19.2 ./scrolls/minecraft/papermc/1.19.2 -p main=25565 -p rcon=25575 -i artifacts.druid.gg/druid-team/druid:v0.1.249 --min-disk 3Gi --min-ram 512Mi --min-cpu 0.25 --smart --category minecraft
  run druid push artifacts.druid.gg/druid-team/scroll-minecraft-paper:1.19.3 ./scrolls/minecraft/papermc/1.19.3 -p main=25565 -p rcon=25575 -i artifacts.druid.gg/druid-team/druid:v0.1.249 --min-disk 3Gi --min-ram 512Mi --min-cpu 0.25 --smart --category minecraft
  run druid push artifacts.druid.gg/druid-team/scroll-minecraft-paper:1.19.4 ./scrolls/minecraft/papermc/1.19.4 -p main=25565 -p rcon=25575 -i artifacts.druid.gg/druid-team/druid:v0.1.249 --min-disk 3Gi --min-ram 512Mi --min-cpu 0.25 --smart --category minecraft
  run druid push artifacts.druid.gg/druid-team/scroll-minecraft-paper:1.20.1 ./scrolls/minecraft/papermc/1.20.1 -p main=25565 -p rcon=25575 -i artifacts.druid.gg/druid-team/druid:v0.1.249 --min-disk 3Gi --min-ram 512Mi --min-cpu 0.25 --smart --category minecraft
  run druid push artifacts.druid.gg/druid-team/scroll-minecraft-paper:1.20.2 ./scrolls/minecraft/papermc/1.20.2 -p main=25565 -p rcon=25575 -i artifacts.druid.gg/druid-team/druid:v0.1.249 --min-disk 3Gi --min-ram 512Mi --min-cpu 0.25 --smart --category minecraft
  run druid push artifacts.druid.gg/druid-team/scroll-minecraft-paper:1.20.4 ./scrolls/minecraft/papermc/1.20.4 -p main=25565 -p rcon=25575 -i artifacts.druid.gg/druid-team/druid:v0.1.249 --min-disk 3Gi --min-ram 512Mi --min-cpu 0.25 --smart --category minecraft
  run druid push artifacts.druid.gg/druid-team/scroll-minecraft-paper:1.20.6 ./scrolls/minecraft/papermc/1.20.6 -p main=25565 -p rcon=25575 -i artifacts.druid.gg/druid-team/druid:v0.1.249 --min-disk 3Gi --min-ram 512Mi --min-cpu 0.25 --smart --category minecraft
  run druid push artifacts.druid.gg/druid-team/scroll-minecraft-paper:1.21.1 ./scrolls/minecraft/papermc/1.21.1 -p main=25565 -p rcon=25575 -i artifacts.druid.gg/druid-team/druid:v0.1.249 --min-disk 3Gi --min-ram 512Mi --min-cpu 0.25 --smart --category minecraft
  run druid push artifacts.druid.gg/druid-team/scroll-minecraft-paper:1.21.3 ./scrolls/minecraft/papermc/1.21.3 -p main=25565 -p rcon=25575 -i artifacts.druid.gg/druid-team/druid:v0.1.249 --min-disk 3Gi --min-ram 512Mi --min-cpu 0.25 --smart --category minecraft
  run druid push artifacts.druid.gg/druid-team/scroll-minecraft-paper:1.21.4 ./scrolls/minecraft/papermc/1.21.4 -p main=25565 -p rcon=25575 -i artifacts.druid.gg/druid-team/druid:v0.1.249 --min-disk 3Gi --min-ram 512Mi --min-cpu 0.25 --smart --category minecraft
  run druid push artifacts.druid.gg/druid-team/scroll-minecraft-paper:1.21.5 ./scrolls/minecraft/papermc/1.21.5 -p main=25565 -p rcon=25575 -i artifacts.druid.gg/druid-team/druid:v0.1.249 --min-disk 3Gi --min-ram 512Mi --min-cpu 0.25 --smart --category minecraft
  run druid push artifacts.druid.gg/druid-team/scroll-minecraft-paper:1.21.6 ./scrolls/minecraft/papermc/1.21.6 -p main=25565 -p rcon=25575 -i artifacts.druid.gg/druid-team/druid:v0.1.249 --min-disk 3Gi --min-ram 512Mi --min-cpu 0.25 --smart --category minecraft
  run druid push artifacts.druid.gg/druid-team/scroll-minecraft-paper:1.21.7 ./scrolls/minecraft/papermc/1.21.7 -p main=25565 -p rcon=25575 -i artifacts.druid.gg/druid-team/druid:v0.1.249 --min-disk 3Gi --min-ram 512Mi --min-cpu 0.25 --smart --category minecraft
  run druid push artifacts.druid.gg/druid-team/scroll-minecraft-forge:1.17.1 ./scrolls/minecraft/forge/1.17.1 -p main=25565 -p rcon=25575 -i artifacts.druid.gg/druid-team/druid:v0.1.249 --min-disk 3Gi --min-ram 512Mi --min-cpu 0.25 --smart --category minecraft
  run druid push artifacts.druid.gg/druid-team/scroll-minecraft-forge:1.18 ./scrolls/minecraft/forge/1.18 -p main=25565 -p rcon=25575 -i artifacts.druid.gg/druid-team/druid:v0.1.249 --min-disk 3Gi --min-ram 512Mi --min-cpu 0.25 --smart --category minecraft
  run druid push artifacts.druid.gg/druid-team/scroll-minecraft-forge:1.18.1 ./scrolls/minecraft/forge/1.18.1 -p main=25565 -p rcon=25575 -i artifacts.druid.gg/druid-team/druid:v0.1.249 --min-disk 3Gi --min-ram 512Mi --min-cpu 0.25 --smart --category minecraft
  run druid push artifacts.druid.gg/druid-team/scroll-minecraft-forge:1.18.2 ./scrolls/minecraft/forge/1.18.2 -p main=25565 -p rcon=25575 -i artifacts.druid.gg/druid-team/druid:v0.1.249 --min-disk 3Gi --min-ram 512Mi --min-cpu 0.25 --smart --category minecraft
  run druid push artifacts.druid.gg/druid-team/scroll-minecraft-forge:1.19 ./scrolls/minecraft/forge/1.19 -p main=25565 -p rcon=25575 -i artifacts.druid.gg/druid-team/druid:v0.1.249 --min-disk 3Gi --min-ram 512Mi --min-cpu 0.25 --smart --category minecraft
  run druid push artifacts.druid.gg/druid-team/scroll-minecraft-forge:1.19.1 ./scrolls/minecraft/forge/1.19.1 -p main=25565 -p rcon=25575 -i artifacts.druid.gg/druid-team/druid:v0.1.249 --min-disk 3Gi --min-ram 512Mi --min-cpu 0.25 --smart --category minecraft
  run druid push artifacts.druid.gg/druid-team/scroll-minecraft-forge:1.19.2 ./scrolls/minecraft/forge/1.19.2 -p main=25565 -p rcon=25575 -i artifacts.druid.gg/druid-team/druid:v0.1.249 --min-disk 3Gi --min-ram 512Mi --min-cpu 0.25 --smart --category minecraft
  run druid push artifacts.druid.gg/druid-team/scroll-minecraft-forge:1.19.3 ./scrolls/minecraft/forge/1.19.3 -p main=25565 -p rcon=25575 -i artifacts.druid.gg/druid-team/druid:v0.1.249 --min-disk 3Gi --min-ram 512Mi --min-cpu 0.25 --smart --category minecraft
  run druid push artifacts.druid.gg/druid-team/scroll-minecraft-forge:1.19.4 ./scrolls/minecraft/forge/1.19.4 -p main=25565 -p rcon=25575 -i artifacts.druid.gg/druid-team/druid:v0.1.249 --min-disk 3Gi --min-ram 512Mi --min-cpu 0.25 --smart --category minecraft
  run druid push artifacts.druid.gg/druid-team/scroll-minecraft-forge:1.20 ./scrolls/minecraft/forge/1.20 -p main=25565 -p rcon=25575 -i artifacts.druid.gg/druid-team/druid:v0.1.249 --min-disk 3Gi --min-ram 512Mi --min-cpu 0.25 --smart --category minecraft
  run druid push artifacts.druid.gg/druid-team/scroll-minecraft-forge:1.20.1 ./scrolls/minecraft/forge/1.20.1 -p main=25565 -p rcon=25575 -i artifacts.druid.gg/druid-team/druid:v0.1.249 --min-disk 3Gi --min-ram 512Mi --min-cpu 0.25 --smart --category minecraft
  run druid push artifacts.druid.gg/druid-team/scroll-minecraft-forge:1.20.2 ./scrolls/minecraft/forge/1.20.2 -p main=25565 -p rcon=25575 -i artifacts.druid.gg/druid-team/druid:v0.1.249 --min-disk 3Gi --min-ram 512Mi --min-cpu 0.25 --smart --category minecraft
  run druid push artifacts.druid.gg/druid-team/scroll-minecraft-forge:1.20.3 ./scrolls/minecraft/forge/1.20.3 -p main=25565 -p rcon=25575 -i artifacts.druid.gg/druid-team/druid:v0.1.249 --min-disk 3Gi --min-ram 512Mi --min-cpu 0.25 --smart --category minecraft
  run druid push artifacts.druid.gg/druid-team/scroll-minecraft-forge:1.20.4 ./scrolls/minecraft/forge/1.20.4 -p main=25565 -p rcon=25575 -i artifacts.druid.gg/druid-team/druid:v0.1.249 --min-disk 3Gi --min-ram 512Mi --min-cpu 0.25 --smart --category minecraft
  run druid push artifacts.druid.gg/druid-team/scroll-minecraft-forge:1.20.6 ./scrolls/minecraft/forge/1.20.6 -p main=25565 -p rcon=25575 -i artifacts.druid.gg/druid-team/druid:v0.1.249 --min-disk 3Gi --min-ram 512Mi --min-cpu 0.25 --smart --category minecraft
  run druid push artifacts.druid.gg/druid-team/scroll-minecraft-forge:1.21.1 ./scrolls/minecraft/forge/1.21.1 -p main=25565 -p rcon=25575 -i artifacts.druid.gg/druid-team/druid:v0.1.249 --min-disk 3Gi --min-ram 512Mi --min-cpu 0.25 --smart --category minecraft
  run druid push artifacts.druid.gg/druid-team/scroll-minecraft-forge:1.21.3 ./scrolls/minecraft/forge/1.21.3 -p main=25565 -p rcon=25575 -i artifacts.druid.gg/druid-team/druid:v0.1.249 --min-disk 3Gi --min-ram 512Mi --min-cpu 0.25 --smart --category minecraft
  run druid push artifacts.druid.gg/druid-team/scroll-minecraft-forge:1.21.4 ./scrolls/minecraft/forge/1.21.4 -p main=25565 -p rcon=25575 -i artifacts.druid.gg/druid-team/druid:v0.1.249 --min-disk 3Gi --min-ram 512Mi --min-cpu 0.25 --smart --category minecraft
  run druid push artifacts.druid.gg/druid-team/scroll-minecraft-forge:1.21.5 ./scrolls/minecraft/forge/1.21.5 -p main=25565 -p rcon=25575 -i artifacts.druid.gg/druid-team/druid:v0.1.249 --min-disk 3Gi --min-ram 512Mi --min-cpu 0.25 --smart --category minecraft
  run druid push artifacts.druid.gg/druid-team/scroll-minecraft-forge:1.21.6 ./scrolls/minecraft/forge/1.21.6 -p main=25565 -p rcon=25575 -i artifacts.druid.gg/druid-team/druid:v0.1.249 --min-disk 3Gi --min-ram 512Mi --min-cpu 0.25 --smart --category minecraft
  run druid push artifacts.druid.gg/druid-team/scroll-minecraft-forge:1.21.7 ./scrolls/minecraft/forge/1.21.7 -p main=25565 -p rcon=25575 -i artifacts.druid.gg/druid-team/druid:v0.1.249 --min-disk 3Gi --min-ram 512Mi --min-cpu 0.25 --smart --category minecraft
  run druid push artifacts.druid.gg/druid-team/scroll-minecraft-cuberite:latest ./scrolls/minecraft/cuberite/latest -p main=25565 -p webpanel=8080 -i artifacts.druid.gg/druid-team/druid:v0.1.249 --min-disk 3Gi --min-ram 512Mi --min-cpu 0.25 --smart --category minecraft
  run druid push artifacts.druid.gg/druid-team/scroll-rust-oxide:latest ./scrolls/rust/rust-oxide/latest -p main=28015/udp -p query=28017/udp -p rcon=28016 -p rustplus=28082 -i artifacts.druid.gg/druid-team/druid:v0.1.249-steamcmd --min-disk 10Gi --min-ram 6Gi --min-cpu 1 --smart --category rust
  run druid push artifacts.druid.gg/druid-team/scroll-rust-vanilla:latest ./scrolls/rust/rust-vanilla/latest -p main=28015/udp -p query=28017/udp -p rcon=28016 -p rustplus=28082 -i artifacts.druid.gg/druid-team/druid:v0.1.249-steamcmd --min-disk 10Gi --min-ram 6Gi --min-cpu 1 --smart --category rust
  run druid push artifacts.druid.gg/druid-team/scroll-lgsm:pwserver ./scrolls/lgsm/pwserver -p main=8211/udp -p rcon=25575 -i artifacts.druid.gg/druid-team/druid:v0.1.249-steamcmd -m --min-disk 7Gi --min-ram 2Gi --min-cpu 0.5 --smart --category palworld
  run druid push artifacts.druid.gg/druid-team/scroll-lgsm:arkserver ./scrolls/lgsm/arkserver -p main=7777/udp -p query=27015/udp -p rcon=27020 -i artifacts.druid.gg/druid-team/druid:v0.1.249-steamcmd -m --min-disk 25Gi --min-ram 7Gi --min-cpu 0.5 --smart --category ark
  run druid push artifacts.druid.gg/druid-team/scroll-lgsm:untserver ./scrolls/lgsm/untserver -p main=27015/udp -p mainv6=27016 -i artifacts.druid.gg/druid-team/druid:v0.1.249-steamcmd -m  --min-disk 7Gi --min-ram 1Gi --min-cpu 0.5 --smart --category unturned
  run druid push artifacts.druid.gg/druid-team/scroll-lgsm:dayzserver ./scrolls/lgsm/dayzserver -p main=2302/udp -p battle-eye=2304/udp -p query=27016/udp -i artifacts.druid.gg/druid-team/druid:v0.1.249-steamcmd -m --min-disk 7Gi --min-ram 5Gi --min-cpu 1 --category dayz
  run druid push artifacts.druid.gg/druid-team/scroll-lgsm:sdtdserver ./scrolls/lgsm/sdtdserver -p main=26900/udp -p main2=26902/udp -p maintcp=26900 -i artifacts.druid.gg/druid-team/druid:v0.1.249-steamcmd -m --min-disk 20Gi --min-ram 2Gi --min-cpu 0.5 --category 7days
  run druid push artifacts.druid.gg/druid-team/scroll-lgsm:gmodserver ./scrolls/lgsm/gmodserver -p query=27005/udp -p main=27015/udp -p sourcetv=27020/udp -p steam=27015 -i artifacts.druid.gg/druid-team/druid:v0.1.249-steamcmd -m --min-disk 8Gi --min-ram 512Mi --min-cpu 0.25 --smart --category gmod
  run druid push artifacts.druid.gg/druid-team/scroll-lgsm:cs2server ./scrolls/lgsm/cs2server -p main=27015/udp -p rcon=27015 -i artifacts.druid.gg/druid-team/druid:v0.1.249-steamcmd -m --min-disk 38Gi --min-ram 1Gi --min-cpu 0.5 --smart --category cs2
  run druid push artifacts.druid.gg/druid-team/scroll-lgsm:pzserver ./scrolls/lgsm/pzserver -p main=16261/udp -p main2=16262/udp -p maintcp=16261 -i artifacts.druid.gg/druid-team/druid:v0.1.249-steamcmd -m --min-disk 3Gi --min-ram 512Mi --min-cpu 0.25 --smart --category zomboid
  run druid push artifacts.druid.gg/druid-team/scroll-lgsm:csgoserver ./scrolls/lgsm/csgoserver -p query=27005/udp -p main=27015/udp -p sourcetv=27020/udp -p steam=27015 -i artifacts.druid.gg/druid-team/druid:v0.1.249-steamcmd -m --smart --category csgo
  run druid push artifacts.druid.gg/druid-team/scroll-hytale:standalone ./scrolls/hytale/hytale-standalone -p main=5520/udp -i artifacts.druid.gg/druid-team/druid:v0.1.249 --min-disk 10Gi --min-ram 4Gi --min-cpu 1 -m --smart --category hytale
  run druid push artifacts.druid.gg/druid-team/scroll-hytale:latest ./scrolls/hytale/hytale-druid-gg -p main=5520/udp -i artifacts.druid.gg/druid-team/druid:v0.1.249 --min-disk 10Gi --min-ram 4Gi --min-cpu 1 -m --smart --category hytale
}

login_if_configured

if [[ "$SCROLL_PUSH_CATEGORIES" = "1" ]]; then
  push_release_categories
  if ! wait_for_push_phase; then
    echo "One or more category pushes failed." >&2
    exit 1
  fi
fi

if [[ "$SCROLL_PUSH_ARTIFACTS" = "1" ]]; then
  push_release_artifacts
  if ! wait_for_push_phase; then
    echo "One or more artifact pushes failed." >&2
    exit 1
  fi
fi
