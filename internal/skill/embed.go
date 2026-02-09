// Package skill provides embedded agent skills and installation utilities.
package skill

import "embed"

//go:embed skills
var skillsFS embed.FS

// Info describes an installable skill.
type Info struct {
	// Name is the directory name and identifier for the skill.
	Name string
	// Description is a short human-readable summary shown in menus.
	Description string
}

// AvailableSkills lists all embedded skills with their descriptions.
var AvailableSkills = []Info{
	{
		Name:        "kanban-md",
		Description: "Task management commands, workflows, and decision trees",
	},
	{
		Name:        "kanban-based-development",
		Description: "Multi-agent parallel development workflow with claims and worktrees",
	},
}

// ReadEmbeddedSkill reads the SKILL.md content for the named skill.
func ReadEmbeddedSkill(name string) ([]byte, error) {
	return skillsFS.ReadFile("skills/" + name + "/SKILL.md")
}

// Names returns just the names of all available skills.
func Names() []string {
	names := make([]string, len(AvailableSkills))
	for i, s := range AvailableSkills {
		names[i] = s.Name
	}
	return names
}
