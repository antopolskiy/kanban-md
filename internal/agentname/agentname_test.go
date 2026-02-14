package agentname

import (
	"strings"
	"testing"
)

func TestGenerate_Format(t *testing.T) {
	name, err := Generate()
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}

	parts := strings.Split(name, "-")
	if len(parts) != nameWordCount {
		t.Errorf("expected %d parts, got %d: %q", nameWordCount, len(parts), name)
	}
	for _, p := range parts {
		if !isValidWord(p) {
			t.Errorf("part %q is not a valid word (4-8 lowercase letters)", p)
		}
	}
}

func TestGenerate_Unique(t *testing.T) {
	const trials = 20
	seen := make(map[string]bool, trials)
	for range trials {
		name, err := Generate()
		if err != nil {
			t.Fatalf("Generate() error: %v", err)
		}
		seen[name] = true
	}
	// With hundreds of words, 20 names should have at least 15 unique.
	const minUnique = 15
	if len(seen) < minUnique {
		t.Errorf("expected at least %d unique names from %d trials, got %d", minUnique, trials, len(seen))
	}
}

func TestGenerate_FallbackWordList(t *testing.T) {
	// Point to a nonexistent dict file to force the fallback.
	orig := dictPath
	dictPath = "/nonexistent/dict/words"
	t.Cleanup(func() { dictPath = orig })

	name, err := Generate()
	if err != nil {
		t.Fatalf("Generate() with fallback error: %v", err)
	}

	parts := strings.Split(name, "-")
	if len(parts) != nameWordCount {
		t.Errorf("expected %d parts, got %d: %q", nameWordCount, len(parts), name)
	}
}

func TestEmbeddedWords_AllValid(t *testing.T) {
	for _, w := range embeddedWords() {
		if !isValidWord(w) {
			t.Errorf("embedded word %q is not valid (4-8 lowercase letters)", w)
		}
	}
}

func TestIsValidWord(t *testing.T) {
	tests := []struct {
		word string
		want bool
	}{
		{"calm", true},
		{"storm", true},
		{"birch", true},
		{"hi", false},
		{"wonderful", false},
		{"Hello", false},
		{"test1", false},
		{"well-done", false},
	}
	for _, tt := range tests {
		if got := isValidWord(tt.word); got != tt.want {
			t.Errorf("isValidWord(%q) = %v, want %v", tt.word, got, tt.want)
		}
	}
}
