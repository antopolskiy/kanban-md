// ---
// relationships:
//   references: 2026-08-12-preserve-extra-task-front-matter-properties
// ---

package task

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"go.yaml.in/yaml/v3"
)

var errTaskFrontmatterNotMapping = errors.New("task frontmatter must be a YAML mapping")

var canonicalTaskYAMLKeys = makeTaskYAMLKeys()

// taskYAML bypasses Task's YAML methods for kanban-md-owned fields.
type taskYAML Task

// UnmarshalYAML decodes canonical fields and retains unknown YAML without
// interpreting its values. Unsafe-to-rewrite metadata remains readable.
func (t *Task) UnmarshalYAML(value *yaml.Node) error {
	t.extraProperties = nil
	t.preservationError = nil
	mapping, err := taskFrontmatterMapping(value)
	if err != nil {
		return err
	}
	if err = mapping.Decode((*taskYAML)(t)); err != nil {
		return err
	}

	extra := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	if mapping.ShortTag() != "!!map" {
		t.preservationError = preservationRefusal(mapping, "frontmatter", "a custom root tag applies to canonical fields too")
	}
	for i := 0; i < len(mapping.Content); i += 2 {
		key := mapping.Content[i]
		switch {
		case key.Kind == yaml.AliasNode:
			t.preservationError = preservationRefusal(key, key.Value, "a top-level alias key cannot be retained safely")
		case key.ShortTag() == "!!merge":
			t.preservationError = preservationRefusal(key, "<<", "a top-level YAML merge can supply canonical fields")
		}
		if key.Kind == yaml.ScalarNode {
			if _, known := canonicalTaskYAMLKeys[key.Value]; known {
				continue
			}
		}
		extra.Content = append(extra.Content, key, mapping.Content[i+1])
	}
	t.extraProperties = extra
	if t.preservationError == nil {
		t.preservationError = validateRetainedAliases(extra)
	}
	return nil
}

// MarshalYAML regenerates authoritative canonical values, then appends the
// immutable retained pairs. Refusal happens before task.Write touches a file.
func (t Task) MarshalYAML() (any, error) {
	if t.preservationError != nil {
		return nil, t.preservationError
	}
	canonical, err := encodeCanonicalTask(&t)
	if err != nil {
		return nil, err
	}
	if t.extraProperties != nil {
		canonical.Content = append(canonical.Content, t.extraProperties.Content...)
	}
	return canonical, nil
}

// validateRetainedAliases walks emission order without following Alias pointers.
// Identity checks preserve ordering and rebinding, including repeated anchors.
func validateRetainedAliases(extra *yaml.Node) error {
	anchors := make(map[string]*yaml.Node)
	var walk func(*yaml.Node, string) error
	walk = func(node *yaml.Node, property string) error {
		if node.Anchor != "" {
			anchors[node.Anchor] = node
		}
		if node.Kind == yaml.AliasNode && (node.Alias == nil || anchors[node.Value] != node.Alias) {
			return preservationRefusal(node, property, "alias *"+node.Value+" depends on an anchor outside retained properties or changes binding")
		}
		for _, child := range node.Content {
			if err := walk(child, property); err != nil {
				return err
			}
		}
		return nil
	}
	for i := 0; i < len(extra.Content); i += 2 {
		key := extra.Content[i]
		if err := walk(key, key.Value); err != nil {
			return err
		}
		if err := walk(extra.Content[i+1], key.Value); err != nil {
			return err
		}
	}
	return nil
}

func preservationRefusal(node *yaml.Node, property, reason string) error {
	return fmt.Errorf("cannot preserve frontmatter property %q at line %d: %s; edit the frontmatter manually before retrying", property, node.Line, reason)
}

func encodeCanonicalTask(t *Task) (*yaml.Node, error) {
	var encoded yaml.Node
	if err := encoded.Encode((*taskYAML)(t)); err != nil {
		return nil, err
	}
	return taskFrontmatterMapping(&encoded)
}

func taskFrontmatterMapping(node *yaml.Node) (*yaml.Node, error) {
	if node.Kind == yaml.DocumentNode && len(node.Content) == 1 {
		node = node.Content[0]
	}
	if node.Kind != yaml.MappingNode {
		return nil, errTaskFrontmatterNotMapping
	}
	return node, nil
}

func makeTaskYAMLKeys() map[string]struct{} {
	typ := reflect.TypeOf(taskYAML{})
	keys := make(map[string]struct{}, typ.NumField())
	for i := 0; i < typ.NumField(); i++ {
		tag := typ.Field(i).Tag.Get("yaml")
		name := strings.Split(tag, ",")[0]
		if name != "" && name != "-" {
			keys[name] = struct{}{}
		}
	}
	return keys
}
