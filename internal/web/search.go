package web

import (
	"strconv"
	"strings"

	"github.com/antopolskiy/kanban-md/internal/task"
)

// matchesBoardSearch matches the TUI board filter: a case-insensitive substring
// of the title or one tag, or a #id prefix. A trailing space after the digits
// requires an exact id. The body is not searched.
func matchesBoardSearch(t *task.Task, query string) bool {
	if strings.HasPrefix(query, "#") {
		return matchesIDSearch(t.ID, query[1:])
	}
	needle := strings.TrimSpace(strings.ToLower(query))
	if needle == "" {
		return true
	}
	if strings.Contains(strings.ToLower(t.Title), needle) {
		return true
	}
	for _, tag := range t.Tags {
		if strings.Contains(strings.ToLower(tag), needle) {
			return true
		}
	}
	return false
}

func matchesIDSearch(id int, rest string) bool {
	exact := strings.HasSuffix(rest, " ")
	digits := strings.TrimSpace(rest)
	if digits == "" {
		return true
	}
	text := strconv.Itoa(id)
	if exact {
		return text == digits
	}
	return strings.HasPrefix(text, digits)
}

func hasTag(t *task.Task, want string) bool {
	want = strings.ToLower(strings.TrimSpace(want))
	if want == "" {
		return true
	}
	for _, tag := range t.Tags {
		if strings.ToLower(tag) == want {
			return true
		}
	}
	return false
}
