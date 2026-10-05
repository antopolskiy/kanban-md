package task

import (
	"errors"
	"fmt"

	"go.yaml.in/yaml/v3"

	"github.com/antopolskiy/kanban-md/internal/property"
)

// PropertyState distinguishes absent data from unsupported retained values.
type PropertyState uint8

// Property states do not coerce absent or unsupported data into a scalar.
const (
	PropertyMissing PropertyState = iota
	PropertySupported
	PropertyUnsupported
)

// PropertyScalar returns a detached direct scalar and never follows aliases.
func (t *Task) PropertyScalar(key string) (property.Scalar, PropertyState) {
	if property.ValidateKey(key) != nil {
		return property.Scalar{}, PropertyUnsupported
	}
	index, err := t.propertyIndex(key)
	if err != nil {
		return property.Scalar{}, PropertyUnsupported
	}
	if index < 0 {
		return property.Scalar{}, PropertyMissing
	}
	value, err := property.FromYAML(t.extraProperties.Content[index+1])
	if err != nil {
		return property.Scalar{}, PropertyUnsupported
	}
	return value, PropertySupported
}

// SetPropertyScalar replaces one pair without mutating any shared retained node.
func (t *Task) SetPropertyScalar(key string, value property.Scalar) (bool, error) {
	if err := validatePropertyWriteKey(key); err != nil {
		return false, err
	}
	if value.Kind() == property.Invalid {
		return false, errors.New("invalid property scalar")
	}
	index, err := t.propertyIndex(key)
	if err != nil {
		return false, err
	}
	old, state := t.PropertyScalar(key)
	if state == PropertySupported && old.Equal(value) {
		return false, nil
	}
	encoded, err := value.MarshalYAML()
	if err != nil {
		return false, err
	}
	node := encoded.(*yaml.Node)
	extra := t.copyPropertyPairs()
	if index >= 0 {
		extra.Content[index+1] = node
	} else {
		extra.Content = append(extra.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key}, node)
	}
	t.extraProperties = extra
	return true, nil
}

// ClearProperty removes one pair. Final graph admissibility is checked at write.
func (t *Task) ClearProperty(key string) (bool, error) {
	if err := validatePropertyWriteKey(key); err != nil {
		return false, err
	}
	index, err := t.propertyIndex(key)
	if err != nil {
		return false, err
	}
	if index < 0 {
		return false, nil
	}
	extra := t.copyPropertyPairs()
	extra.Content = append(extra.Content[:index], extra.Content[index+2:]...)
	t.extraProperties = extra
	return true, nil
}

func validatePropertyWriteKey(key string) error {
	if err := property.ValidateKey(key); err != nil {
		return err
	}
	if _, owned := canonicalTaskYAMLKeys[key]; owned {
		return fmt.Errorf("property key %q is owned by kanban-md", key)
	}
	return nil
}

func (t *Task) propertyIndex(key string) (int, error) {
	if t.extraProperties == nil {
		return -1, nil
	}
	for i := 0; i < len(t.extraProperties.Content); i += 2 {
		node := t.extraProperties.Content[i]
		if node.Kind == yaml.ScalarNode && node.Value == key {
			if node.ShortTag() != "!!str" {
				return -1, fmt.Errorf("property %q has an unsupported key tag", key)
			}
			return i, nil
		}
	}
	return -1, nil
}

func (t *Task) copyPropertyPairs() *yaml.Node {
	extra := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	if t.extraProperties != nil {
		extra.Content = append([]*yaml.Node{}, t.extraProperties.Content...)
	}
	return extra
}
