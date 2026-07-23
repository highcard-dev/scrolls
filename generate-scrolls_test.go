package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestParsePortsSupportsDynamicPorts(t *testing.T) {
	ports := ParsePorts("main=/udp;query=0/udp;rcon;fixed=27015;zero=0")
	if len(ports) != 5 {
		t.Fatalf("ports = %#v", ports)
	}
	if ports[0].Name != "main" || ports[0].Port != "" || ports[0].Protocol != "udp" {
		t.Fatalf("main = %#v", ports[0])
	}
	if ports[1].Name != "query" || ports[1].Port != "" || ports[1].Protocol != "udp" {
		t.Fatalf("query = %#v", ports[1])
	}
	if ports[2].Name != "rcon" || ports[2].Port != "" || ports[2].Protocol != "tcp" {
		t.Fatalf("rcon = %#v", ports[2])
	}
	if ports[3].Port != "27015" {
		t.Fatalf("fixed = %#v", ports[3])
	}
	if ports[4].Name != "zero" || ports[4].Port != "" || ports[4].Protocol != "tcp" {
		t.Fatalf("zero = %#v", ports[4])
	}
}

func TestGeneratedARKHandsRCONToRuntimeServer(t *testing.T) {
	scroll := readGeneratedScroll(t, "scrolls/lgsm/arkserver/scroll.yaml")
	console := scroll.Commands["console"]
	for _, procedureID := range []string{"coldstart", "start"} {
		procedure := findGeneratedProcedure(t, console.Procedures, procedureID)
		if !hasGeneratedExpectedPort(procedure.ExpectedPorts, "rcon") {
			t.Fatalf("%s expectedPorts = %#v, want rcon", procedureID, procedure.ExpectedPorts)
		}
	}
}

func TestGeneratedARKDefersRuntimeConfigUntilStart(t *testing.T) {
	scroll := readGeneratedScroll(t, "scrolls/lgsm/arkserver/scroll.yaml")
	for _, procedure := range scroll.Commands["install"].Procedures {
		if slices.Contains(procedure.Command, "configure-runtime.sh") {
			t.Fatalf("install procedure runs runtime-only configuration: %#v", procedure.Command)
		}
	}
}

func TestGeneratedSharedPortsRemainConcrete(t *testing.T) {
	tests := []struct {
		path string
		want map[string]int
	}{
		{
			path: "scrolls/lgsm/cs2server/scroll.yaml",
			want: map[string]int{"main": 27015, "rcon": 27015},
		},
		{
			path: "scrolls/lgsm/pzserver/scroll.yaml",
			want: map[string]int{"main": 16261, "main2": 16262, "maintcp": 16261},
		},
	}
	for _, test := range tests {
		t.Run(test.path, func(t *testing.T) {
			scroll := readGeneratedScroll(t, test.path)
			got := make(map[string]int, len(scroll.Ports))
			for _, port := range scroll.Ports {
				got[port.Name] = port.Port
			}
			for name, want := range test.want {
				if got[name] != want {
					t.Fatalf("port %s = %d, want %d", name, got[name], want)
				}
			}
		})
	}
}

