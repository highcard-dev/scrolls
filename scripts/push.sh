#!/usr/bin/env bash

set -euo pipefail

pr_number="${1:?usage: $0 <pr-number>}"
registry_host="${SCROLL_REGISTRY_HOST:?SCROLL_REGISTRY_HOST is required}"
registry_host="${registry_host#http://}"
registry_host="${registry_host#https://}"
registry_host="${registry_host%%/*}"

namespace="${SCROLL_REGISTRY_PR_NAMESPACE:-druid-team-experimental}"
runtime_namespace="${SCROLL_REGISTRY_RUNTIME_NAMESPACE:-druid-team}"
runtime_image="${DRUID_SCROLL_RUNTIME_IMAGE:-${registry_host}/${runtime_namespace}/druid:stable-nix}"
roots="${SCROLL_PR_ROOTS:-./scrolls/minecraft/minecraft-vanilla}"

yaml_value() {
  awk -F':[[:space:]]*' -v key="$2" '
    $1 == key {
      value = $2
      gsub(/^"/, "", value)
      gsub(/"$/, "", value)
      print value
      exit
    }
  ' "$1"
}

push_dir() {
  local dir="$1"
  local name app_version image artifact

  name="$(yaml_value "$dir/scroll.yaml" name)"
  app_version="$(yaml_value "$dir/scroll.yaml" app_version)"
  if [[ "$app_version" == "latest" ]]; then
    echo "Skipping ${dir}: PR previews do not publish latest tags"
    return
  fi

  image="${name##*/}"
  artifact="${registry_host}/${namespace}/${image}:${app_version}-pr${pr_number}"

  echo "Pushing ${artifact} from ${dir}"
  druid push "$artifact" "$dir" \
    -p main=25565 -p rcon=25575 \
    -i "$runtime_image" \
    --min-disk 3Gi --min-ram 512Mi --min-cpu 0.25 \
    --smart --category minecraft
}

for root in $roots; do
  for dir in "$root"/*; do
    [[ -f "$dir/scroll.yaml" ]] || continue
    push_dir "$dir"
  done
done
