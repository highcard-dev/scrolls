#!/usr/bin/env bash

set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
push_script="$repo_root/scripts/push.sh"
fake_druid_source="$repo_root/scripts/tests/fixtures/fake-druid.sh"
tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT
fake_druid="$tmp_dir/fake-druid"
cp "$fake_druid_source" "$fake_druid"
chmod +x "$fake_druid"

fail() {
  echo "FAIL: $*" >&2
  exit 1
}

assert_equals() {
  local expected="$1"
  local actual="$2"
  local message="$3"
  [[ "$actual" = "$expected" ]] || fail "$message (expected $expected, got $actual)"
}

reset_state() {
  rm -rf "$tmp_dir/state"
  mkdir -p "$tmp_dir/state"
}

category_dry_run="$(
  SCROLL_PUSH_DRY_RUN=1 \
  SCROLL_PUSH_CATEGORIES=1 \
  SCROLL_PUSH_ARTIFACTS=0 \
    bash "$push_script"
)"
artifact_dry_run="$(
  SCROLL_PUSH_DRY_RUN=1 \
  SCROLL_PUSH_CATEGORIES=0 \
  SCROLL_PUSH_ARTIFACTS=1 \
    bash "$push_script"
)"
[[ "$artifact_dry_run" == *"artifacts.druid.gg/druid-team/scroll-minecraft-spigot:1.17"* ]] || \
  fail "release pushes must update the existing repositories and tags"
[[ "$artifact_dry_run" == *"artifacts.druid.gg/druid-team/scroll-minecraft-forge:26.3 ./scrolls/minecraft/forge/26.3"* ]] || \
  fail "release pushes must include the new Forge 26.3 Scroll"
[[ "$artifact_dry_run" == *"-i artifacts.druid.gg/druid-team/druid:v0.1.281"* ]] || \
  fail "release pushes must preserve runtime image references"
[[ "$category_dry_run" == *"artifacts.druid.gg/druid-team/scroll-minecraft-spigot"* ]] || \
  fail "category metadata must retain the configured registry namespace"
expected_categories="$(printf '%s\n' "$category_dry_run" | grep -Ec '^"?druid"? push category ')"
expected_artifacts="$(printf '%s\n' "$artifact_dry_run" | grep -Ec '^"?druid"? push ')"

preview_dry_run="$(
  SCROLL_PUSH_DRY_RUN=1 \
  SCROLL_PUSH_CATEGORIES=0 \
  SCROLL_REGISTRY_NAMESPACE=druid-team-experimental \
  SCROLL_REGISTRY_RUNTIME_NAMESPACE=druid-team \
  SCROLL_TAG_SUFFIX=-pr123 \
    bash "$push_script"
)"
[[ "$preview_dry_run" == *"artifacts.druid.gg/druid-team-experimental/scroll-minecraft-spigot:1.17-pr123"* ]] || \
  fail "preview pushes must use the preview namespace and tag suffix"
[[ "$preview_dry_run" == *"-i artifacts.druid.gg/druid-team/druid:v0.1.281"* ]] || \
  fail "preview pushes must leave runtime images in the release namespace"
[[ "$preview_dry_run" == *"-i artifacts.druid.gg/druid-team/druid:v0.1.281-steamcmd"* ]] || \
  fail "preview pushes must preserve the SteamCMD runtime image"

serial_dry_run="$(SCROLL_PUSH_DRY_RUN=1 SCROLL_PUSH_JOBS=1 bash "$push_script")"
parallel_dry_run="$(SCROLL_PUSH_DRY_RUN=1 SCROLL_PUSH_JOBS=4 bash "$push_script")"
assert_equals "$serial_dry_run" "$parallel_dry_run" \
  "dry-run catalog output must stay deterministic across job counts"

reset_state
if SCROLL_PUSH_JOBS=0 \
  DRUID_BIN="$fake_druid" \
  FAKE_DRUID_STATE_DIR="$tmp_dir/state" \
    bash "$push_script" >"$tmp_dir/invalid.out" 2>"$tmp_dir/invalid.err"; then
  fail "SCROLL_PUSH_JOBS=0 must be rejected"
fi
grep -q 'SCROLL_PUSH_JOBS must be a positive integer' "$tmp_dir/invalid.err" || \
  fail "invalid job count must report a useful validation error"
[[ ! -f "$tmp_dir/state/events.log" ]] || \
  fail "invalid job count must be rejected before invoking druid"

