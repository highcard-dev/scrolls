package main

import (
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestPrebuildCommandLeavesScriptsUntouched(t *testing.T) {
	entrypoint, args := prebuildCommand([]string{"bash", "postinstall.sh"})
	if entrypoint != "bash" {
		t.Fatalf("entrypoint = %q, want bash", entrypoint)
	}
	if len(args) != 1 || args[0] != "postinstall.sh" {
		t.Fatalf("args = %#v", args)
	}
}

func TestPrebuildCommandLeavesRegularCommandsUntouched(t *testing.T) {
	entrypoint, args := prebuildCommand([]string{"./cs2server", "auto-install"})
	if entrypoint != "./cs2server" {
		t.Fatalf("entrypoint = %q, want ./cs2server", entrypoint)
	}
	if len(args) != 1 || args[0] != "auto-install" {
		t.Fatalf("args = %#v", args)
	}
}

func TestPrebuildCommandLeavesShellFlagsUntouched(t *testing.T) {
	entrypoint, args := prebuildCommand([]string{"sh", "-c", "echo ok"})
	if entrypoint != "sh" {
		t.Fatalf("entrypoint = %q, want sh", entrypoint)
	}
	if len(args) != 2 || args[0] != "-c" || args[1] != "echo ok" {
		t.Fatalf("args = %#v", args)
	}
}

func TestSelectSpecsSupportsCommaSeparatedTargets(t *testing.T) {
	specs, err := selectSpecs("arkserver,rust-oxide")
	if err != nil {
		t.Fatal(err)
	}
	if len(specs) != 2 {
		t.Fatalf("specs = %#v, want two specs", specs)
	}
	if specs[0].Target != "arkserver" || specs[1].Target != "rust-oxide" {
		t.Fatalf("targets = %#v, want arkserver and rust-oxide", []string{specs[0].Target, specs[1].Target})
	}
}

func TestSelectSpecsAllSteamReturnsIndependentTargets(t *testing.T) {
	specs, err := selectSpecs("all-steam")
	if err != nil {
		t.Fatal(err)
	}
	if len(specs) < 11 {
		t.Fatalf("spec count = %d, want all steam targets", len(specs))
	}
	hasDayZ := false
	for _, spec := range specs {
		if spec.Target == "all-steam" {
			t.Fatalf("all-steam should resolve to concrete targets: %#v", specs)
		}
		if spec.Target == "dayzserver" {
			hasDayZ = true
		}
	}
	if !hasDayZ {
		t.Fatalf("all-steam should include dayzserver: %#v", specs)
	}
}

func TestDayZPrebuildRequiresSteamCredentials(t *testing.T) {
	specs, err := selectSpecs("dayzserver")
	if err != nil {
		t.Fatal(err)
	}
	if len(specs) != 1 {
		t.Fatalf("specs = %#v, want one spec", specs)
	}
	if len(specs[0].RequiredEnv) != 2 || specs[0].RequiredEnv[0] != "STEAM_USER" || specs[0].RequiredEnv[1] != "STEAM_PASS" {
		t.Fatalf("required env = %#v, want STEAM_USER and STEAM_PASS", specs[0].RequiredEnv)
	}
}

func TestRustPrebuildPortsAreDynamic(t *testing.T) {
	specs, err := selectSpecs("rust-vanilla,rust-oxide")
	if err != nil {
		t.Fatal(err)
	}
	for _, spec := range specs {
		want := []string{"main=/udp", "query=/udp", "rcon", "rustplus"}
		if len(spec.Ports) != len(want) {
			t.Fatalf("%s ports = %#v", spec.Target, spec.Ports)
		}
		for index := range want {
			if spec.Ports[index] != want[index] {
				t.Fatalf("%s ports = %#v, want %#v", spec.Target, spec.Ports, want)
			}
		}
	}
}

func TestSharedPortPrebuildsRemainConcrete(t *testing.T) {
	tests := []struct {
		target string
		want   []string
	}{
		{target: "cs2server", want: []string{"main=27015/udp", "rcon=27015"}},
		{target: "pzserver", want: []string{"main=16261/udp", "main2=16262/udp", "maintcp=16261"}},
	}
	for _, test := range tests {
		t.Run(test.target, func(t *testing.T) {
			specs, err := selectSpecs(test.target)
			if err != nil {
				t.Fatal(err)
			}
			if len(specs) != 1 || len(specs[0].Ports) != len(test.want) {
				t.Fatalf("specs = %#v", specs)
			}
			for index := range test.want {
				if specs[0].Ports[index] != test.want[index] {
					t.Fatalf("ports = %#v, want %#v", specs[0].Ports, test.want)
				}
			}
		})
	}
}

func TestValidateRequiredEnvFailsBeforePrebuild(t *testing.T) {
	t.Setenv("PREBUILD_TEST_REQUIRED", "")
	err := validateRequiredEnv(prebuildSpec{Target: "test", RequiredEnv: []string{"PREBUILD_TEST_REQUIRED"}})
	if err == nil {
		t.Fatal("validateRequiredEnv returned nil, want missing env error")
	}
}

func TestParseSizeBytesSupportsBinaryUnits(t *testing.T) {
	got, err := parseSizeBytes("1.5Gi")
	if err != nil {
		t.Fatal(err)
	}
	want := uint64(1610612736)
	if got != want {
		t.Fatalf("bytes = %d, want %d", got, want)
	}
}

func TestValidateRequiredEnvAcceptsPresentEnv(t *testing.T) {
	t.Setenv("PREBUILD_TEST_REQUIRED", "present")
	if err := validateRequiredEnv(prebuildSpec{Target: "test", RequiredEnv: []string{"PREBUILD_TEST_REQUIRED"}}); err != nil {
		t.Fatal(err)
	}
}

func TestSanitizeInstalledRootRemovesEscapingSymlinks(t *testing.T) {
	root := t.TempDir()
	dataRoot := filepath.Join(root, "data")
	if err := os.MkdirAll(filepath.Join(dataRoot, "Zomboid"), 0755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dataRoot, "Zomboid", "Logs")
	if err := os.Symlink("/home/druid/Zomboid/Logs", link); err != nil {
		t.Fatal(err)
	}

	if err := sanitizeInstalledRoot(root); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Lstat(link); !os.IsNotExist(err) {
		t.Fatalf("escaping symlink still exists or returned unexpected error: %v", err)
	}
}

func TestSanitizeInstalledRootKeepsInternalSymlinks(t *testing.T) {
	root := t.TempDir()
	dataRoot := filepath.Join(root, "data")
	if err := os.MkdirAll(filepath.Join(dataRoot, "shared"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dataRoot, "shared", "config.txt"), []byte("ok"), 0644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dataRoot, "config-link")
	if err := os.Symlink("shared/config.txt", link); err != nil {
		t.Fatal(err)
	}

	if err := sanitizeInstalledRoot(root); err != nil {
		t.Fatal(err)
	}

	target, err := os.Readlink(link)
	if err != nil {
		t.Fatal(err)
	}
	if target != "shared/config.txt" {
		t.Fatalf("link target = %q, want shared/config.txt", target)
	}
}

func TestRewriteInstallForPrebuildMakesInstallNoop(t *testing.T) {
	root := t.TempDir()
	scrollPath := filepath.Join(root, "scroll.yaml")
	content := []byte(`name: test
version: 1
app_version: test
commands:
  install:
    run: once
    procedures:
    - image: old
      command: ["real-install"]
  start:
    needs: ["install"]
    procedures:
    - image: old
      command: ["start"]
`)
	if err := os.WriteFile(scrollPath, content, 0644); err != nil {
		t.Fatal(err)
	}

	if err := rewriteInstallForPrebuild(root, prebuildSpec{Image: "runtime-image"}); err != nil {
		t.Fatal(err)
	}

	updated, err := os.ReadFile(scrollPath)
	if err != nil {
		t.Fatal(err)
	}
	var scroll map[string]any
	if err := yaml.Unmarshal(updated, &scroll); err != nil {
		t.Fatal(err)
	}
	install := scroll["commands"].(map[string]any)["install"].(map[string]any)
	procedures := install["procedures"].([]any)
	procedure := procedures[0].(map[string]any)
	if install["run"] != "once" {
		t.Fatalf("install run = %v, want once", install["run"])
	}
	if procedure["id"] != "prebuild" || procedure["image"] != "runtime-image" {
		t.Fatalf("procedure = %#v", procedure)
	}
	command := procedure["command"].([]any)
	if len(command) != 3 || command[0] != "sh" || command[1] != "-lc" {
		t.Fatalf("command = %#v", command)
	}
}
