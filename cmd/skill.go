package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/antopolskiy/kanban-md/internal/output"
	"github.com/antopolskiy/kanban-md/internal/skill"
)

var skillCmd = &cobra.Command{
	Use:   "skill",
	Short: "Manage agent skills",
	Long:  `Install, update, check, and show embedded agent skills for AI coding assistants.`,
}

var skillInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Install agent skills",
	Long: `Installs kanban-md skills for AI coding agents (Claude Code, Codex, Cursor, OpenClaw).
In interactive mode, shows a multi-select menu for agents and skills.
In non-interactive mode (piped/CI), installs all skills for all detected agents.`,
	RunE: runSkillInstall,
}

var skillUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update installed skills to current version",
	Long:  `Finds all installed kanban-md skills and updates any that are outdated.`,
	RunE:  runSkillUpdate,
}

var skillCheckCmd = &cobra.Command{
	Use:   "check",
	Short: "Check if installed skills are up to date",
	Long:  `Compares installed skill versions against the current CLI version.`,
	RunE:  runSkillCheck,
}

var skillShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Print embedded skill content to stdout",
	Long:  `Displays the embedded skill content for inspection or piping.`,
	RunE:  runSkillShow,
}

func init() {
	skillInstallCmd.Flags().StringSlice("agent", nil, "agent(s) to install for (claude, codex, cursor, openclaw)")
	skillInstallCmd.Flags().StringSlice("skill", nil, "skill(s) to install (kanban-md, kanban-based-development)")
	skillInstallCmd.Flags().Bool("global", false, "install to user-level (global) skill directory")
	skillInstallCmd.Flags().Bool("force", false, "overwrite existing skills without checking version")
	skillInstallCmd.Flags().String("path", "", "install skills to a specific directory (skips agent selection)")

	skillUpdateCmd.Flags().StringSlice("agent", nil, "agent(s) to update")
	skillUpdateCmd.Flags().Bool("global", false, "update user-level (global) skills")

	skillCheckCmd.Flags().StringSlice("agent", nil, "agent(s) to check")
	skillCheckCmd.Flags().Bool("global", false, "check user-level (global) skills")

	skillShowCmd.Flags().String("skill", "", "skill to show (kanban-md or kanban-based-development)")

	skillCmd.AddCommand(skillInstallCmd)
	skillCmd.AddCommand(skillUpdateCmd)
	skillCmd.AddCommand(skillCheckCmd)
	skillCmd.AddCommand(skillShowCmd)
	rootCmd.AddCommand(skillCmd)
}

func runSkillInstall(cmd *cobra.Command, _ []string) error {
	global, _ := cmd.Flags().GetBool("global")
	force, _ := cmd.Flags().GetBool("force")
	agentFilter, _ := cmd.Flags().GetStringSlice("agent")
	skillFilter, _ := cmd.Flags().GetStringSlice("skill")
	pathFlag, _ := cmd.Flags().GetString("path")

	// Determine which skills to install.
	selectedSkills, err := resolveSkills(cmd, skillFilter)
	if err != nil {
		return err
	}
	if len(selectedSkills) == 0 {
		output.Messagef(os.Stdout, "No skills selected.")
		return nil
	}

	// --path mode: install directly to the given directory, skip agent selection.
	if pathFlag != "" {
		return installToPath(pathFlag, selectedSkills, force)
	}

	projectRoot, err := findProjectRoot()
	if err != nil && !global {
		return fmt.Errorf("finding project root: %w", err)
	}

	// Determine which agents to install for.
	selectedAgents := resolveAgents(agentFilter, projectRoot, global)
	if len(selectedAgents) == 0 {
		output.Messagef(os.Stdout, "No agents selected.")
		return nil
	}

	// Install.
	var installed int
	for _, agent := range selectedAgents {
		baseDir := agent.SkillPath(projectRoot, global)

		for _, s := range selectedSkills {
			destPath := filepath.Join(baseDir, s.Name, "SKILL.md")
			displayPath := relativePath(projectRoot, destPath)

			if !force {
				if v := skill.InstalledVersion(destPath); v == version {
					output.Messagef(os.Stdout, "  %s — already at %s (skipped)", displayPath, version)
					continue
				}
			}

			if err := skill.Install(s.Name, baseDir, version); err != nil {
				return fmt.Errorf("installing %s for %s: %w", s.Name, agent.DisplayName, err)
			}
			output.Messagef(os.Stdout, "  %s (%s)", displayPath, version)
			installed++
		}
	}

	if installed > 0 {
		output.Messagef(os.Stdout, "Installed %d skill(s).", installed)
	} else {
		output.Messagef(os.Stdout, "All skills are already up to date.")
	}
	return nil
}

