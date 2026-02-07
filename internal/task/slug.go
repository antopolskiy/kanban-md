package task

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

const maxSlugLength = 50

var nonAlphanumeric = regexp.MustCompile(`[^a-z0-9]+`)

// GenerateSlug converts a title to a URL-friendly slug.
func GenerateSlug(title string) string {
	slug := strings.ToLower(title)
	slug = nonAlphanumeric.ReplaceAllString(slug, "-")
	slug = strings.Trim(slug, "-")

	if len(slug) > maxSlugLength {
		// Truncate at word boundary.
		truncated := slug[:maxSlugLength]
		// Only trim to last hyphen if we cut mid-word.
		if slug[maxSlugLength] != '-' {
			if idx := strings.LastIndex(truncated, "-"); idx > 0 {
				truncated = truncated[:idx]
			}
		}
		slug = strings.TrimRight(truncated, "-")
	}

	return slug
}

// GenerateFilename creates a task filename from an ID and slug.
func GenerateFilename(id int, slug string) string {
	padWidth := 3
	idStr := strconv.Itoa(id)
	if len(idStr) > padWidth {
		padWidth = len(idStr)
	}
	return fmt.Sprintf("%0*d-%s.md", padWidth, id, slug)
}