reset_state
(
  unset SCROLL_PUSH_JOBS
  DRUID_BIN="$fake_druid" \
  FAKE_DRUID_STATE_DIR="$tmp_dir/state" \
  FAKE_DRUID_SLEEP=0.01 \
  SCROLL_PUSH_CATEGORIES=1 \
  SCROLL_PUSH_ARTIFACTS=0 \
    bash "$push_script" >"$tmp_dir/default.out"
)
assert_equals "1" "$(<"$tmp_dir/state/max")" \
  "the default must preserve serial push behavior"

reset_state
DRUID_BIN="$fake_druid" \
FAKE_DRUID_STATE_DIR="$tmp_dir/state" \
FAKE_DRUID_SLEEP=0 \
SCROLL_REGISTRY_USER=test-publisher \
SCROLL_REGISTRY_PASSWORD=test-password \
SCROLL_PUSH_ARTIFACTS=0 \
  bash "$push_script" >"$tmp_dir/login.out"
grep -q '^start artifact login --host artifacts.druid.gg --user test-publisher --password test-password$' "$tmp_dir/state/events.log" || \
  fail "configured registry credentials must be passed to druid login"
assert_equals "start artifact login" "$(head -n 1 "$tmp_dir/state/events.log" | cut -d ' ' -f1-3)" \
  "registry login must happen before any push"

reset_state
if DRUID_BIN="$fake_druid" \
  FAKE_DRUID_STATE_DIR="$tmp_dir/state" \
  FAKE_DRUID_SLEEP=0 \
  FAKE_DRUID_FAIL_REF=--host \
  SCROLL_REGISTRY_USER=test-publisher \
  SCROLL_REGISTRY_PASSWORD=test-password \
  SCROLL_PUSH_ARTIFACTS=0 \
    bash "$push_script" >"$tmp_dir/login-failure.out" 2>"$tmp_dir/login-failure.err"; then
  fail "a failed registry login must make push.sh fail"
fi
if grep -q '^start .* push ' "$tmp_dir/state/events.log"; then
  fail "no artifacts or categories may be pushed after a failed login"
fi

reset_state
DRUID_BIN="$fake_druid" \
FAKE_DRUID_STATE_DIR="$tmp_dir/state" \
FAKE_DRUID_SLEEP=0.02 \
SCROLL_PUSH_JOBS=3 \
  bash "$push_script" >"$tmp_dir/parallel.out"

maximum="$(<"$tmp_dir/state/max")"
((maximum >= 2)) || fail "SCROLL_PUSH_JOBS=3 must actually overlap push jobs"
((maximum <= 3)) || fail "parallel pushes exceeded the configured job bound (max=$maximum)"

actual_categories="$(grep -c '^start category ' "$tmp_dir/state/events.log")"
actual_artifacts="$(grep -c '^start artifact ' "$tmp_dir/state/events.log")"
assert_equals "$expected_categories" "$actual_categories" \
  "parallel mode must invoke every category push exactly once"
assert_equals "$expected_artifacts" "$actual_artifacts" \
  "parallel mode must invoke every artifact push exactly once"
last_category_end="$(grep -n '^end category ' "$tmp_dir/state/events.log" | tail -n 1 | cut -d: -f1)"
first_artifact_start="$(grep -m 1 -n '^start artifact ' "$tmp_dir/state/events.log" | cut -d: -f1)"
((last_category_end < first_artifact_start)) || \
  fail "artifact pushes started before all category pushes completed"

reset_state
if DRUID_BIN="$fake_druid" \
  FAKE_DRUID_STATE_DIR="$tmp_dir/state" \
  FAKE_DRUID_SLEEP=0.01 \
  FAKE_DRUID_FAIL_REF='artifacts.druid.gg/druid-team/scroll-minecraft-spigot:1.17' \
  SCROLL_PUSH_JOBS=4 \
  SCROLL_PUSH_CATEGORIES=0 \
  SCROLL_PUSH_ARTIFACTS=1 \
    bash "$push_script" >"$tmp_dir/failure.out" 2>"$tmp_dir/failure.err"; then
  fail "a failed parallel push must make push.sh fail"
fi

failure_starts="$(grep -c '^start artifact ' "$tmp_dir/state/events.log")"
failure_ends="$(grep -c '^end artifact ' "$tmp_dir/state/events.log")"
assert_equals "$expected_artifacts" "$failure_starts" \
  "a job failure must not abandon the remaining queued artifact pushes"
assert_equals "$failure_starts" "$failure_ends" \
  "push.sh must wait for every started job before returning a failure"
assert_equals "0" "$(<"$tmp_dir/state/active")" \
  "push.sh must not leave background push jobs running after failure"

echo "push parallelism tests passed ($expected_categories categories, $expected_artifacts artifacts)"