func TestReleasedScrollsHaveTruthfulInstallProgress(t *testing.T) {
	for _, path := range releasedScrollPaths(t) {
		path := path
		t.Run(path, func(t *testing.T) {
			scroll := readGeneratedScroll(t, filepath.Join(path, "scroll.yaml"))
			commands := allGeneratedProcedureCommands(scroll)
			for _, command := range commands {
				joined := strings.Join(command, " ")
				if rawDownloadCommandPattern.MatchString(joined) &&
					!generatedCommandHasPrefix(command, "druid", "progress") {
					t.Fatalf("raw payload download has no structured progress: %q", joined)
				}
			}
			requireCompatibleProgressImages(t, scroll)
			rejectRawDownloadScripts(t, path)

			switch {
			case strings.Contains(path, "/lgsm/"):
				requireGeneratedCommandPrefix(t, commands, "druid", "progress", "steamcmd", "--")
				requireGeneratedProcedureImage(
					t,
					scroll,
					[]string{"sh", "install-lgsm.sh"},
					"artifacts.druid.gg/druid-team/druid:v0.1.258-steamcmd",
				)
			case strings.Contains(path, "/rust/"):
				requireGeneratedCommandPrefix(t, commands, "druid", "progress", "steamcmd", "--")
				if strings.Contains(path, "rust-oxide") {
					requireGeneratedCommandPrefix(t, commands, "druid", "progress", "download")
				}
			case strings.Contains(path, "/minecraft/"):
				requireGeneratedCommandPrefix(t, commands, "druid", "progress", "download")
				if strings.Contains(path, "/forge/") {
					findGeneratedProcedure(t, scroll.Commands["install"].Procedures, "install-forge-server")
				}
			case strings.Contains(path, "/hytale/hytale-druid-gg"):
				requireGeneratedCommandPrefix(t, commands, "druid", "progress", "download")
				findGeneratedProcedure(t, scroll.Commands["install-server"].Procedures, "install-hytale-server")
			case strings.Contains(path, "/hytale/hytale-standalone"):
				installScript, err := os.ReadFile(filepath.Join(path, "data", "install.sh"))
				if err != nil {
					t.Fatal(err)
				}
				if !strings.Contains(string(installScript), "druid progress download") {
					t.Fatal("Hytale standalone HSM download has no structured progress")
				}
				requireGeneratedProcedureImage(
					t,
					scroll,
					[]string{"sh", "install.sh"},
					"artifacts.druid.gg/druid-team/druid:v0.1.258",
				)
				findGeneratedProcedure(t, scroll.Commands["login"].Procedures, "authenticate-hytale")
				findGeneratedProcedure(t, scroll.Commands["update"].Procedures, "install-hytale-server")
			default:
				t.Fatalf("released Scroll has no explicit HIG-24 progress classification: %s", path)
			}
		})
	}
}

func TestPushRetargetsEmbeddedProgressImages(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash is required")
	}

	tests := []struct {
		name          string
		environment   []string
		expectedImage string
	}{
		{
			name: "explicit preview images",
			environment: []string{
				"DRUID_SCROLL_RUNTIME_IMAGE=local.example/druid:test",
				"DRUID_SCROLL_STEAMCMD_IMAGE=local.example/druid:test-steamcmd",
			},
			expectedImage: "local.example/druid:test",
		},
		{
			name: "plain HTTP local seed",
			environment: []string{
				"SCROLL_REGISTRY_HOST=http://druid-gs:8088",
				"DRUID_REGISTRY_PLAIN_HTTP=true",
			},
			expectedImage: "druid:local",
		},
	}

	for index, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fakeDir := filepath.Join(".", fmt.Sprintf(".push-test-%d-%d", os.Getpid(), index))
			if err := os.Mkdir(fakeDir, 0o755); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = os.RemoveAll(fakeDir) })
			fakeDruid := filepath.Join(fakeDir, "druid")
			script := `#!/usr/bin/env bash
set -euo pipefail
if [[ "$1" != "push" || "$2" == "category" ]]; then
  exit 0
fi
source_dir="$3"
scroll="$source_dir/scroll.yaml"
if grep -Fq 'artifacts.druid.gg/druid-team/druid:v0.1.258' "$scroll"; then
  echo "pinned progress image leaked into packaged Scroll: $scroll" >&2
  exit 41
fi
if ! grep -Fq "$EXPECTED_PROGRESS_IMAGE" "$scroll"; then
  echo "retargeted progress image missing from packaged Scroll: $scroll" >&2
  exit 42
fi
`
			if err := os.WriteFile(fakeDruid, []byte(script), 0o755); err != nil {
				t.Fatal(err)
			}

			command := exec.Command("bash", "scripts/push.sh")
			command.Env = append(
				os.Environ(),
				"DRUID_BIN="+filepath.ToSlash(fakeDruid),
				"SCROLL_PUSH_CATEGORIES=0",
				"SCROLL_PUSH_ARTIFACTS=1",
				"EXPECTED_PROGRESS_IMAGE="+test.expectedImage,
			)
			command.Env = append(command.Env, test.environment...)
			output, err := command.CombinedOutput()
			if err != nil {
				t.Fatalf("push.sh failed: %v\n%s", err, output)
			}
		})
	}
}

func TestWorkflowsUseProgressCapableCLIVersion(t *testing.T) {
	for _, path := range []string{".github/workflows/pr.yml", ".github/workflows/release.yml"} {
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(content), "DRUID_CLI_VERSION: v0.1.258") {
			t.Fatalf("%s does not use the progress-capable CLI release", path)
		}
	}
}

type generatedScroll struct {
	Ports    []generatedPort             `yaml:"ports"`
	Commands map[string]generatedCommand `yaml:"commands"`
}

