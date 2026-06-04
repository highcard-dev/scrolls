package main

import "testing"

func TestPrebuildCommandWrapsTemplateBackedScripts(t *testing.T) {
	entrypoint, args := prebuildCommand([]string{"bash", "postinstall.sh"})
	if entrypoint != "sh" {
		t.Fatalf("entrypoint = %q, want sh", entrypoint)
	}
	if len(args) != 4 {
		t.Fatalf("args = %#v", args)
	}
	if args[0] != "-lc" || args[1] != templateBackedScriptWrapper || args[2] != "bash" || args[3] != "postinstall.sh" {
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

func TestValidateRequiredEnvFailsBeforePrebuild(t *testing.T) {
	t.Setenv("PREBUILD_TEST_REQUIRED", "")
	err := validateRequiredEnv(prebuildSpec{Target: "test", RequiredEnv: []string{"PREBUILD_TEST_REQUIRED"}})
	if err == nil {
		t.Fatal("validateRequiredEnv returned nil, want missing env error")
	}
}

func TestValidateRequiredEnvAcceptsPresentEnv(t *testing.T) {
	t.Setenv("PREBUILD_TEST_REQUIRED", "present")
	if err := validateRequiredEnv(prebuildSpec{Target: "test", RequiredEnv: []string{"PREBUILD_TEST_REQUIRED"}}); err != nil {
		t.Fatal(err)
	}
}
