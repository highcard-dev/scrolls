package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFindLegacyRuntimeTemplatesIncludesBuildSources(t *testing.T) {
	root := t.TempDir()
	legacy := filepath.Join(root, "scrolls", "game", ".build", "data", "server.cfg.scroll_template")
	if err := os.MkdirAll(filepath.Dir(legacy), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(legacy, []byte("port=1234\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "scroll.yaml.tmpl"), []byte("generated\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	files, err := findLegacyRuntimeTemplates(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || files[0] != legacy {
		t.Fatalf("legacy templates = %#v, want %q", files, legacy)
	}
}

func TestValidatePortsAllowsDynamicTransportPorts(t *testing.T) {
	ports := []any{
		map[string]any{"name": "tcp", "protocol": "tcp"},
		map[string]any{"name": "udp", "protocol": "udp"},
	}
	if _, err := validatePorts(ports); err != nil {
		t.Fatal(err)
	}
}

func TestValidatePortsRejectsDynamicHTTPPorts(t *testing.T) {
	for _, protocol := range []string{"http", "https"} {
		_, err := validatePorts([]any{
			map[string]any{"name": "web", "protocol": protocol},
		})
		if err == nil || !strings.Contains(err.Error(), "requires a fixed port") {
			t.Fatalf("%s error = %v", protocol, err)
		}
	}
}
