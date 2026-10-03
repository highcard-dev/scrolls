package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// Exercise the real catalog/pool with an offline publisher boundary. Publisher
// HTTP/auth behavior is covered separately in main_test.go; no registry is used.
func TestPushScriptUsesLifecycleForEveryArtifact(t *testing.T) {
	for _, jobs := range []string{"1", "3"} {
		t.Run("jobs-"+jobs, func(t *testing.T) {
			state, run := lifecycleScriptFixture(t, jobs)
			if output, err := run(); err != nil {
				t.Fatalf("push script: %v\n%s", err, output)
			}
			calls := readFixture(t, filepath.Join(state, "publisher.log"))
			events := readFixture(t, filepath.Join(state, "events.log"))
			starts, active, peak := 0, 0, 0
			for _, line := range strings.Split(events, "\n") {
				if strings.HasPrefix(line, "start artifact ") {
					starts++
					active++
					if active > peak {
						peak = active
					}
				}
				if strings.HasPrefix(line, "end artifact ") {
					active--
				}
			}
			expected, categories := catalogCounts(t)
			if starts == 0 || starts != expected || len(strings.Split(strings.TrimSpace(calls), "\n")) != starts {
				t.Fatalf("not every catalog artifact used the lifecycle publisher: %d starts\n%s", starts, calls)
			}
			for _, line := range strings.Split(strings.TrimSpace(calls), "\n") {
				fields := strings.Fields(line)
				if len(fields) < 2 || fields[0] != "1700000000" || !strings.HasPrefix(fields[1], "registry.invalid/fixture-owner/") || !strings.HasSuffix(fields[1], "-commit") {
					t.Fatalf("unstable timestamp or wrong staging reference: %s", line)
				}
				if strings.Contains(line, " -i ") && !strings.Contains(line, " -i registry.invalid/druid-team/druid:") {
					t.Fatalf("runtime image was redirected into private staging: %s", line)
				}
			}
			if strings.Count(events, "start category ") != categories || strings.Count(events, "end category ") != categories ||
				strings.LastIndex(events, "end category ") < 0 || strings.LastIndex(events, "end category ") > strings.Index(events, "start artifact ") {
				t.Fatal("lifecycle artifacts started before category completion")
			}
			if readFixture(t, filepath.Join(state, "active")) != "0\n" {
				t.Fatal("publisher children remain active")
			}
			limit, _ := strconv.Atoi(jobs)
			if active != 0 || peak < 1 || peak > limit || (limit > 1 && peak < 2) {
				t.Fatalf("lifecycle artifact concurrency = %d, configured = %d, active = %d", peak, limit, active)
			}
		})
	}
}

func TestPushScriptFailsBeforeMutationWhenLifecyclePreflightFails(t *testing.T) {
	state, run := lifecycleScriptFixture(t, "3", "FAKE_PREFLIGHT_FAIL=1")
	if output, err := run(); err == nil {
		t.Fatalf("failed preflight accepted: %s", output)
	}
	for _, name := range []string{"events.log", "publisher.log", "login"} {
		if _, err := os.Stat(filepath.Join(state, name)); !os.IsNotExist(err) {
			t.Fatalf("failed preflight reached %s: %v", name, err)
		}
	}
}

func TestPushScriptReapsLifecycleFailures(t *testing.T) {
	state, run := lifecycleScriptFixture(t, "3", "FAKE_DRUID_FAIL_REF=registry.invalid/fixture-owner/scroll-minecraft-spigot:1.17-commit")
	if output, err := run(); err == nil {
		t.Fatalf("failed lifecycle artifact accepted: %s", output)
	}
	events := readFixture(t, filepath.Join(state, "events.log"))
	expected, _ := catalogCounts(t)
	if strings.Count(events, "start artifact ") != expected {
		t.Fatal("lifecycle failure abandoned unstarted catalog artifacts")
	}
	if strings.Count(events, "start artifact ") != strings.Count(events, "end artifact ") || readFixture(t, filepath.Join(state, "active")) != "0\n" {
		t.Fatal("lifecycle failure abandoned active publisher jobs")
	}
	if !strings.Contains(events, "status=23") {
		t.Fatal("fixture did not exercise the publisher failure")
	}
}

