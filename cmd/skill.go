package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

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

	projectRoot, err := findProjectRoot()
	if err != nil && !global {
		return fmt.Errorf("finding project root: %w", err)
	}

	// Determine which agents to install for.
	selectedAgents, err := resolveAgents(cmd, agentFilter, projectRoot, global)
	if err != nil {
		return err
	}
	if len(selectedAgents) == 0 {
		output.Messagef(os.Stdout, "No agents selected.")
		return nil
	}

	// Determine which skills to install.
	selectedSkills, err := resolveSkills(cmd, skillFilter)
	if err != nil {
		return err
	}
	if len(selectedSkills) == 0 {
		output.Messagef(os.Stdout, "No skills selected.")
		return nil
	}

	// Install.
	var installed int
	for _, agent := range selectedAgents {
		var baseDir string
		if global {
			baseDir = agent.GlobalPath()
		} else {
			baseDir = agent.ProjectPath(projectRoot)
		}

		for _, s := range selectedSkills {
			destPath := filepath.Join(baseDir, s.Name, "SKILL.md")

			if !force {
				if v := skill.InstalledVersion(destPath); v == version {
					output.Messagef(os.Stdout, "  %s/%s — already at %s (skipped)", agent.DisplayName, s.Name, version)
					continue
				}
			}

			if err := skill.Install(s.Name, baseDir, version); err != nil {
				return fmt.Errorf("installing %s for %s: %w", s.Name, agent.DisplayName, err)
			}
			output.Messagef(os.Stdout, "  %s/%s — installed (%s)", agent.DisplayName, s.Name, version)
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
		var baseDir string
		if global {
			baseDir = agent.GlobalPath()
		} else {
			baseDir = agent.ProjectPath(projectRoot)
		}

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
		var baseDir string
		if global {
			baseDir = agent.GlobalPath()
		} else {
			baseDir = agent.ProjectPath(projectRoot)
		}

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
func resolveAgents(_ *cobra.Command, filter []string, projectRoot string, global bool) ([]skill.Agent, error) {
	// If explicit --agent flag, use those.
	if len(filter) > 0 {
		return resolveAgentList(filter), nil
	}

	// Non-interactive: use all detected agents.
	if !isInteractive() {
		if global {
			return skill.Agents(), nil
		}
		return skill.DetectAgents(projectRoot), nil
	}

	// Interactive: detect agents and show multi-select.
	var candidates []skill.Agent
	if global {
		candidates = skill.Agents()
	} else {
		candidates = skill.DetectAgents(projectRoot)
	}

	if len(candidates) == 0 {
		return nil, fmt.Errorf("no agents detected in %s", projectRoot)
	}

	items := make([]menuItem, len(candidates))
	for i, a := range candidates {
		var dir string
		if global {
			dir = a.GlobalPath()
		} else {
			dir = a.ProjectDir + "/"
		}
		items[i] = menuItem{
			label:       a.DisplayName,
			description: dir,
			selected:    true,
		}
	}

	selected := multiSelect("Select agents to install for:", items)
	var result []skill.Agent
	for _, idx := range selected {
		result = append(result, candidates[idx])
	}
	return result, nil
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

// isInteractive returns true if stdin is a terminal.
func isInteractive() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

// menuItem represents an item in a multi-select menu.
type menuItem struct {
	label       string
	description string
	selected    bool
}

// multiSelect displays a terminal multi-select menu and returns the indices
// of selected items. All items are pre-selected by default.
func multiSelect(prompt string, items []menuItem) []int {
	fmt.Fprintln(os.Stderr, prompt)
	for i, item := range items {
		check := "x"
		if !item.selected {
			check = " "
		}
		fmt.Fprintf(os.Stderr, "  [%s] %d. %s — %s\n", check, i+1, item.label, item.description)
	}
	fmt.Fprintln(os.Stderr)
	fmt.Fprintf(os.Stderr, "Press Enter to confirm, or type numbers to toggle (e.g. '1,3'): ")

	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			// Toggle specified items.
			for _, part := range strings.Split(line, ",") {
				part = strings.TrimSpace(part)
				if idx, err := strconv.Atoi(part); err == nil && idx >= 1 && idx <= len(items) {
					items[idx-1].selected = !items[idx-1].selected
				}
			}
		}
	}

	var selected []int
	for i, item := range items {
		if item.selected {
			selected = append(selected, i)
		}
	}
	return selected
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
