package config

import (
	"errors"
	"fmt"
	"slices"

	"go.yaml.in/yaml/v3"

	"github.com/antopolskiy/kanban-md/internal/property"
)

// FieldList distinguishes omission from an explicitly empty or null list.
type FieldList []string

// UnmarshalYAML requires a supplied list to contain selectors.
func (fields *FieldList) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind != yaml.SequenceNode || len(node.Content) == 0 {
		return errors.New("display field list must contain one or two selectors; null and empty lists are invalid")
	}
	var values []string
	if err := node.Decode(&values); err != nil {
		return err
	}
	*fields = values
	return nil
}

// DisplayConfig selects compact human fields.
type DisplayConfig struct {
	CompactFields FieldList `yaml:"compact_fields,omitempty"`
}

// ChildrenConfig selects direct-child detail presentation only.
type ChildrenConfig struct {
	DetailSort string `yaml:"detail_sort,omitempty"`
}

// PropertyGroupOrders orders scalar groups without restricting task values.
type PropertyGroupOrders map[string][]property.Scalar

// UnmarshalYAML parses scalar nodes directly, including explicit null.
func (orders *PropertyGroupOrders) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind != yaml.MappingNode {
		return errors.New("group_orders must be a mapping")
	}
	values := PropertyGroupOrders{}
	for i := 0; i < len(node.Content); i += 2 {
		key, list := node.Content[i], node.Content[i+1]
		if key.ShortTag() != "!!str" || list.Kind != yaml.SequenceNode {
			return errors.New("group_orders entries must name a selector and a scalar list")
		}
		if _, duplicate := values[key.Value]; duplicate {
			return fmt.Errorf("duplicate group order %q", key.Value)
		}
		values[key.Value] = []property.Scalar{}
		for _, item := range list.Content {
			value, err := property.FromYAML(item)
			if err != nil {
				return fmt.Errorf("group order %q: %w", key.Value, err)
			}
			values[key.Value] = append(values[key.Value], value)
		}
	}
	*orders = values
	return nil
}

// UnmarshalYAML rejects explicit null field lists which yaml otherwise skips.
func (c *Config) UnmarshalYAML(node *yaml.Node) error {
	type plain Config
	if node.Kind != yaml.MappingNode {
		return node.Decode((*plain)(c))
	}
	for i := 0; i < len(node.Content); i += 2 {
		section, value := node.Content[i], node.Content[i+1]
		name := ""
		if section.Value == "display" {
			name = "compact_fields"
		}
		if section.Value == "tui" {
			name = "card_fields"
		}
		if name == "" || value.Kind != yaml.MappingNode {
			continue
		}
		for j := 0; j < len(value.Content); j += 2 {
			if value.Content[j].Value == name && value.Content[j+1].ShortTag() == "!!null" {
				return fmt.Errorf("%s.%s cannot be null; omit it for defaults", section.Value, name)
			}
		}
	}
	return node.Decode((*plain)(c))
}

// CompactFields returns the selected compact fields, with legacy defaults.
func (c *Config) CompactFields() []string {
	if c.Display.CompactFields == nil {
		return []string{"status", "priority"}
	}
	return append([]string{}, c.Display.CompactFields...)
}

// CardFields returns the selected TUI fields, with legacy defaults.
func (c *Config) CardFields() []string {
	if c.TUI.CardFields == nil {
		return []string{"priority"}
	}
	return append([]string{}, c.TUI.CardFields...)
}

func (c *Config) validatePropertyViews() error {
	if err := validateFieldList(c.Display.CompactFields, "display.compact_fields"); err != nil {
		return err
	}
	if err := validateFieldList(c.TUI.CardFields, "tui.card_fields"); err != nil {
		return err
	}
	if c.Children.DetailSort != "" && c.Children.DetailSort != "id" {
		_, selected, err := property.SelectorKey(c.Children.DetailSort)
		if err != nil || !selected {
			return fmt.Errorf("%w: children.detail_sort must be id or property:KEY", ErrInvalid)
		}
	}
	for selector, values := range c.GroupOrders {
		_, selected, err := property.SelectorKey(selector)
		if err != nil || !selected {
			return fmt.Errorf("%w: invalid property group order selector %q", ErrInvalid, selector)
		}
		seen := map[string]bool{}
		for _, value := range values {
			if value.Kind() == property.Invalid || seen[value.Identity()] {
				return fmt.Errorf("%w: group order %q contains an invalid or duplicate scalar", ErrInvalid, selector)
			}
			seen[value.Identity()] = true
		}
	}
	return nil
}

func (c *Config) validateViews() error {
	if err := c.validateTUI(); err != nil {
		return err
	}
	return c.validatePropertyViews()
}

func validateFieldList(fields FieldList, name string) error {
	if fields == nil {
		return nil
	}
	if len(fields) < 1 || len(fields) > 2 {
		return fmt.Errorf("%w: %s must contain one or two selectors", ErrInvalid, name)
	}
	seen := map[string]bool{}
	for _, field := range fields {
		_, selected, err := property.SelectorKey(field)
		if err != nil || !selected && !slices.Contains([]string{"status", "priority", "class"}, field) {
			return fmt.Errorf("%w: invalid %s selector %q", ErrInvalid, name, field)
		}
		if seen[field] {
			return fmt.Errorf("%w: duplicate %s selector %q", ErrInvalid, name, field)
		}
		seen[field] = true
	}
	return nil
}
