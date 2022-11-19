package registry

import (
	"gopkg.in/yaml.v3"
	"io"
	"os"
)

type Scroll struct {
	Version SemanticVersion `yaml:"version"`
}

func OpenScrollYaml(path string) (*Scroll, error) {
	yamlFile, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer yamlFile.Close()

	byteValue, _ := io.ReadAll(yamlFile)
	var scroll Scroll
	err = yaml.Unmarshal(byteValue, &scroll)
	if err != nil {
		return nil, err
	}
	return &scroll, nil
}
