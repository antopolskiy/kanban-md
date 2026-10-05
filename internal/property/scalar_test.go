package property

import (
	"encoding/json"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

func scalar(t *testing.T, literal string) Scalar {
	t.Helper()
	value, err := ParseLiteral(literal)
	if err != nil {
		t.Fatal(err)
	}
	return value
}

func TestExactNumberComparison(t *testing.T) {
	ordered := []string{"-1e999999999999999999999", "-2", "-0.1", "0", "0.00001", "0.1", "1", "1.000000000000000000001", "9007199254740992", "9007199254740993", "18446744073709551617", "1e999999999999999999999"}
	for i, a := range ordered {
		for j, b := range ordered {
			got, ok := scalar(t, a).CompareNumber(scalar(t, b))
			want := 0
			if i < j {
				want = -1
			}
			if i > j {
				want = 1
			}
			if !ok || got != want {
				t.Fatalf("compare(%s,%s)=%d,%t want %d", a, b, got, ok, want)
			}
		}
	}
	for _, pair := range [][2]string{{"20", "20.0"}, {"20", "2e1"}, {"0", "-0.00"}, {"1e10000000000000000000", "10e9999999999999999999"}} {
		if !scalar(t, pair[0]).Equal(scalar(t, pair[1])) {
			t.Errorf("not equal: %v", pair)
		}
	}
}

func TestYAMLNumericSpellingAndJSON(t *testing.T) {
	for _, tc := range []struct{ raw, want string }{{"0x10", "16"}, {"0b10000", "16"}, {"0o20", "16"}, {"010", "8"}, {"1_000", "1000"}, {"18446744073709551617", "18446744073709551617"}, {".5", "0.5"}, {"20.", "20"}, {"1.000000000000000000001", "1.000000000000000000001"}} {
		t.Run(tc.raw, func(t *testing.T) {
			var doc yaml.Node
			if err := yaml.Unmarshal([]byte(tc.raw), &doc); err != nil {
				t.Fatal(err)
			}
			value, err := FromYAML(doc.Content[0])
			if err != nil {
				t.Fatal(err)
			}
			if value.JSONLiteral() != tc.want {
				t.Fatalf("literal=%s want %s", value.JSONLiteral(), tc.want)
			}
			data, err := json.Marshal(value)
			if err != nil || !json.Valid(data) {
				t.Fatalf("JSON %s: %v", data, err)
			}
		})
	}
	var doc yaml.Node
	if err := yaml.Unmarshal([]byte(`"20"`), &doc); err != nil {
		t.Fatal(err)
	}
	value, err := FromYAML(doc.Content[0])
	if err != nil || value.Kind() != String || value.Equal(scalar(t, "20")) {
		t.Fatal("quoted numeric string coerced")
	}
}

func TestLiteralKindsAndBoundaries(t *testing.T) {
	for _, tc := range []struct {
		raw  string
		kind Kind
	}{{"story", String}, {`"20"`, String}, {`"true"`, String}, {`""`, String}, {"TRUE", String}, {"~", String}, {"20", Number}, {"false", Boolean}, {"null", Null}} {
		if got := scalar(t, tc.raw).Kind(); got != tc.kind {
			t.Errorf("%s kind=%d", tc.raw, got)
		}
	}
	for _, raw := range []string{"", "01", "+20", ".5", "20.", "0x10", "1_000", ".inf", "-.inf", ".nan", "[1]", "{a: 1}", "!foo bar", "&x 1", "*x", "'story'", `"x" trailing`} {
		if _, err := ParseLiteral(raw); err == nil {
			t.Errorf("accepted %q", raw)
		}
	}
	if scalar(t, `"null"`).Equal(scalar(t, "null")) {
		t.Fatal("typed values collided")
	}
}

func TestUnsupportedAndSafeDisplay(t *testing.T) {
	for _, raw := range []string{"[1]", "{x: 1}", "2026-10-05", ".inf", ".nan", "!custom value"} {
		var doc yaml.Node
		if err := yaml.Unmarshal([]byte(raw), &doc); err != nil {
			t.Fatal(err)
		}
		if _, err := FromYAML(doc.Content[0]); err == nil {
			t.Errorf("accepted %s", raw)
		}
	}
	value := scalar(t, `"a\n\t\u001b[31m\u009bX\u2028"`)
	if got := value.DisplayLiteral(); strings.ContainsAny(got, "\n\t\x1b\u009b\u2028") {
		t.Fatalf("unsafe display %q", got)
	}
	for _, key := range []string{"status", "body", "file", "x y", "x:y", "x=1", ""} {
		if ValidateKey(key) == nil {
			t.Errorf("accepted key %q", key)
		}
	}
	if ValidateKey("session.id") != nil {
		t.Fatal("literal dotted key rejected")
	}
}