type generatedPort struct {
	Name string `yaml:"name"`
	Port int    `yaml:"port"`
}

type generatedCommand struct {
	Procedures []generatedProcedure `yaml:"procedures"`
}

type generatedProcedure struct {
	ID            string                  `yaml:"id"`
	Image         string                  `yaml:"image"`
	ExpectedPorts []generatedExpectedPort `yaml:"expectedPorts"`
	Command       []string                `yaml:"command"`
}

type generatedExpectedPort struct {
	Name string `yaml:"name"`
}

func readGeneratedScroll(t *testing.T, path string) generatedScroll {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var scroll generatedScroll
	if err := yaml.Unmarshal(content, &scroll); err != nil {
		t.Fatal(err)
	}
	return scroll
}

func findGeneratedProcedure(t *testing.T, procedures []generatedProcedure, id string) generatedProcedure {
	t.Helper()
	for _, procedure := range procedures {
		if procedure.ID == id {
			return procedure
		}
	}
	t.Fatalf("procedure %q not found", id)
	return generatedProcedure{}
}

func hasGeneratedExpectedPort(ports []generatedExpectedPort, name string) bool {
	for _, port := range ports {
		if port.Name == name {
			return true
		}
	}
	return false
}

func releasedScrollPaths(t *testing.T) []string {
	t.Helper()
	content, err := os.ReadFile("scripts/push.sh")
	if err != nil {
		t.Fatal(err)
	}
	pattern := regexp.MustCompile(`(?m)^\s*run druid push \S+\s+(\./scrolls/\S+)`)
	matches := pattern.FindAllStringSubmatch(string(content), -1)
	seen := map[string]bool{}
	paths := make([]string, 0, len(matches))
	for _, match := range matches {
		path := strings.TrimPrefix(match[1], "./")
		if !seen[path] {
			seen[path] = true
			paths = append(paths, path)
		}
	}
	if len(paths) == 0 {
		t.Fatal("release catalog did not contain any Scroll paths")
	}
	return paths
}

func allGeneratedProcedureCommands(scroll generatedScroll) [][]string {
	commands := make([][]string, 0)
	for _, command := range scroll.Commands {
		for _, procedure := range command.Procedures {
			commands = append(commands, procedure.Command)
		}
	}
	return commands
}

func requireGeneratedCommandPrefix(t *testing.T, commands [][]string, prefix ...string) {
	t.Helper()
	for _, command := range commands {
		if generatedCommandHasPrefix(command, prefix...) {
			return
		}
	}
	t.Fatalf("no procedure command starts with %q", strings.Join(prefix, " "))
}

func generatedCommandHasPrefix(command []string, prefix ...string) bool {
	return len(command) >= len(prefix) && slices.Equal(command[:len(prefix)], prefix)
}

func requireCompatibleProgressImages(t *testing.T, scroll generatedScroll) {
	t.Helper()
	for _, command := range scroll.Commands {
		for _, procedure := range command.Procedures {
			if !generatedCommandHasPrefix(procedure.Command, "druid", "progress") {
				continue
			}
			expected := "artifacts.druid.gg/druid-team/druid:v0.1.258"
			if len(procedure.Command) > 2 && procedure.Command[2] == "steamcmd" {
				expected += "-steamcmd"
			}
			if procedure.Image != expected {
				t.Fatalf("progress command %q uses incompatible image %q; want %q",
					strings.Join(procedure.Command, " "), procedure.Image, expected)
			}
		}
	}
}

func requireGeneratedProcedureImage(
	t *testing.T,
	scroll generatedScroll,
	command []string,
	expectedImage string,
) {
	t.Helper()
	for _, scrollCommand := range scroll.Commands {
		for _, procedure := range scrollCommand.Procedures {
			if slices.Equal(procedure.Command, command) {
				if procedure.Image != expectedImage {
					t.Fatalf("command %q uses incompatible image %q; want %q",
						strings.Join(command, " "), procedure.Image, expectedImage)
				}
				return
			}
		}
	}
	t.Fatalf("procedure command %q not found", strings.Join(command, " "))
}

var rawDownloadCommandPattern = regexp.MustCompile(`(^|[[:space:];|&])(wget|curl|steamcmd)([[:space:];|&]|$)`)

func rejectRawDownloadScripts(t *testing.T, root string) {
	t.Helper()
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || filepath.Ext(path) != ".sh" {
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if rawDownloadCommandPattern.Match(content) {
			t.Fatalf("raw payload download in published script %s has no structured progress", path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
