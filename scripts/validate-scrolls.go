package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

var envNamePattern = regexp.MustCompile(`^[A-Z][A-Z0-9_]*$`)

func main() {
	files, err := findScrollFiles(".")
	if err != nil {
		fail(err)
	}
	if len(files) == 0 {
		fail(errors.New("no scroll.yaml files found"))
	}
	for _, file := range files {
		fmt.Printf("Validating %s\n", filepath.Dir(file))
		if err := validateScroll(file, shouldValidateColdstarterFiles(file)); err != nil {
			fail(fmt.Errorf("%s: %w", file, err))
		}
	}
}

func findScrollFiles(root string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			name := d.Name()
			if name == ".git" || name == ".build" || name == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}
		if d.Name() == "scroll.yaml" {
			files = append(files, path)
		}
		return nil
	})
	sort.Strings(files)
	return files, err
}

func validateScroll(file string, validateColdstarterFiles bool) error {
	data, err := os.ReadFile(file)
	if err != nil {
		return err
	}
	var scroll map[string]any
	if err := yaml.Unmarshal(data, &scroll); err != nil {
		return err
	}
	if requiredString(scroll, "name") == "" {
		return errors.New("missing name")
	}
	if requiredString(scroll, "version") == "" {
		return errors.New("missing version")
	}
	if requiredString(scroll, "app_version") == "" {
		return errors.New("missing app_version")
	}
	portNames, err := validatePorts(scroll["ports"])
	if err != nil {
		return err
	}
	commands, ok := asMap(scroll["commands"])
	if !ok || len(commands) == 0 {
		return errors.New("missing commands")
	}
	for commandName, rawCommand := range commands {
		command, ok := asMap(rawCommand)
		if !ok {
			return fmt.Errorf("command %s must be an object", commandName)
		}
		if err := validateCommand(filepath.Dir(file), commandName, command, portNames, validateColdstarterFiles); err != nil {
			return err
		}
	}
	if serve := optionalString(scroll["serve"]); serve != "" {
		if _, ok := commands[serve]; !ok {
			return fmt.Errorf("serve command %q does not exist", serve)
		}
	}
	return nil
}

func validatePorts(raw any) (map[string]bool, error) {
	names := map[string]bool{}
	if raw == nil {
		return names, nil
	}
	ports, ok := asList(raw)
	if !ok {
		return nil, errors.New("ports must be a list")
	}
	for index, rawPort := range ports {
		port, ok := asMap(rawPort)
		if !ok {
			return nil, fmt.Errorf("ports[%d] must be an object", index)
		}
		name := requiredString(port, "name")
		if name == "" {
			return nil, fmt.Errorf("ports[%d] missing name", index)
		}
		if names[name] {
			return nil, fmt.Errorf("duplicate port %q", name)
		}
		names[name] = true
		protocol := optionalString(port["protocol"])
		if protocol != "" && protocol != "tcp" && protocol != "udp" && protocol != "http" && protocol != "https" {
			return nil, fmt.Errorf("port %q has unsupported protocol %q", name, protocol)
		}
	}
	return names, nil
}

func validateCommand(root string, name string, command map[string]any, portNames map[string]bool, validateColdstarterFiles bool) error {
	run := optionalString(command["run"])
	if run != "" && run != "always" && run != "once" && run != "restart" && run != "persistent" {
		return fmt.Errorf("command %s has unsupported run mode %q", name, run)
	}
	if _, err := asStringList(command["needs"]); err != nil {
		return fmt.Errorf("command %s needs: %w", name, err)
	}
	procedures, ok := asList(command["procedures"])
	if !ok || len(procedures) == 0 {
		return fmt.Errorf("command %s missing procedures", name)
	}
	for index, rawProcedure := range procedures {
		procedure, ok := asMap(rawProcedure)
		if !ok {
			return fmt.Errorf("command %s procedure %d must be an object", name, index)
		}
		if err := validateProcedure(root, name, index, procedure, portNames, validateColdstarterFiles); err != nil {
			return err
		}
	}
	return nil
}

func validateProcedure(root string, command string, index int, procedure map[string]any, portNames map[string]bool, validateColdstarterFiles bool) error {
	prefix := fmt.Sprintf("command %s procedure %d", command, index)
	if mode := optionalString(procedure["mode"]); mode != "" && mode != "container" && mode != "exec" && mode != "signal" {
		return fmt.Errorf("%s has unsupported mode %q", prefix, mode)
	}
	if env, ok := asStringMap(procedure["env"]); ok {
		for key, value := range env {
			if !envNamePattern.MatchString(key) {
				return fmt.Errorf("%s env key %q is not uppercase snake case", prefix, key)
			}
			if strings.HasPrefix(key, "DRUID_PORT_") && strings.HasSuffix(key, "_COLDSTARTER") {
				if value == "generic" {
					continue
				}
				if filepath.IsAbs(value) || strings.Contains(value, "..") {
					return fmt.Errorf("%s coldstarter handler %q must be a safe relative path", prefix, value)
				}
				if validateColdstarterFiles {
					if _, err := os.Stat(filepath.Join(root, value)); err != nil {
						return fmt.Errorf("%s coldstarter handler %q does not exist", prefix, value)
					}
				}
			}
		}
	}
	if expectedPorts, ok := asList(procedure["expectedPorts"]); ok {
		for portIndex, rawPort := range expectedPorts {
			port, ok := asMap(rawPort)
			if !ok {
				return fmt.Errorf("%s expectedPorts[%d] must be an object", prefix, portIndex)
			}
			name := requiredString(port, "name")
			if name == "" {
				return fmt.Errorf("%s expectedPorts[%d] missing name", prefix, portIndex)
			}
			if !portNames[name] {
				return fmt.Errorf("%s expectedPorts[%d] references unknown port %q", prefix, portIndex, name)
			}
		}
	}
	return nil
}

func shouldValidateColdstarterFiles(file string) bool {
	return strings.Contains(filepath.ToSlash(file), "/minecraft/")
}

func requiredString(values map[string]any, key string) string {
	return optionalString(values[key])
}

func optionalString(value any) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case int:
		return fmt.Sprint(typed)
	case int64:
		return fmt.Sprint(typed)
	case float64:
		return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%f", typed), "0"), ".")
	default:
		return ""
	}
}

func asMap(value any) (map[string]any, bool) {
	typed, ok := value.(map[string]any)
	return typed, ok
}

func asList(value any) ([]any, bool) {
	typed, ok := value.([]any)
	return typed, ok
}

func asStringList(value any) ([]string, error) {
	if value == nil {
		return nil, nil
	}
	list, ok := asList(value)
	if !ok {
		return nil, errors.New("must be a list")
	}
	out := make([]string, 0, len(list))
	for _, item := range list {
		text, ok := item.(string)
		if !ok {
			return nil, errors.New("must contain only strings")
		}
		out = append(out, text)
	}
	return out, nil
}

func asStringMap(value any) (map[string]string, bool) {
	values, ok := asMap(value)
	if !ok {
		return nil, false
	}
	out := make(map[string]string, len(values))
	for key, value := range values {
		out[key] = fmt.Sprint(value)
	}
	return out, true
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "Error:", err)
	os.Exit(1)
}
