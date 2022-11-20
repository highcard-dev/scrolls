package registry

import (
	"fmt"
	"github.com/Masterminds/semver/v3"
	"gopkg.in/yaml.v3"
)

type Variant string
type VariantVersion string
type SemanticVersion struct {
	*semver.Version `yaml:",inline"`
}

type Entry struct {
	Latest SemanticVersion   `yaml:"latest"`
	Old    []SemanticVersion `yaml:"old,omitempty"`
}

func NewRegistryEntry(latest SemanticVersion) Entry {
	return Entry{Latest: latest}
}

func (re Entry) Refresh(version SemanticVersion) (Entry, bool) {
	current := re.Latest
	isLatest := version.Version.GreaterThan(current.Version)
	isEqual := version.Version.Equal(current.Version)
	if isLatest || isEqual {
		re.Latest = version
		if !isEqual {
			var alreadyExist bool
			for _, semanticVersion := range re.Old {
				if semanticVersion == current {
					alreadyExist = true
				}
			}
			if !alreadyExist {
				re.Old = append(re.Old, current)
			}
		}
	}
	return re, isLatest || isEqual
}

func (s *SemanticVersion) UnmarshalYAML(value *yaml.Node) error {
	if value.Value != "" {
		v, err := semver.NewVersion(value.Value)
		if err != nil {
			return err
		}
		*s = SemanticVersion{v}
		return nil
	}
	return fmt.Errorf("no scroll version specified")
}

func (s SemanticVersion) MarshalYAML() (interface{}, error) {
	return s.Version.String(), nil
}
