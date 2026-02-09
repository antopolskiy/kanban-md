package skill

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

const (
	versionPrefix = "<!-- kanban-md-skill-version: "
	versionSuffix = " -->"
)

// InstalledVersion reads the version comment from an installed SKILL.md file.
// Returns empty string if not found or file doesn't exist.
func InstalledVersion(path string) string {
	f, err := os.Open(path) //nolint:gosec // path from trusted source (agent skill directory)
	if err != nil {
		return ""
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	// Scan enough lines to find the version comment past any frontmatter.
	const maxLines = 30
	for i := 0; i < maxLines && scanner.Scan(); i++ {
		line := scanner.Text()
		if strings.HasPrefix(line, versionPrefix) && strings.HasSuffix(line, versionSuffix) {
			return strings.TrimSuffix(strings.TrimPrefix(line, versionPrefix), versionSuffix)
		}
	}
	return ""
}

// IsOutdated checks if the installed skill at path has a different version
// than currentVersion. Returns false if the skill is not installed.
func IsOutdated(path string, currentVersion string) bool {
	installed := InstalledVersion(path)
	if installed == "" {
		return false
	}
	return installed != currentVersion
}

// VersionComment returns the HTML comment line for embedding in SKILL.md.
func VersionComment(ver string) string {
	return fmt.Sprintf("%s%s%s", versionPrefix, ver, versionSuffix)
}
