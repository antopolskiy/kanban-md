// Package property provides literal scalar values for explicitly selected task properties.
package property

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"go.yaml.in/yaml/v3"
)

// Kind is the semantic kind of a scalar. A zero Scalar is invalid.
type Kind uint8

// Scalar kinds distinguish typed values, including explicit null.
const (
	Invalid Kind = iota
	String
	Number
	Boolean
	Null
)

// Scalar is immutable and detached from retained YAML nodes.
type Scalar struct {
	kind Kind
	text string
	num  number
}

var (
	errInvalidScalar     = errors.New("invalid property scalar")
	errUnsupportedScalar = errors.New("property is not a supported scalar")
)

// Kind returns the scalar's semantic kind.
func (s Scalar) Kind() Kind { return s.kind }

// Equal compares scalar kinds and exact values.
func (s Scalar) Equal(other Scalar) bool {
	return s.kind != Invalid && s.kind == other.kind && s.Identity() == other.Identity()
}

// Identity separates string, numeric, boolean, and null groups.
func (s Scalar) Identity() string {
	if s.kind == Number {
		return fmt.Sprintf("%d:%t:%s:%s", s.kind, s.num.negative, s.num.digits, s.num.exponent)
	}
	return fmt.Sprintf("%d:%s", s.kind, s.text)
}

// CompareNumber compares exact finite numbers without expanding their magnitude.
func (s Scalar) CompareNumber(other Scalar) (int, bool) {
	if s.kind != Number || other.kind != Number {
		return 0, false
	}
	return s.num.compare(other.num), true
}

// Compare orders supported kinds, with exact numbers and lexical strings.
func (s Scalar) Compare(other Scalar) int {
	if s.kind != other.kind {
		return int(s.kind) - int(other.kind)
	}
	if s.kind == Number {
		return s.num.compare(other.num)
	}
	return strings.Compare(s.text, other.text)
}

// JSONLiteral returns a complete scalar literal without native-number conversion.
func (s Scalar) JSONLiteral() string {
	switch s.kind {
	case String:
		value, _ := json.Marshal(s.text)
		return string(value)
	case Number:
		return s.num.literal()
	case Boolean:
		return s.text
	case Null:
		return "null"
	default:
		return ""
	}
}

// DisplayLiteral also escapes terminal controls which JSON permits unescaped.
func (s Scalar) DisplayLiteral() string {
	if s.kind != String {
		return s.JSONLiteral()
	}
	var result strings.Builder
	for _, r := range s.JSONLiteral() {
		if r >= 0x7f && r <= 0x9f {
			fmt.Fprintf(&result, "\\u%04x", r)
		} else {
			result.WriteRune(r)
		}
	}
	return result.String()
}

// MarshalJSON emits a complete exact scalar token.
func (s Scalar) MarshalJSON() ([]byte, error) {
	if s.kind == Invalid {
		return nil, errInvalidScalar
	}
	return []byte(s.JSONLiteral()), nil
}

// MarshalYAML allocates a fresh scalar node.
func (s Scalar) MarshalYAML() (any, error) {
	if s.kind == Invalid {
		return nil, errInvalidScalar
	}
	node := &yaml.Node{Kind: yaml.ScalarNode, Value: s.JSONLiteral()}
	switch s.kind {
	case String:
		node.Tag, node.Value, node.Style = "!!str", s.text, yaml.DoubleQuotedStyle
	case Boolean:
		node.Tag = "!!bool"
	case Null:
		node.Tag = "!!null"
	case Number:
		node.Tag = "!!int"
		if strings.ContainsAny(node.Value, ".eE") {
			node.Tag = "!!float"
		}
	}
	return node, nil
}

// FromYAML reads a direct standard scalar only. It never follows aliases.
func FromYAML(node *yaml.Node) (Scalar, error) {
	if node == nil || node.Kind != yaml.ScalarNode {
		return Scalar{}, errUnsupportedScalar
	}
	switch node.ShortTag() {
	case "!!str":
		return Scalar{kind: String, text: node.Value}, nil
	case "!!null":
		if node.Value == "" || node.Value == "~" || strings.EqualFold(node.Value, "null") {
			return Scalar{kind: Null}, nil
		}
	case "!!bool":
		value := strings.ToLower(node.Value)
		if value == "true" || value == "false" {
			return Scalar{kind: Boolean, text: value}, nil
		}
	case "!!int":
		value, err := integerNumber(node.Value)
		if err == nil {
			return Scalar{kind: Number, num: value}, nil
		}
	case "!!float":
		value, err := decimalNumber(strings.ReplaceAll(node.Value, "_", ""))
		if err == nil {
			return Scalar{kind: Number, num: value}, nil
		}
	}
	return Scalar{}, errUnsupportedScalar
}