// installToPath installs skills directly to the given directory path.
func installToPath(dir string, skills []skill.Info, force bool) error {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("resolving path: %w", err)
	}

	var installed int
	for _, s := range skills {
		destPath := filepath.Join(absDir, s.Name, "SKILL.md")

		if !force {
			if v := skill.InstalledVersion(destPath); v == version {
				output.Messagef(os.Stdout, "  %s — already at %s (skipped)", destPath, version)
				continue
			}
		}

		if err := skill.Install(s.Name, absDir, version); err != nil {
			return fmt.Errorf("installing %s to %s: %w", s.Name, absDir, err)
		}
		output.Messagef(os.Stdout, "  %s (%s)", destPath, version)
		installed++
	}

	if installed > 0 {
		output.Messagef(os.Stdout, "Installed %d skill(s).", installed)
	} else {
		output.Messagef(os.Stdout, "All skills are already up to date.")
	}
	return nil
}

func runSkillUpdate(cmd *cobra.Command, _ []string) error {
	global, _ := cmd.Flags().GetBool("global")
	agentFilter, _ := cmd.Flags().GetStringSlice("agent")

	projectRoot, err := findProjectRoot()
	if err != nil && !global {
		return fmt.Errorf("finding project root: %w", err)
	}

	agents := resolveAgentList(agentFilter)
	var updated int

	for _, agent := range agents {
		baseDir := agent.SkillPath(projectRoot, global)

		installed := skill.FindInstalledSkills(baseDir)
		for skillName, skillPath := range installed {
			if skill.IsOutdated(skillPath, version) {
				oldVer := skill.InstalledVersion(skillPath)
				if err := skill.Install(skillName, baseDir, version); err != nil {
					return fmt.Errorf("updating %s for %s: %w", skillName, agent.DisplayName, err)
				}
				output.Messagef(os.Stdout, "  %s/%s — updated (%s → %s)", agent.DisplayName, skillName, oldVer, version)
				updated++
			}
		}
	}

	if updated > 0 {
		output.Messagef(os.Stdout, "Updated %d skill(s).", updated)
	} else {
		output.Messagef(os.Stdout, "All skills are up to date.")
	}
	return nil
}

func runSkillCheck(cmd *cobra.Command, _ []string) error {
	global, _ := cmd.Flags().GetBool("global")
	agentFilter, _ := cmd.Flags().GetStringSlice("agent")

	projectRoot, err := findProjectRoot()
	if err != nil && !global {
		return fmt.Errorf("finding project root: %w", err)
	}

	agents := resolveAgentList(agentFilter)
	var anyOutdated bool
	var anyFound bool

	for _, agent := range agents {
		baseDir := agent.SkillPath(projectRoot, global)

		installed := skill.FindInstalledSkills(baseDir)
		for skillName, skillPath := range installed {
			anyFound = true
			installedVer := skill.InstalledVersion(skillPath)
			if skill.IsOutdated(skillPath, version) {
				anyOutdated = true
				output.Messagef(os.Stdout, "  ✗ %s/%s (%s → %s)", agent.DisplayName, skillName, installedVer, version)
			} else {
				output.Messagef(os.Stdout, "  ✓ %s/%s (%s)", agent.DisplayName, skillName, installedVer)
			}
		}
	}

	if !anyFound {
		output.Messagef(os.Stdout, "No kanban-md skills installed. Run: kanban-md skill install")
		return nil
	}

	if anyOutdated {
		output.Messagef(os.Stdout, "Run: kanban-md skill update")
		return &exitCodeError{code: 1}
	}

	output.Messagef(os.Stdout, "All skills are up to date.")
	return nil
}

