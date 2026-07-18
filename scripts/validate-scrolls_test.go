package main

import (
	"strings"
	"testing"
)

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
