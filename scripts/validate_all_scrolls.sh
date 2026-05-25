#!/usr/bin/env bash

set -euo pipefail

if command -v druid >/dev/null 2>&1; then
  while IFS= read -r file; do
    dir="${file%/scroll.yaml}"
    echo "Validating ${dir}"
    druid validate --strict "$dir"
  done < <(find ./scrolls -type f -name scroll.yaml | sort)
else
  go run ./scripts/validate-scrolls.go
fi