func runSkillShow(cmd *cobra.Command, _ []string) error {
	filter, _ := cmd.Flags().GetString("skill")

	for _, s := range skill.AvailableSkills {
		if filter != "" && s.Name != filter {
			continue
		}

		content, err := skill.ReadEmbeddedSkill(s.Name)
		if err != nil {
			return err
		}

		if filter == "" {
			fmt.Fprintf(os.Stdout, "=== %s ===\n\n", s.Name)
		}
		fmt.Fprint(os.Stdout, string(content))
		if filter == "" {
			fmt.Fprintln(os.Stdout)
		}
	}
	return nil
}

// resolveAgents determines which agents to install for, using flags or interactive selection.
func resolveAgents(filter []string, projectRoot string, global bool) []skill.Agent {
	// If explicit --agent flag, use those.
	if len(filter) > 0 {
		return resolveAgentList(filter)
	}

	// Non-interactive: use all detected agents.
	if !isInteractive() {
		if global {
			return skill.Agents()
		}
		return skill.DetectAgents(projectRoot)
	}

	// Interactive: show all agents, pre-select detected ones.
	// This lets users install for agents not yet detected (e.g. OpenClaw whose
	// "skills/" directory doesn't exist until the first install).
	allAgents := skill.Agents()
	detected := make(map[string]bool)
	if !global {
		for _, a := range skill.DetectAgents(projectRoot) {
			detected[a.Name] = true
		}
	}

	items := make([]menuItem, len(allAgents))
	for i, a := range allAgents {
		var dir string
		switch {
		case global || a.GlobalOnly():
			dir = a.GlobalPath()
		default:
			dir = a.ProjectDir + "/"
		}
		// Pre-select: global-only agents are always shown (they have no
		// project detection), detected agents are pre-checked.
		items[i] = menuItem{
			label:       a.DisplayName,
			description: dir,
			selected:    global || a.GlobalOnly() || detected[a.Name],
		}
	}

	selected := multiSelect("Select agents to install for:", items)
	var result []skill.Agent
	for _, idx := range selected {
		result = append(result, allAgents[idx])
	}
	return result
}

// resolveSkills determines which skills to install, using flags or interactive selection.
func resolveSkills(_ *cobra.Command, filter []string) ([]skill.Info, error) {
	all := skill.AvailableSkills

	// If explicit --skill flag, use those.
	if len(filter) > 0 {
		var result []skill.Info
		for _, name := range filter {
			found := false
			for _, s := range all {
				if s.Name == name {
					result = append(result, s)
					found = true
					break
				}
			}
			if !found {
				return nil, fmt.Errorf("unknown skill: %q (available: %s)", name, strings.Join(skill.Names(), ", "))
			}
		}
		return result, nil
	}

	// Non-interactive: use all skills.
	if !isInteractive() {
		return all, nil
	}

	// Interactive: show multi-select.
	items := make([]menuItem, len(all))
	for i, s := range all {
		items[i] = menuItem{
			label:       s.Name,
			description: s.Description,
			selected:    true,
		}
	}

	selected := multiSelect("Select skills to install:", items)
	var result []skill.Info
	for _, idx := range selected {
		result = append(result, all[idx])
	}
	return result, nil
}

// resolveAgentList converts agent name strings to Agent structs. Unknown names are ignored.
func resolveAgentList(names []string) []skill.Agent {
	if len(names) == 0 {
		return skill.Agents()
	}
	var result []skill.Agent
	for _, name := range names {
		if a := skill.AgentByName(name); a != nil {
			result = append(result, *a)
		}
	}
	return result
}

// findProjectRoot finds the git root or current working directory.
func findProjectRoot() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	// Walk up looking for .git directory (similar to config.FindDir).
	dir := cwd
	for {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached filesystem root — use cwd.
			return cwd, nil
		}
		dir = parent
	}
}

