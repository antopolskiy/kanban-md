package property

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

var (
	jsonNumberPattern = regexp.MustCompile(`^-?(?:0|[1-9][0-9]*)(?:\.[0-9]+)?(?:[eE][+-]?[0-9]+)?$`)
	numericLooking    = regexp.MustCompile(`^[+-]?(?:[0-9]|\.[0-9])`)
)

// ParseLiteral accepts JSON scalars or a trimmed bare string, never YAML syntax.
func ParseLiteral(raw string) (Scalar, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return Scalar{}, errors.New("empty property literal; use \"\" for an empty string")
	}
	if raw[0] == '"' {
		var value string
		if err := json.Unmarshal([]byte(raw), &value); err != nil {
			return Scalar{}, fmt.Errorf("invalid quoted property string: %w", err)
		}
		return Scalar{kind: String, text: value}, nil
	}
	switch raw {
	case "true", "false":
		return Scalar{kind: Boolean, text: raw}, nil
	case "null":
		return Scalar{kind: Null}, nil
	}
	if jsonNumberPattern.MatchString(raw) {
		value, err := decimalNumber(raw)
		return Scalar{kind: Number, num: value}, err
	}
	lower := strings.ToLower(raw)
	nonfinite := lower == ".inf" || lower == "+.inf" || lower == "-.inf" || lower == ".nan" || lower == "+.nan" || lower == "-.nan"
	if numericLooking.MatchString(raw) || nonfinite || strings.ContainsAny(raw[:1], "[{!&*'") {
		return Scalar{}, fmt.Errorf("invalid property literal %q; use a JSON scalar or quote it as a string", raw)
	}
	return Scalar{kind: String, text: raw}, nil
}

// Assignment is one parsed property key and scalar.
type Assignment struct {
	Key   string
	Value Scalar
}

// ParseAssignment splits a literal top-level key from a scalar literal.
func ParseAssignment(raw string) (Assignment, error) {
	key, value, found := strings.Cut(raw, "=")
	if !found {
		return Assignment{}, errors.New("property assignment must be KEY=LITERAL")
	}
	if err := ValidateKey(key); err != nil {
		return Assignment{}, err
	}
	scalar, err := ParseLiteral(value)
	return Assignment{Key: key, Value: scalar}, err
}
