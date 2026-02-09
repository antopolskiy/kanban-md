package skill

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

const (
	fileMode = 0o600
	dirMode  = 0o750
)

// Install writes the embedded skill files for the given skill to the target
// directory, injecting a version comment. The target directory is the agent's
// skill base directory (e.g., .claude/skills/). Files are written to
// targetDir/<skillName>/.
func Install(skillName, targetDir, ver string) error {
	skillDir := filepath.Join("skills", skillName)

	// Read the embedded skill tree.
	entries, err := fs.ReadDir(skillsFS, skillDir)
	if err != nil {
		return fmt.Errorf("reading embedded skill %q: %w", skillName, err)
	}

	outputBase := filepath.Join(targetDir, skillName)
	if err := os.MkdirAll(outputBase, dirMode); err != nil {
		return fmt.Errorf("creating skill directory: %w", err)
	}

	// Walk the embedded tree and write all files.
	return fs.WalkDir(skillsFS, skillDir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		_ = entries // used above for validation

		// Compute the relative path from the skill root.
		relPath, err := filepath.Rel(skillDir, path)
		if err != nil {
			return err
		}

		destPath := filepath.Join(outputBase, relPath)

		if d.IsDir() {
			return os.MkdirAll(destPath, dirMode)
		}

		data, err := fs.ReadFile(skillsFS, path)
		if err != nil {
			return fmt.Errorf("reading embedded file %q: %w", path, err)
		}

		// Inject version comment into SKILL.md files.
		if d.Name() == "SKILL.md" {
			data = injectVersionComment(data, ver)
		}

		return os.WriteFile(destPath, data, fileMode)
	})
}

// injectVersionComment inserts the version comment line after the closing
// frontmatter delimiter (---). If no frontmatter is found, it prepends.
func injectVersionComment(data []byte, ver string) []byte {
	content := string(data)
	comment := VersionComment(ver)

	// Find the closing --- of frontmatter.
	// Frontmatter starts with --- on line 1 and ends with --- on a later line.
	lines := strings.SplitAfter(content, "\n")
	const minFrontmatterLines = 2
	if len(lines) < minFrontmatterLines || strings.TrimSpace(lines[0]) != "---" {
		// No frontmatter — prepend the comment.
		return []byte(comment + "\n" + content)
	}

	// Find the closing ---.
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" || strings.TrimSpace(lines[i]) == "---\n" {
			// Insert comment after this line.
			before := strings.Join(lines[:i+1], "")
			after := strings.Join(lines[i+1:], "")
			return []byte(before + comment + "\n" + after)
		}
	}

	// No closing --- found — prepend.
	return []byte(comment + "\n" + content)
}

// FindInstalledSkills scans the given base directory for installed kanban-md
// skills (directories containing SKILL.md with a version comment).
// Returns a map of skill name → SKILL.md path.
func FindInstalledSkills(baseDir string) map[string]string {
	result := make(map[string]string)
	for _, s := range AvailableSkills {
		skillMD := filepath.Join(baseDir, s.Name, "SKILL.md")
		if v := InstalledVersion(skillMD); v != "" {
			result[s.Name] = skillMD
		}
	}
	return result
}
