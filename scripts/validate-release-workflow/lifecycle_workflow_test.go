package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

func TestLifecycleWorkflows(t *testing.T) {
	if err := validateLifecycleWorkflows("../../.github/workflows"); err != nil {
		t.Fatal(err)
	}
}

func TestLifecycleProvisioningGate(t *testing.T) {
	data, err := os.ReadFile("../../.github/workflows/scroll-lifecycle.yml")
	if err != nil {
		t.Fatal(err)
	}
	var workflow lifecycleWorkflow
	if err := yaml.Unmarshal(data, &workflow); err != nil {
		t.Fatal(err)
	}
	steps := workflow.Jobs["publish"].Steps
	if len(steps) == 0 || steps[0].Name != "Require provisioned lifecycle environment" {
		t.Fatal("provisioning gate must run before checkout or any credential-bearing step")
	}
	for _, value := range []string{"", "false", "TRUE", "true"} {
		t.Run("ready="+value, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			command := exec.CommandContext(ctx, "bash", "-eo", "pipefail", "-c", steps[0].Run)
			command.Env = []string{"PATH=" + os.Getenv("PATH")}
			if value != "" {
				command.Env = append(command.Env, "LIFECYCLE_READY="+value)
			}
			output, err := command.CombinedOutput()
			if ctx.Err() != nil {
				t.Fatal(ctx.Err())
			}
			if value == "true" {
				if err != nil {
					t.Fatalf("provisioned environment rejected: %v: %s", err, output)
				}
			} else if exit, ok := err.(*exec.ExitError); !ok || exit.ExitCode() != 1 {
				t.Fatalf("unprovisioned environment must exit 1: %v: %s", err, output)
			}
		})
	}
}

func TestLifecycleWorkflowRejectsMissingCutoverGuards(t *testing.T) {
	for name, change := range map[string][3]string{
		"legacy caller":             {"release.yml", "./.github/workflows/scroll-lifecycle.yml", "./.github/workflows/legacy.yml"},
		"public preview":            {"pr.yml", "public: false", "public: true"},
		"direct fallback":           {"scroll-lifecycle.yml", "SCROLL_PUBLISH_MODE: lifecycle", "SCROLL_PUBLISH_MODE: direct"},
		"mutable revision":          {"scroll-lifecycle.yml", "SCROLL_TAG_SUFFIX: -${{ github.event.pull_request.head.sha || github.sha }}", "SCROLL_TAG_SUFFIX: -latest"},
		"public preview repository": {"scroll-lifecycle.yml", "format('-pr{0}', github.event.pull_request.number) || ''", "''"},
		"admin credential fallback": {"scroll-lifecycle.yml", "secrets.SCROLL_TEAM_REGISTRY_USER", "secrets.SCROLL_REGISTRY_USER"},
		"fork publication":          {"scroll-lifecycle.yml", "github.event.pull_request.head.repo.full_name == github.repository", "true"},
		"no review environment":     {"scroll-lifecycle.yml", "environment: ${{ inputs.public && 'scroll-release' || 'scroll-preview' }}", "environment: production"},
		"mutable CLI":               {"scroll-lifecycle.yml", "a9f5dd9ec361ee5b268dfa5b5c2a56ef60f350fd", "master"},
		"cancel mutation":           {"scroll-lifecycle.yml", "cancel-in-progress: false", "cancel-in-progress: true"},
		"skip publisher tests":      {"scroll-lifecycle.yml", "go test ./scripts/publish-lifecycle", "go test ./scripts/prebuild"},
		"late provisioning gate":    {"scroll-lifecycle.yml", "      - name: Require provisioned lifecycle environment", "      - run: echo premature-step\n      - name: Require provisioned lifecycle environment"},
	} {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			for _, file := range []string{"release.yml", "pr.yml", "scroll-lifecycle.yml"} {
				data, err := os.ReadFile(filepath.Join("../../.github/workflows", file))
				if err != nil {
					t.Fatal(err)
				}
				if file == change[0] {
					if !strings.Contains(string(data), change[1]) {
						t.Fatal("mutation did not match workflow")
					}
					data = []byte(strings.Replace(string(data), change[1], change[2], 1))
				}
				if err := os.WriteFile(filepath.Join(dir, file), data, 0600); err != nil {
					t.Fatal(err)
				}
			}
			if err := validateLifecycleWorkflows(dir); err == nil {
				t.Fatal("unsafe workflow contract accepted")
			}
		})
	}
}
