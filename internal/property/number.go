package property

import (
	"errors"
	"math/big"
	"regexp"
	"strings"
)

var decimalPattern = regexp.MustCompile(`^([+-]?)([0-9]+(?:\.[0-9]*)?|\.[0-9]+)(?:[eE]([+-]?[0-9]+))?$`)

const decimalRadix = 10

// Only immutable strings are stored. big.Int temporaries never escape a method.
type number struct {
	negative bool
	digits   string
	exponent string
}

func integerNumber(raw string) (number, error) {
	value, ok := new(big.Int).SetString(strings.ReplaceAll(raw, "_", ""), 0)
	if !ok {
		return number{}, errors.New("invalid integer")
	}
	return decimalNumber(value.String())
}

func decimalNumber(raw string) (number, error) {
	parts := decimalPattern.FindStringSubmatch(raw)
	if parts == nil {
		return number{}, errors.New("invalid finite decimal number")
	}
	exponent := new(big.Int)
	if parts[3] != "" {
		exponent.SetString(parts[3], decimalRadix)
	}
	mantissa := parts[2]
	if dot := strings.IndexByte(mantissa, '.'); dot >= 0 {
		exponent.Sub(exponent, big.NewInt(int64(len(mantissa)-dot-1)))
		mantissa = strings.ReplaceAll(mantissa, ".", "")
	}
	digits := strings.TrimLeft(mantissa, "0")
	if digits == "" {
		return number{digits: "0", exponent: "0"}, nil
	}
	trimmed := strings.TrimRight(digits, "0")
	exponent.Add(exponent, big.NewInt(int64(len(digits)-len(trimmed))))
	return number{negative: parts[1] == "-", digits: trimmed, exponent: exponent.String()}, nil
}

func (n number) compare(other number) int {
	if n.negative != other.negative {
		if n.negative {
			return -1
		}
		return 1
	}
	result := n.compareMagnitude(other)
	if n.negative {
		return -result
	}
	return result
}

func (n number) compareMagnitude(other number) int {
	if n.digits == "0" || other.digits == "0" {
		if n.digits == other.digits {
			return 0
		}
		if n.digits == "0" {
			return -1
		}
		return 1
	}
	a, _ := new(big.Int).SetString(n.exponent, decimalRadix)
	b, _ := new(big.Int).SetString(other.exponent, decimalRadix)
	a.Add(a, big.NewInt(int64(len(n.digits))))
	b.Add(b, big.NewInt(int64(len(other.digits))))
	if result := a.Cmp(b); result != 0 {
		return result
	}
	for i := 0; i < max(len(n.digits), len(other.digits)); i++ {
		ac, bc := byte('0'), byte('0')
		if i < len(n.digits) {
			ac = n.digits[i]
		}
		if i < len(other.digits) {
			bc = other.digits[i]
		}
		if ac < bc {
			return -1
		}
		if ac > bc {
			return 1
		}
	}
	return 0
}

func (n number) literal() string {
	prefix := ""
	if n.negative {
		prefix = "-"
	}
	exponent, _ := new(big.Int).SetString(n.exponent, decimalRadix)
	const maxExpansion = 128
	if exponent.IsInt64() {
		e := exponent.Int64()
		if e >= 0 && e <= maxExpansion && int64(len(n.digits))+e <= maxExpansion {
			return prefix + n.digits + strings.Repeat("0", int(e))
		}
		if e < 0 && e >= -maxExpansion {
			point := int64(len(n.digits)) + e
			if point > 0 {
				return prefix + n.digits[:point] + "." + n.digits[point:]
			}
			return prefix + "0." + strings.Repeat("0", int(-point)) + n.digits
		}
	}
	return prefix + n.digits + "e" + n.exponent
}
