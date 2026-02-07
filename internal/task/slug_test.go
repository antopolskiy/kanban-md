package task

import "testing"

func TestGenerateSlug(t *testing.T) {
	tests := []struct {
		title string
		want  string
	}{
		{"Set up database", "set-up-database"},
		{"Fix Login Redirect Bug", "fix-login-redirect-bug"},
		{"Hello, World!", "hello-world"},
		{"  leading spaces  ", "leading-spaces"},
		{"special!!chars##here", "special-chars-here"},
		{"UPPERCASE TITLE", "uppercase-title"},
		{"a-already-slugged", "a-already-slugged"},
		{"This is a very long title that should be truncated at a word boundary to fit within the limit", "this-is-a-very-long-title-that-should-be-truncated"},
		{"A title that gets cut mid word abcdefghijklmnopqrst and more", "a-title-that-gets-cut-mid-word"},
		{"", ""},
	}
	for _, tt := range tests {
		got := GenerateSlug(tt.title)
		if got != tt.want {
			t.Errorf("GenerateSlug(%q) = %q, want %q", tt.title, got, tt.want)
		}
	}
}

func TestGenerateFilename(t *testing.T) {
	tests := []struct {
		id   int
		slug string
		want string
	}{
		{1, "setup-database", "001-setup-database.md"},
		{42, "fix-bug", "042-fix-bug.md"},
		{999, "last-three-digit", "999-last-three-digit.md"},
		{1000, "four-digits", "1000-four-digits.md"},
	}
	for _, tt := range tests {
		got := GenerateFilename(tt.id, tt.slug)
		if got != tt.want {
			t.Errorf("GenerateFilename(%d, %q) = %q, want %q", tt.id, tt.slug, got, tt.want)
		}
	}
}
