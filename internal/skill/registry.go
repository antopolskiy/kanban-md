package skill

import (
	"os"
	"path/filepath"
)

// Agent describes an AI coding agent and its skill directory conventions.
type Agent struct {
	// Name is the identifier used in --agent flags.
	Name string
	// DisplayName is the human-readable name shown in menus.
	DisplayName string
	// ProjectDir is the skill directory relative to the project root.
	ProjectDir string
	// GlobalDir is the skill directory relative to the user's home directory.
	GlobalDir string
}

// agents is the registry of supported AI coding agents.
var agents = []Agent{
	{
		Name:        "claude",
		DisplayName: "Claude Code",
		ProjectDir:  ".claude/skills",
		GlobalDir:   ".claude/skills",
	},
	{
		Name:        "codex",
		DisplayName: "Codex",
		ProjectDir:  ".agents/skills",
		GlobalDir:   ".codex/skills",
	},
	{
		Name:        "cursor",
		DisplayName: "Cursor",
		ProjectDir:  ".cursor/skills",
		GlobalDir:   ".cursor/skills",
	},
	{
		Name:        "openclaw",
		DisplayName: "OpenClaw",
		ProjectDir:  "skills",
		GlobalDir:   ".openclaw/skills",
	},
}

// Agents returns all supported agents.
func Agents() []Agent {
	return agents
}

// AgentByName returns the agent with the given name, or nil if not found.
func AgentByName(name string) *Agent {
	for i := range agents {
		if agents[i].Name == name {
			return &agents[i]
		}
	}
	return nil
}

// AllAgentNames returns the names of all supported agents.
func AllAgentNames() []string {
	names := make([]string, len(agents))
	for i, a := range agents {
		names[i] = a.Name
	}
	return names
}

// ProjectPath returns the absolute path for a skill directory for this agent
// within the given project root.
func (a *Agent) ProjectPath(projectRoot string) string {
	return filepath.Join(projectRoot, a.ProjectDir)
}

// GlobalPath returns the absolute path for the global skill directory.
func (a *Agent) GlobalPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, a.GlobalDir)
}

// DetectAgents returns agents whose project-level skill directory parent exists
// in the given project root. For example, if .claude/ exists, Claude Code is detected.
func DetectAgents(projectRoot string) []Agent {
	var detected []Agent
	for _, a := range agents {
		parentDir := filepath.Dir(a.ProjectDir)
		if parentDir == "." {
			// OpenClaw uses "skills" at root — always detected.
			detected = append(detected, a)
			continue
		}
		absParent := filepath.Join(projectRoot, parentDir)
		if info, err := os.Stat(absParent); err == nil && info.IsDir() {
			detected = append(detected, a)
		}
	}
	return detected
}
