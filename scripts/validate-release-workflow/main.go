package main

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
)

var (
	forbiddenReleaseValues = []string{
		"v0.1.248",
		"highcard/druid",
		"stable-nix",
		"latest-nix",
		"-nix-steamcmd",
	}
	portOverrideNamePattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_-]*$`)
	portOverridePattern     = regexp.MustCompile(`^([A-Za-z][A-Za-z0-9_-]*)=([0-9]+)(?:/(tcp|udp|http|https))?$`)
	pzserverRequiredPorts   = map[string]string{
		"main":    "16261/udp",
		"main2":   "16262/udp",
		"maintcp": "16261",
	}
)

func main() {
	path := ".github/workflows/release.yml"
	if len(os.Args) > 1 {
		path = os.Args[1]
	}
	if err := validateReleaseWorkflow(path); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

func validateReleaseWorkflow(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	lines := strings.Split(string(data), "\n")
	var failures []string
	foundPZServer := false

	for index, line := range lines {
		lineNumber := index + 1
		for _, forbidden := range forbiddenReleaseValues {
			if strings.Contains(line, forbidden) {
				failures = append(failures, fmt.Sprintf("%s:%d contains forbidden value %q", path, lineNumber, forbidden))
			}
		}
		if !isArtifactPushLine(line) {
			continue
		}
		fields := strings.Fields(line)
		ports, err := validatePortOverrides(fields)
		if err != nil {
			failures = append(failures, fmt.Sprintf("%s:%d %v", path, lineNumber, err))
		}
		if strings.Contains(line, "scroll-lgsm:pzserver") {
			foundPZServer = true
			for name, expected := range pzserverRequiredPorts {
				if actual := ports[name]; actual != expected {
					failures = append(failures, fmt.Sprintf("%s:%d pzserver requires -p %s=%s, got %q", path, lineNumber, name, expected, actual))
				}
			}
		}
	}

	if !foundPZServer {
		failures = append(failures, fmt.Sprintf("%s: missing pzserver release push", path))
	}
	if len(failures) > 0 {
		return errors.New(strings.Join(failures, "\n"))
	}
	return nil
}

func isArtifactPushLine(line string) bool {
	fields := strings.Fields(line)
	for i := 0; i+2 < len(fields); i++ {
		if fields[i] == "druid" && fields[i+1] == "push" && fields[i+2] != "category" {
			return true
		}
	}
	return false
}

func validatePortOverrides(fields []string) (map[string]string, error) {
	ports := make(map[string]string)
	var failures []string

	for i, field := range fields {
		if field != "-p" && field != "--port" {
			continue
		}
		if i+1 >= len(fields) {
			failures = append(failures, fmt.Sprintf("%s is missing an override", field))
			continue
		}
		override := fields[i+1]
		matches := portOverridePattern.FindStringSubmatch(override)
		if matches == nil {
			failures = append(failures, fmt.Sprintf("invalid %s override %q; expected name=port or name=port/protocol", field, override))
			continue
		}
		name := matches[1]
		portText := matches[2]
		port, err := strconv.Atoi(portText)
		if err != nil || port < 1 || port > 65535 {
			failures = append(failures, fmt.Sprintf("invalid %s override %q; port must be 1-65535", field, override))
			continue
		}
		if !portOverrideNamePattern.MatchString(name) {
			failures = append(failures, fmt.Sprintf("invalid %s override %q; invalid port name", field, override))
			continue
		}
		ports[name] = strings.TrimPrefix(override, name+"=")
	}

	if len(failures) > 0 {
		return ports, errors.New(strings.Join(failures, "; "))
	}
	return ports, nil
}