func lifecycleScriptFixture(t *testing.T, jobs string, extra ...string) (string, func() ([]byte, error)) {
	t.Helper()
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	state := t.TempDir()
	bin := filepath.Join(state, "bin")
	if err := os.Mkdir(bin, 0700); err != nil {
		t.Fatal(err)
	}
	stubs := map[string]string{
		"go": `#!/usr/bin/env bash
set -euo pipefail
[[ "$1" == run && "$2" == ./scripts/publish-lifecycle ]] || exit 91
if [[ "$3" == preflight ]]; then
  [[ "${FAKE_PREFLIGHT_FAIL:-0}" != 1 ]] || exit 92
  touch "$FAKE_DRUID_STATE_DIR/preflight"
  exit 0
fi
[[ -f "$FAKE_DRUID_STATE_DIR/preflight" && "$3" == -- && "$4" == "$DRUID_BIN" && "$5" == push && "$6" != category ]] || exit 93
printf '%s %s %s\n' "$SOURCE_DATE_EPOCH" "$6" "$*" >> "$FAKE_DRUID_STATE_DIR/publisher.log"
shift 4
exec bash "$FAKE_DRUID_SOURCE" "$@"
`,
		"druid": `#!/usr/bin/env bash
set -euo pipefail
[[ -f "$FAKE_DRUID_STATE_DIR/preflight" ]] || exit 94
if [[ "$1" == login ]]; then
  touch "$FAKE_DRUID_STATE_DIR/login"
  exit 0
fi
[[ "$1" == push && "$2" == category ]] || exit 95
exec bash "$FAKE_DRUID_SOURCE" "$@"
`,
	}
	for name, script := range stubs {
		if err := os.WriteFile(filepath.Join(bin, name), []byte(script), 0700); err != nil {
			t.Fatal(err)
		}
	}
	return state, func() ([]byte, error) {
		cmd := exec.Command("bash", filepath.Join(root, "scripts/push.sh"))
		cmd.Dir = root
		cmd.Env = append(os.Environ(), "PATH="+bin+":"+os.Getenv("PATH"), "DRUID_BIN="+filepath.Join(bin, "druid"),
			"FAKE_DRUID_SOURCE="+filepath.Join(root, "scripts/tests/fixtures/fake-druid.sh"), "FAKE_DRUID_STATE_DIR="+state,
			"FAKE_DRUID_SLEEP=0.001", "FAKE_DRUID_FAIL_REF=", "FAKE_PREFLIGHT_FAIL=0", "SCROLL_PUBLISH_MODE=lifecycle",
			"SCROLL_PUSH_DRY_RUN=0", "SCROLL_PUSH_UI=0", "SCROLL_PUSH_CATEGORIES=1", "SCROLL_PUSH_ARTIFACTS=1",
			"SCROLL_PUSH_JOBS="+jobs, "SCROLL_REGISTRY_HOST=registry.invalid", "SCROLL_REGISTRY_NAMESPACE=fixture-owner",
			"SCROLL_REGISTRY_RUNTIME_NAMESPACE=druid-team", "SCROLL_TAG_SUFFIX=-commit", "SOURCE_DATE_EPOCH=1700000000",
			"SCROLL_REGISTRY_USER=fixture", "SCROLL_REGISTRY_PASSWORD=fixture", "DRUID_SCROLL_RUNTIME_IMAGE=", "DRUID_SCROLL_STEAMCMD_IMAGE=")
		cmd.Env = append(cmd.Env, extra...)
		return cmd.CombinedOutput()
	}
}

func readFixture(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func catalogCounts(t *testing.T) (artifacts, categories int) {
	t.Helper()
	for _, line := range strings.Split(readFixture(t, "../push.sh"), "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "run druid push ") {
			if strings.Contains(line, "push category ") {
				categories++
			} else {
				artifacts++
			}
		}
	}
	if artifacts == 0 || categories == 0 {
		t.Fatal("fixture could not read the explicit catalog")
	}
	return
}