// isInteractive returns true if stdin is a real terminal (not /dev/null, NUL, or a pipe).
func isInteractive() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
}

// menuItem represents an item in a multi-select menu.
type menuItem struct {
	label       string
	description string
	selected    bool
}

// multiSelect displays an interactive multi-select menu using bubbletea
// and returns the indices of selected items. Navigate with j/k or arrows,
// space to toggle, enter to confirm.
func multiSelect(prompt string, items []menuItem) []int {
	m := selectModel{
		prompt: prompt,
		items:  items,
	}

	p := tea.NewProgram(m, tea.WithOutput(os.Stderr))
	result, err := p.Run()
	if err != nil {
		// Fallback: return all items selected.
		indices := make([]int, len(items))
		for i := range indices {
			indices[i] = i
		}
		return indices
	}

	final := result.(selectModel)
	if final.canceled {
		return nil
	}
	var selected []int
	for i, item := range final.items {
		if item.selected {
			selected = append(selected, i)
		}
	}
	return selected
}

// selectModel is a bubbletea model for the multi-select menu.
type selectModel struct {
	prompt   string
	items    []menuItem
	cursor   int
	done     bool
	canceled bool
}

var (
	selectActiveStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	selectDimStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	selectCheckStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
)

func (m selectModel) Init() tea.Cmd { return nil }

func (m selectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.done {
		return m, nil
	}

	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	switch keyMsg.String() {
	case "j", "down":
		if m.cursor < len(m.items)-1 {
			m.cursor++
		}
	case "k", "up":
		if m.cursor > 0 {
			m.cursor--
		}
	case " ":
		m.items[m.cursor].selected = !m.items[m.cursor].selected
	case "enter":
		m.done = true
		return m, tea.Quit
	case "q", "esc", "ctrl+c":
		m.done = true
		m.canceled = true
		return m, tea.Quit
	}
	return m, nil
}

func (m selectModel) View() string {
	if m.done {
		return ""
	}

	var b strings.Builder
	b.WriteString(m.prompt + "\n")

	for i, item := range m.items {
		check := "✓"
		checkRendered := selectCheckStyle.Render(check)
		if !item.selected {
			check = " "
			checkRendered = check
		}

		cursor := " "
		if i == m.cursor {
			cursor = "›"
		}

		label := item.label
		desc := selectDimStyle.Render(item.description)
		if i == m.cursor {
			label = selectActiveStyle.Render(label)
		}

		b.WriteString(fmt.Sprintf("  %s [%s] %s — %s\n", cursor, checkRendered, label, desc))
	}

	b.WriteString(selectDimStyle.Render("\n  ↑/↓ navigate • space toggle • enter confirm • esc cancel\n"))
	return b.String()
}

// relativePath returns a path relative to root, or the absolute path if it cannot be made relative.
func relativePath(root, abs string) string {
	rel, err := filepath.Rel(root, abs)
	if err != nil {
		return abs
	}
	return rel
}

// exitCodeError is a simple error that carries an exit code.
type exitCodeError struct {
	code int
}

func (e *exitCodeError) Error() string {
	return fmt.Sprintf("exit code %d", e.code)
}

// CheckSkillStaleness checks installed skills and prints a warning if outdated.
// Called from PersistentPreRun for every command. Must be fast.
func CheckSkillStaleness(projectRoot string) {
	if version == "dev" {
		return
	}

	// Only check Claude Code (most common) for speed.
	claude := skill.AgentByName("claude")
	if claude == nil {
		return
	}

	baseDir := claude.ProjectPath(projectRoot)
	installed := skill.FindInstalledSkills(baseDir)
	for _, skillPath := range installed {
		if skill.IsOutdated(skillPath, version) {
			oldVer := skill.InstalledVersion(skillPath)
			fmt.Fprintf(os.Stderr, "hint: kanban-md skill outdated (%s → %s), run: kanban-md skill update\n", oldVer, version)
			return // One warning is enough.
		}
	}
}
