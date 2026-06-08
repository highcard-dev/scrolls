#!/usr/bin/env bash

set -euo pipefail

pr_number="${1:?usage: $0 <pr-number>}"
registry_host="${SCROLL_REGISTRY_HOST:?SCROLL_REGISTRY_HOST is required}"
registry_host="${registry_host#http://}"
registry_host="${registry_host#https://}"
registry_host="${registry_host%%/*}"

namespace="${SCROLL_REGISTRY_PR_NAMESPACE:-druid-team-experimental}"
runtime_namespace="${SCROLL_REGISTRY_RUNTIME_NAMESPACE:-druid-team}"
runtime_image="${DRUID_SCROLL_RUNTIME_IMAGE:-${registry_host}/${runtime_namespace}/druid:v0.1.249}"
roots="${SCROLL_PR_ROOTS:-}"

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

scroll_dirs() {
  if [[ -n "$roots" ]]; then
    for root in $roots; do
      for dir in "$root"/*; do
        [[ -f "$dir/scroll.yaml" ]] || continue
        printf '%s\n' "$dir"
      done
      [[ -f "$root/scroll.yaml" ]] && printf '%s\n' "$root"
    done
    return
  fi
  find ./scrolls -type f -name scroll.yaml -print | sed 's#/scroll.yaml$##' | sort
}

port_args() {
  awk '
    /^ports:/ { in_ports=1; next }
    /^[^[:space:]-]/ { in_ports=0 }
    in_ports && /^[[:space:]]*-[[:space:]]*name:/ {
      name=$0
      sub(/^[[:space:]]*-[[:space:]]*name:[[:space:]]*/, "", name)
      gsub(/"/, "", name)
    }
    in_ports && /^[[:space:]]*port:/ {
      port=$0
      sub(/^[[:space:]]*port:[[:space:]]*/, "", port)
      if (name != "" && port != "") {
        printf " -p %s=%s", name, port
      }
    }
  ' "$1/scroll.yaml"
}

category_for_dir() {
  local dir="$1"
  dir="${dir#./scrolls/}"
  echo "${dir%%/*}"
}

push_dir() {
  local dir="$1"
  local name app_version image artifact ports category

  if [[ "$dir" == ./scrolls/.sample || "$dir" == ./scrolls/.sample/* ]]; then
    echo "Skipping ${dir}: sample scrolls are validation fixtures"
    return
  fi

  name="$(yaml_value "$dir/scroll.yaml" name)"
  app_version="$(yaml_value "$dir/scroll.yaml" app_version)"
  if [[ "$app_version" == "latest" ]]; then
    echo "Skipping ${dir}: PR previews do not publish latest tags"
    return
  fi

  image="${name##*/}"
  artifact="${registry_host}/${namespace}/${image}:${app_version}-pr${pr_number}"
  ports="$(port_args "$dir")"
  category="$(category_for_dir "$dir")"

  echo "Pushing ${artifact} from ${dir}"
  # shellcheck disable=SC2086
  druid push "$artifact" "$dir" $ports \
    -i "$runtime_image" \
    --min-disk 3Gi --min-ram 512Mi --min-cpu 0.25 \
    --smart --category "$category"
}

while IFS= read -r dir; do
  push_dir "$dir"
done < <(scroll_dirs)
