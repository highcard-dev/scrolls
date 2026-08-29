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
	if err := os.MkdirAll(filepath.Join(source, "data"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "data", "server.properties.default"), []byte("max-players=20\n"), 0644); err != nil {
		t.Fatal(err)
	}
	uiSource, bundle := writeTestUISource(t)
	if err := os.WriteFile(filepath.Join(uiSource, "dist", "stale.js"), []byte("stale\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(uiSource, "node_modules", "dependency"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(uiSource, "node_modules", "dependency", "index.js"), []byte("dependency\n"), 0644); err != nil {
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
	manifestBytes, err := os.ReadFile(filepath.Join(destination, "data", ".druid", "config-editor.manifest.json"))
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
	if got.Files[0].Path != "server.properties" {
		t.Fatalf("runtime manifest path = %q, want server.properties", got.Files[0].Path)
	}
	if _, err := os.Stat(filepath.Join(destination, "private", "dist", "app.wasm")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(destination, "private", "src", "app.tsx")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(destination, "private", "package.json")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(destination, "private", "dist", "stale.js")); !os.IsNotExist(err) {
		t.Fatalf("stale build output was packaged: %v", err)
	}
	if _, err := os.Stat(filepath.Join(destination, "private", "node_modules")); !os.IsNotExist(err) {
		t.Fatalf("node_modules was packaged: %v", err)
	}
	if _, err := os.Stat(filepath.Join(destination, "data", "server.properties")); !os.IsNotExist(err) {
		t.Fatalf("active config was unexpectedly created beside packaged default: %v", err)
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

func TestLinuxGSMCatalogsExposeManagementAndGameConfiguration(t *testing.T) {
	want := map[string][]string{
		"arkserver":  {"data/lgsm/config-lgsm/arkserver/arkserver.cfg", "data/serverfiles/ShooterGame/Saved/Config/LinuxServer/GameUserSettings.ini", "data/serverfiles/ShooterGame/Saved/Config/LinuxServer/Game.ini"},
		"cs2server":  {"data/lgsm/config-lgsm/cs2server/cs2server.cfg", "data/serverfiles/game/csgo/cfg/server.cfg"},
		"csgoserver": {"data/lgsm/config-lgsm/csgoserver/csgoserver.cfg", "data/serverfiles/csgo/cfg/csgoserver.cfg"},
		"dayzserver": {"data/lgsm/config-lgsm/dayzserver/common.cfg", "data/lgsm/config-lgsm/dayzserver/dayzserver.cfg", "data/serverfiles/cfg/dayzserver.server.cfg"},
		"gmodserver": {"data/lgsm/config-lgsm/gmodserver/gmodserver.cfg", "data/serverfiles/garrysmod/cfg/gmodserver.cfg"},
		"pwserver":   {"data/lgsm/config-lgsm/pwserver/common.cfg", "data/lgsm/config-lgsm/pwserver/pwserver.cfg", "data/serverfiles/Pal/Saved/Config/LinuxServer/PalWorldSettings.ini"},
		"pzserver":   {"data/lgsm/config-lgsm/pzserver/pzserver.cfg", "data/Zomboid/Server/pzserver.ini"},
		"sdtdserver": {"data/lgsm/config-lgsm/sdtdserver/sdtdserver.cfg", "data/serverfiles/sdtdserver.xml"},
		"untserver":  {"data/lgsm/config-lgsm/untserver/untserver.cfg", "data/serverfiles/Servers/untserver/Config.json", "data/serverfiles/Servers/untserver/Commands.dat", "data/serverfiles/Servers/untserver/WorkshopDownloadConfig.json"},
	}
	for id, paths := range want {
		got, err := familyManifest("scrolls/lgsm/"+id, "latest")
		if err != nil {
			t.Fatalf("%s: %v", id, err)
		}
		gotPaths := make([]string, 0, len(got.Files))
		for _, file := range got.Files {
			gotPaths = append(gotPaths, file.Path)
		}
		if strings.Join(gotPaths, "\n") != strings.Join(paths, "\n") {
			t.Errorf("%s paths = %#v, want %#v", id, gotPaths, paths)
		}
	}
}

func TestHytaleCatalogIncludesEveryDocumentedTopLevelConfiguration(t *testing.T) {
	got, err := familyManifest("scrolls/hytale/hytale-standalone", "latest")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"data/Server/config.json", "data/Server/permissions.json", "data/Server/whitelist.json", "data/Server/bans.json"}
	gotPaths := make([]string, 0, len(got.Files))
	for _, file := range got.Files {
		gotPaths = append(gotPaths, file.Path)
	}
	if strings.Join(gotPaths, "\n") != strings.Join(want, "\n") {
		t.Fatalf("Hytale paths = %#v, want %#v", gotPaths, want)
	}
}

func TestSevenDaysToDieUsesTypedXMLProperties(t *testing.T) {
	got, err := familyManifest("scrolls/lgsm/sdtdserver", "latest")
	if err != nil {
		t.Fatal(err)
	}
	if got.Files[1].Format != "xml-properties" {
		t.Fatalf("7 Days to Die game config format = %q", got.Files[1].Format)
	}
}

func TestCuberiteCatalogIncludesDocumentedServerAndDefaultWorldConfiguration(t *testing.T) {
	got, err := familyManifest("scrolls/minecraft/cuberite/latest", "latest")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"data/settings.ini", "data/webadmin.ini",
		"data/world/world.ini", "data/world_nether/world.ini", "data/world_the_end/world.ini",
		"data/monsters.ini", "data/motd.txt", "data/crafting.txt", "data/brewing.txt",
		"data/furnace.txt", "data/items.ini",
	}
	gotPaths := make([]string, 0, len(got.Files))
	for _, file := range got.Files {
		gotPaths = append(gotPaths, file.Path)
	}
	if strings.Join(gotPaths, "\n") != strings.Join(want, "\n") {
		t.Fatalf("Cuberite paths = %#v, want %#v", gotPaths, want)
	}
}

func TestArkCatalogIncludesTypedSettingsAndRawFallback(t *testing.T) {
	got, err := familyManifest("scrolls/lgsm/arkserver", "arkserver")
	if err != nil {
		t.Fatal(err)
	}
	if got.Server.Family != "ark" || len(got.Files) != 3 {
		t.Fatalf("ARK manifest = %#v", got)
	}
	if len(got.Files[1].Sections) < 3 || len(got.Files[2].Sections[0].Fields) < 10 {
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
	packagedDefault := filepath.Join(root, "data", "packaged.cfg.default")
	if err := os.WriteFile(packagedDefault, []byte("generated=false\n"), 0644); err != nil {
		t.Fatal(err)
	}
	config := manifest{Files: []fileSchema{
		{Path: "data/templated.cfg", Format: "key-value"},
		{Path: "data/packaged.cfg", Format: "key-value"},
		{Path: "data/config.json", Format: "json", CreateIfMissing: true},
		{Path: "data/server/druid/cfg/server.cfg", Format: "key-value", CreateIfMissing: true},
		{Path: "data/settings.ini", Format: "ini", CreateIfMissing: true},
	}}

	if err := ensureConfigFiles(root, config); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "data", "templated.cfg")); !os.IsNotExist(err) {
		t.Fatalf("template output was unexpectedly created: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "data", "packaged.cfg")); !os.IsNotExist(err) {
		t.Fatalf("packaged-default output was unexpectedly created: %v", err)
	}
	jsonBytes, err := os.ReadFile(filepath.Join(root, "data", "config.json"))
	if err != nil || string(jsonBytes) != "{}\n" {
		t.Fatalf("JSON default = %q, %v", jsonBytes, err)
	}
	rustBytes, err := os.ReadFile(filepath.Join(root, "data", "server", "druid", "cfg", "server.cfg"))
	if err != nil || !strings.Contains(string(rustBytes), "server.maxplayers 75") ||
		!strings.Contains(string(rustBytes), "server.level Procedural Map") ||
		!strings.Contains(string(rustBytes), "server.headerimage https://druid.gg/") ||
		!strings.Contains(string(rustBytes), "server.url https://druid.gg/") {
		t.Fatalf("Rust default = %q, %v", rustBytes, err)
	}
	cuberiteBytes, err := os.ReadFile(filepath.Join(root, "data", "settings.ini"))
	if err != nil || !strings.Contains(string(cuberiteBytes), "[Server]") ||
		!strings.Contains(string(cuberiteBytes), "Ports=25565") ||
		!strings.Contains(string(cuberiteBytes), "[Worlds]") {
		t.Fatalf("Cuberite default = %q, %v", cuberiteBytes, err)
	}
}

func TestRustStartScriptsDoNotOverrideUIManagedConvars(t *testing.T) {
	managed := []string{
		"-server.maxplayers", "-server.hostname", "-server.level", "-server.worldsize",
		"-server.saveinterval", "-server.globalchat", "-server.description",
		"-server.headerimage", "-server.url",
	}
	for _, variant := range []string{"rust-vanilla", "rust-oxide"} {
		path := filepath.Join("..", "..", "scrolls", "rust", variant, "latest", "data", "start.sh")
		contents, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		for _, flag := range managed {
			if strings.Contains(string(contents), flag) {
				t.Errorf("%s still overrides UI-managed convar %s", variant, flag)
			}
		}
		if !strings.Contains(string(contents), `-server.identity "druid"`) {
			t.Errorf("%s must retain server.identity to locate data/server/druid/cfg/server.cfg", variant)
		}
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
	if count != 124 {
		t.Fatalf("checked Scroll count = %d, want 124", count)
	}
}

func TestEveryCheckedInGameServerScrollStagesAsACompleteUIPackage(t *testing.T) {
	_, bundle := writeTestUISource(t)
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
		manifestBytes, err := os.ReadFile(filepath.Join(destination, "data", ".druid", "config-editor.manifest.json"))
		if err != nil {
			t.Errorf("manifest %s: %v", path, err)
			return nil
		}
		var got manifest
		if err := json.Unmarshal(manifestBytes, &got); err != nil {
			t.Errorf("parse manifest %s: %v", path, err)
			return nil
		}
		expected, manifestErr := familyManifest(filepath.ToSlash(path), "test")
		if manifestErr != nil {
			t.Errorf("catalog %s: %v", path, manifestErr)
			return nil
		}
		for index, file := range got.Files {
			target := filepath.Join(destination, "data", filepath.FromSlash(file.Path))
			if _, err := os.Stat(target); err != nil {
				_, templateErr := os.Stat(target + ".scroll_template")
				_, defaultErr := os.Stat(target + ".default")
				if templateErr != nil && defaultErr != nil && expected.Files[index].CreateIfMissing {
					t.Errorf("%s has neither config nor template for %s", path, file.Path)
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if count != 124 {
		t.Fatalf("staged Scroll count = %d, want 124", count)
	}
}

func writeTestUISource(t *testing.T) (string, string) {
	t.Helper()
	uiSource := filepath.Join(t.TempDir(), "ui")
	if err := os.MkdirAll(filepath.Join(uiSource, "src"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(uiSource, "package.json"), []byte("{}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(uiSource, "src", "app.tsx"), []byte("export const source = true;\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(uiSource, "dist"), 0755); err != nil {
		t.Fatal(err)
	}
	bundle := filepath.Join(uiSource, "dist", "app.wasm")
	if err := os.WriteFile(bundle, []byte("wasm"), 0644); err != nil {
		t.Fatal(err)
	}
	return uiSource, bundle
}

func contains(value, part string) bool {
	for i := 0; i+len(part) <= len(value); i++ {
		if value[i:i+len(part)] == part {
			return true
		}
	}
	return false
}
