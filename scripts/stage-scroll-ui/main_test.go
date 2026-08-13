package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStageAddsPrivateUIAndMinecraftManifest(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "scrolls", "minecraft", "papermc", "1.21.7")
	if err := os.MkdirAll(source, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "scroll.yaml"), []byte("name: test\ndesc: test\nversion: 0.0.1\napp_version: 1.21.7\ncommands: {}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	bundle := filepath.Join(root, "app.wasm")
	if err := os.WriteFile(bundle, []byte("wasm"), 0644); err != nil {
		t.Fatal(err)
	}
	destination := filepath.Join(root, "staged")

	if err := stage(source, destination, bundle); err != nil {
		t.Fatal(err)
	}

	stagedYAML, err := os.ReadFile(filepath.Join(destination, "scroll.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if string(stagedYAML) == "" || !contains(string(stagedYAML), "path: private/dist/app.wasm") {
		t.Fatalf("staged yaml = %s", stagedYAML)
	}
	manifestBytes, err := os.ReadFile(filepath.Join(destination, "private", "config-editor.manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	var got manifest
	if err := json.Unmarshal(manifestBytes, &got); err != nil {
		t.Fatal(err)
	}
	if got.Server.Family != "minecraft" || got.Server.AppVersion != "1.21.7" {
		t.Fatalf("manifest server = %#v", got.Server)
	}
	if len(got.Files) != 1 || len(got.Files[0].Sections) < 4 {
		t.Fatalf("manifest files = %#v", got.Files)
	}
	if _, err := os.Stat(filepath.Join(destination, "private", "dist", "app.wasm")); err != nil {
		t.Fatal(err)
	}
}

func TestCatalogCoversEveryReleasedFamily(t *testing.T) {
	sources := []string{
		"scrolls/minecraft/papermc/1.21.7", "scrolls/minecraft/minecraft-vanilla/1.21.7",
		"scrolls/minecraft/minecraft-spigot/1.21.8", "scrolls/minecraft/forge/1.21.7",
		"scrolls/minecraft/cuberite/latest", "scrolls/rust/rust-vanilla/latest", "scrolls/rust/rust-oxide/latest",
		"scrolls/hytale/hytale-standalone", "scrolls/hytale/hytale-druid-gg",
		"scrolls/lgsm/arkserver", "scrolls/lgsm/cs2server", "scrolls/lgsm/csgoserver", "scrolls/lgsm/dayzserver",
		"scrolls/lgsm/gmodserver", "scrolls/lgsm/pwserver", "scrolls/lgsm/pzserver", "scrolls/lgsm/sdtdserver", "scrolls/lgsm/untserver",
	}
	for _, source := range sources {
		if _, err := familyManifest(source, "latest"); err != nil {
			t.Errorf("%s: %v", source, err)
		}
	}
}

func TestArkCatalogIncludesTypedSettingsAndRawFallback(t *testing.T) {
	got, err := familyManifest("scrolls/lgsm/arkserver", "arkserver")
	if err != nil {
		t.Fatal(err)
	}
	if got.Server.Family != "ark" || len(got.Files) != 2 {
		t.Fatalf("ARK manifest = %#v", got)
	}
	if len(got.Files[0].Sections) < 3 || len(got.Files[1].Sections[0].Fields) < 10 {
		t.Fatalf("ARK typed coverage is incomplete: %#v", got.Files)
	}
}

func TestEnsureConfigFilesCreatesRuntimeDefaultsButPreservesTemplates(t *testing.T) {
	root := t.TempDir()
	template := filepath.Join(root, "data", "templated.cfg.scroll_template")
	if err := os.MkdirAll(filepath.Dir(template), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(template, []byte("generated=true\n"), 0644); err != nil {
		t.Fatal(err)
	}
	config := manifest{Files: []fileSchema{
		{Path: "data/templated.cfg", Format: "key-value"},
		{Path: "data/config.json", Format: "json"},
		{Path: "data/server/druid/cfg/server.cfg", Format: "key-value"},
	}}

	if err := ensureConfigFiles(root, config); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "data", "templated.cfg")); !os.IsNotExist(err) {
		t.Fatalf("template output was unexpectedly created: %v", err)
	}
	jsonBytes, err := os.ReadFile(filepath.Join(root, "data", "config.json"))
	if err != nil || string(jsonBytes) != "{}\n" {
		t.Fatalf("JSON default = %q, %v", jsonBytes, err)
	}
	rustBytes, err := os.ReadFile(filepath.Join(root, "data", "server", "druid", "cfg", "server.cfg"))
	if err != nil || !strings.Contains(string(rustBytes), "server.maxplayers 75") {
		t.Fatalf("Rust default = %q, %v", rustBytes, err)
	}
}

func TestCatalogCoversEveryCheckedInGameServerScroll(t *testing.T) {
	count := 0
	err := filepath.Walk(filepath.Join("..", "..", "scrolls"), func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || info.Name() != "scroll.yaml" || strings.Contains(filepath.ToSlash(path), "/.sample/") {
			return nil
		}
		count++
		if _, err := familyManifest(filepath.ToSlash(path), "test"); err != nil {
			t.Errorf("%s: %v", path, err)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if count != 94 {
		t.Fatalf("checked Scroll count = %d, want 94", count)
	}
}

func TestEveryCheckedInGameServerScrollStagesAsACompleteUIPackage(t *testing.T) {
	bundle := filepath.Join(t.TempDir(), "app.wasm")
	if err := os.WriteFile(bundle, []byte("wasm"), 0644); err != nil {
		t.Fatal(err)
	}
	count := 0
	err := filepath.Walk(filepath.Join("..", "..", "scrolls"), func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || info.Name() != "scroll.yaml" || strings.Contains(filepath.ToSlash(path), "/.sample/") {
			return nil
		}
		count++
		destination := filepath.Join(t.TempDir(), "artifact")
		if err := stage(filepath.Dir(path), destination, bundle); err != nil {
			t.Errorf("stage %s: %v", path, err)
			return nil
		}
		manifestBytes, err := os.ReadFile(filepath.Join(destination, "private", "config-editor.manifest.json"))
		if err != nil {
			t.Errorf("manifest %s: %v", path, err)
			return nil
		}
		var got manifest
		if err := json.Unmarshal(manifestBytes, &got); err != nil {
			t.Errorf("parse manifest %s: %v", path, err)
			return nil
		}
		for _, file := range got.Files {
			target := filepath.Join(destination, filepath.FromSlash(file.Path))
			if _, err := os.Stat(target); err != nil {
				if _, templateErr := os.Stat(target + ".scroll_template"); templateErr != nil {
					t.Errorf("%s has neither config nor template for %s", path, file.Path)
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if count != 94 {
		t.Fatalf("staged Scroll count = %d, want 94", count)
	}
}

func contains(value, part string) bool {
	for i := 0; i+len(part) <= len(value); i++ {
		if value[i:i+len(part)] == part {
			return true
		}
	}
	return false
}
