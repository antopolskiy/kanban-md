package property

import (
	"fmt"
	"regexp"
	"strings"
)

var keyPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_.-]*$`)

var reserved = map[string]bool{
	"id": true, "title": true, "status": true, "priority": true, "created": true, "updated": true,
	"started": true, "completed": true, "assignee": true, "tags": true, "due": true, "estimate": true,
	"parent": true, "depends_on": true, "blocked": true, "block_reason": true, "claimed_by": true,
	"claimed_at": true, "class": true, "body": true, "file": true,
}

// ValidateKey accepts literal top-level foreign identifiers only.
func ValidateKey(key string) error {
	if !keyPattern.MatchString(key) {
		return fmt.Errorf("invalid property key %q: use a literal identifier", key)
	}
	if reserved[key] {
		return fmt.Errorf("property key %q is owned by kanban-md; use its existing field option", key)
	}
	return nil
}

// SelectorKey distinguishes dynamic property selectors from canonical selectors.
func SelectorKey(selector string) (string, bool, error) {
	if !strings.HasPrefix(selector, "property:") {
		return "", false, nil
	}
	key := strings.TrimPrefix(selector, "property:")
	return key, true, ValidateKey(key)
}
