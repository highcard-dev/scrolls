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
