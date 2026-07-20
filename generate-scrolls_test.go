package main

import (
	"os"
	"slices"
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
