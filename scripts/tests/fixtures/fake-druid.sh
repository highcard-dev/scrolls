#!/usr/bin/env bash

set -u

state_dir="${FAKE_DRUID_STATE_DIR:?FAKE_DRUID_STATE_DIR is required}"
sleep_seconds="${FAKE_DRUID_SLEEP:-0.02}"
lock_dir="$state_dir/.lock"

acquire_lock() {
  while ! mkdir "$lock_dir" 2>/dev/null; do
    sleep 0.005
  done
}

release_lock() {
  rmdir "$lock_dir"
}

kind="artifact"
ref="${2:-}"
if [[ "${1:-}" = "push" && "${2:-}" = "category" ]]; then
  kind="category"
  ref="${3:-}"
fi

acquire_lock
active=0
maximum=0
[[ -f "$state_dir/active" ]] && read -r active < "$state_dir/active"
[[ -f "$state_dir/max" ]] && read -r maximum < "$state_dir/max"
active=$((active + 1))
if ((active > maximum)); then
  maximum="$active"
fi
printf '%s\n' "$active" > "$state_dir/active"
printf '%s\n' "$maximum" > "$state_dir/max"
printf 'start %s %s\n' "$kind" "$*" >> "$state_dir/events.log"
release_lock

sleep "$sleep_seconds"

status=0
if [[ -n "${FAKE_DRUID_FAIL_REF:-}" && "$ref" = "$FAKE_DRUID_FAIL_REF" ]]; then
  status=23
fi

acquire_lock
read -r active < "$state_dir/active"
active=$((active - 1))
printf '%s\n' "$active" > "$state_dir/active"
printf 'end %s %s status=%s\n' "$kind" "$*" "$status" >> "$state_dir/events.log"
release_lock

exit "$status"
