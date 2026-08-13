package main

import (
	"strings"
	"testing"
)

func TestValidatePortOverridesAcceptsConcretePorts(t *testing.T) {
	fields := strings.Fields("druid push repo:tag ./scroll -p main=25565 -p query=27015/udp -p battle-eye=2304/udp -p webpanel=8080")

	ports, err := validatePortOverrides(fields)
	if err != nil {
		t.Fatal(err)
	}
	if ports["main"] != "25565" || ports["query"] != "27015/udp" || ports["battle-eye"] != "2304/udp" || ports["webpanel"] != "8080" {
		t.Fatalf("ports = %#v", ports)
	}
}

func TestValidatePortOverridesAcceptsDynamicPorts(t *testing.T) {
	for _, value := range []string{
		"main",
		"main=",
		"main=/udp",
		"main=0",
		"main=0/udp",
	} {
		fields := []string{"druid", "push", "repo:tag", "./scroll", "-p", value}
		if _, err := validatePortOverrides(fields); err != nil {
			t.Fatalf("%s failed validation: %v", value, err)
		}
	}
}

func TestValidatePortOverridesRejectsInvalidPorts(t *testing.T) {
	for _, value := range []string{
		"main=65536",
		"main=/",
		"main=/sctp",
		"web=/http",
		"web=0/https",
		"main=123/udp/extra",
		"1main=123",
	} {
		fields := []string{"druid", "push", "repo:tag", "./scroll", "-p", value}
		if _, err := validatePortOverrides(fields); err == nil {
			t.Fatalf("%s passed validation, want failure", value)
		}
	}
}

func TestIsArtifactPushLineSkipsCategories(t *testing.T) {
	if isArtifactPushLine("druid push category artifacts.druid.gg/foo bar ./meta") {
		t.Fatal("category push should not be treated as artifact push")
	}
	if !isArtifactPushLine("druid push artifacts.druid.gg/foo:tag ./scroll -p main=1") {
		t.Fatal("artifact push was not detected")
	}
}
