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
	// Empty means the agent only supports global skills.
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
		ProjectDir:  "", // global-only
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

// GlobalOnly returns true if the agent only supports global skill installation.
func (a *Agent) GlobalOnly() bool {
	return a.ProjectDir == ""
}

// ProjectPath returns the absolute path for a skill directory for this agent
// within the given project root. Returns empty string for global-only agents.
func (a *Agent) ProjectPath(projectRoot string) string {
	if a.ProjectDir == "" {
		return ""
	}
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

// SkillPath returns the install path for this agent — global path for global-only
// agents, otherwise project path unless global is explicitly requested.
func (a *Agent) SkillPath(projectRoot string, global bool) string {
	if global || a.GlobalOnly() {
		return a.GlobalPath()
	}
	return a.ProjectPath(projectRoot)
}

// DetectAgents returns agents whose project-level skill directory parent exists
// in the given project root. For example, if .claude/ exists, Claude Code is detected.
// Global-only agents are not detected at the project level.
func DetectAgents(projectRoot string) []Agent {
	var detected []Agent
	for _, a := range agents {
		if a.GlobalOnly() {
			continue
		}
		parentDir := filepath.Dir(a.ProjectDir)
		absParent := filepath.Join(projectRoot, parentDir)
		if info, err := os.Stat(absParent); err == nil && info.IsDir() {
			detected = append(detected, a)
		}
	}
	return detected
}
