// Package tui provides the Bubble Tea-based terminal UI for Celeste CLI.
// This file contains the skills panel component.
package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// MenuState represents the current menu display mode.
type MenuState int

const (
	MenuStateStatus   MenuState = iota // Default - show status
	MenuStateCommands                  // /menu - show commands
	MenuStateSkills                    // /skills - show skills
)

// SkillsModel represents the skills panel (RPG-style menu).
type SkillsModel struct {
	skills          []SkillDefinition
	executingSkills map[string]string // name -> status
	width           int
	height          int
	// Config info for RPG menu
	endpoint       string
	model          string
	skillsEnabled  bool
	skillsCount    int    // Number of available skills
	disabledReason string // Why skills are disabled (if they are)
	nsfwMode       bool
	// Menu navigation
	menuState    MenuState
	currentInput string // What user is currently typing
}

// NewSkillsModel creates a new skills model.
func NewSkillsModel(skills []SkillDefinition) SkillsModel {
	return SkillsModel{
		skills:          skills,
		executingSkills: make(map[string]string),
	}
}

// SetSize sets the skills panel size.
func (m SkillsModel) SetSize(width, height int) SkillsModel {
	m.width = width
	m.height = height
	return m
}

// SetConfig sets the current configuration info for the RPG menu.
func (m SkillsModel) SetConfig(endpoint, model string, skillsEnabled, nsfwMode bool, skillsCount int, disabledReason string) SkillsModel {
	m.endpoint = endpoint
	m.model = model
	m.skillsEnabled = skillsEnabled
	m.nsfwMode = nsfwMode
	m.skillsCount = skillsCount
	m.disabledReason = disabledReason
	return m
}

// SetCurrentInput updates what the user is currently typing for context-aware help.
func (m SkillsModel) SetCurrentInput(input string) SkillsModel {
	m.currentInput = input
	// If user starts typing, return to status view
	if input != "" && (m.menuState == MenuStateCommands || m.menuState == MenuStateSkills) {
		m.menuState = MenuStateStatus
	}
	return m
}

// SetMenuState sets the menu display mode.
func (m SkillsModel) SetMenuState(state string) SkillsModel {
	switch state {
	case "commands":
		m.menuState = MenuStateCommands
	case "skills":
		m.menuState = MenuStateSkills
	case "status":
		m.menuState = MenuStateStatus
	default:
		// Toggle: if already in that state, return to status
		if m.menuState == MenuStateCommands && state == "commands" {
			m.menuState = MenuStateStatus
		} else if m.menuState == MenuStateSkills && state == "skills" {
			m.menuState = MenuStateStatus
		}
	}
	return m
}

// SetExecuting marks a skill as executing.
func (m SkillsModel) SetExecuting(name string) SkillsModel {
	m.executingSkills[name] = "executing"
	return m
}

// SetCompleted marks a skill as completed.
func (m SkillsModel) SetCompleted(name string) SkillsModel {
	m.executingSkills[name] = "completed"
	return m
}

// SetError marks a skill as errored.
func (m SkillsModel) SetError(name string, err error) SkillsModel {
	m.executingSkills[name] = "error"
	return m
}

// ClearStatus clears all skill statuses.
func (m SkillsModel) ClearStatus() SkillsModel {
	m.executingSkills = make(map[string]string)
	return m
}

// GetDefinitions returns the skill definitions.
func (m SkillsModel) GetDefinitions() []SkillDefinition {
	return m.skills
}

// View renders the skills panel based on current menu state.
func (m SkillsModel) View() string {
	switch m.menuState {
	case MenuStateCommands:
		return m.renderCommandsMenu()
	case MenuStateSkills:
		return m.renderSkillsMenu()
	default:
		return m.renderStatusView()
	}
}

// renderStatusView renders the default status view with contextual help.
func (m SkillsModel) renderStatusView() string {
	var sections []string

	// === SYSTEM STATUS ===
	sections = append(sections, AccentStyle.Render("⚡ SYSTEM STATUS"))

	// Provider line
	providerLine := TextMutedStyle.Render("Provider: ") + PurpleStyle.Bold(true).Render(m.endpoint)
	sections = append(sections, providerLine)

	// Model line
	modelLine := TextMutedStyle.Render("Model: ") + ModelStyle.Render(m.model)
	sections = append(sections, modelLine)

	// Skills status line (with count or reason)
	var skillsLine string
	if m.skillsEnabled {
		skillsLine = TextMutedStyle.Render("Skills: ") +
			SkillCompletedStyle.Render(fmt.Sprintf("✓ Enabled (%d available)", m.skillsCount))
	} else {
		reason := m.disabledReason
		if reason == "" {
			reason = "Model doesn't support tools"
		}
		skillsLine = TextMutedStyle.Render("Skills: ") +
			SkillErrorStyle.Render("✗ Disabled") +
			TextMutedStyle.Render(" - ") + TextSecondaryStyle.Render(reason)
	}

	// NSFW indicator on same line
	if m.nsfwMode {
		skillsLine += TextMutedStyle.Render("        NSFW: ") + NSFWStyle.Render(" 🔥 Enabled ")
	} else {
		skillsLine += TextMutedStyle.Render("        NSFW: ") + TextSecondaryStyle.Render("✗ Disabled")
	}

	sections = append(sections, skillsLine)

	// Show executing skill with corruption animation
	var executingSkill string
	for _, skill := range m.skills {
		if status, ok := m.executingSkills[skill.Name]; ok && status == "executing" {
			corrupted := CorruptText(skill.Name, 0.4)
			executingSkill = SkillExecutingStyle.Render(fmt.Sprintf("⏳ %s ", corrupted)) +
				TextMutedStyle.Render("(Executing...)")
			break
		}
	}

	if executingSkill != "" {
		sections = append(sections, "")
		sections = append(sections, executingSkill)
	}

	// === CONTEXTUAL HELP (when user is typing) ===
	if m.currentInput != "" {
		desc := m.getContextualHelp(m.currentInput)
		if desc != "" {
			sections = append(sections, "")
			sections = append(sections, AccentStyle.Render("💡 HELP"))
			sections = append(sections, TextSecondaryStyle.Render(desc))
		}
	} else {
		// Show tip when not typing
		sections = append(sections, "")
		tip := TextMutedStyle.Render("💡 TIP: Type ") +
			AccentStyle.Render("/menu") +
			TextMutedStyle.Render(" to see commands • ") +
			AccentStyle.Render("/skills") +
			TextMutedStyle.Render(" to see tools")
		sections = append(sections, tip)
	}

	content := lipgloss.JoinVertical(lipgloss.Left, sections...)

	return SkillsPanelStyle.
		Width(m.width).
		Height(m.height).
		Render(content)
}

// renderCommandsMenu renders the commands menu view.
func (m SkillsModel) renderCommandsMenu() string {
	var sections []string

	sections = append(sections, AccentStyle.Render("📋 COMMANDS MENU"))
	sections = append(sections, "")

	// List all commands with descriptions
	commands := []struct {
		cmd  string
		desc string
	}{
		{"/help", "Show detailed help"},
		{"/menu", "Toggle this commands menu"},
		{"/skills", "View available AI skills"},
		{"/config", "List configuration profiles"},
		{"/endpoint", "Switch API provider"},
		{"/model", "Change current model"},
		{"/nsfw", "Enable uncensored mode"},
		{"/safe", "Return to safe mode"},
		{"/clear", "Clear chat history"},
	}

	for _, c := range commands {
		line := AccentStyle.Render(c.cmd) +
			strings.Repeat(" ", 15-len(c.cmd)) +
			TextSecondaryStyle.Render(c.desc)
		sections = append(sections, line)
	}

	sections = append(sections, "")
	tip := TextMutedStyle.Render("💡 Type command name to see details as you type")
	sections = append(sections, tip)

	content := lipgloss.JoinVertical(lipgloss.Left, sections...)

	return SkillsPanelStyle.
		Width(m.width).
		Height(m.height).
		Render(content)
}

// renderSkillsMenu renders the skills menu view.
func (m SkillsModel) renderSkillsMenu() string {
	var sections []string

	title := AccentStyle.Render(fmt.Sprintf("✨ AVAILABLE SKILLS (%d total)", len(m.skills)))
	sections = append(sections, title)
	sections = append(sections, "")

	if !m.skillsEnabled {
		reason := m.disabledReason
		if reason == "" {
			reason = "Model doesn't support function calling"
		}
		sections = append(sections, SkillErrorStyle.Render("❌ Skills are currently disabled"))
		sections = append(sections, TextSecondaryStyle.Render("Reason: "+reason))
		sections = append(sections, "")
		sections = append(sections, TextMutedStyle.Render("Use /safe to enable OpenAI mode with skills support"))
	} else {
		// List all skills with status indicators
		for _, skill := range m.skills {
			status, ok := m.executingSkills[skill.Name]

			var indicator string
			var nameStyle lipgloss.Style

			if !ok {
				indicator = TextMutedStyle.Render("○")
				nameStyle = SkillNameStyle
			} else {
				switch status {
				case "executing":
					corrupted := CorruptText(skill.Name, 0.4)
					indicator = SkillExecutingStyle.Render("⏳")
					nameStyle = SkillExecutingStyle
					skill.Name = corrupted + " (EXECUTING)"
				case "completed":
					indicator = SkillCompletedStyle.Render("✓")
					nameStyle = SkillNameStyle
				case "error":
					indicator = SkillErrorStyle.Render("✗")
					nameStyle = SkillErrorStyle
				default:
					indicator = TextMutedStyle.Render("○")
					nameStyle = SkillNameStyle
				}
			}

			skillLine := indicator + " " + nameStyle.Render(skill.Name)
			sections = append(sections, skillLine)
		}

		sections = append(sections, "")
		tip := TextMutedStyle.Render("💡 Type skill name to see full description")
		sections = append(sections, tip)
	}

	content := lipgloss.JoinVertical(lipgloss.Left, sections...)

	return SkillsPanelStyle.
		Width(m.width).
		Height(m.height).
		Render(content)
}

// getContextualHelp returns help text based on what user is typing.
func (m SkillsModel) getContextualHelp(input string) string {
	input = strings.ToLower(strings.TrimSpace(input))

	// Command descriptions
	commands := map[string]string{
		"/help":     "Show full help menu with all commands and skills",
		"/clear":    "Clear chat history and start fresh conversation",
		"/config":   "List available configuration profiles",
		"/nsfw":     "Switch to Venice.ai uncensored mode (disables skills)",
		"/safe":     "Return to OpenAI safe mode (enables skills)",
		"/endpoint": "Switch API endpoint: /endpoint <openai|grok|venice>",
	}

	// Check if typing a command
	for cmd, desc := range commands {
		if strings.HasPrefix(input, cmd) {
			return cmd + ": " + desc
		}
	}

	// Check if matches a skill name
	for _, skill := range m.skills {
		if strings.Contains(strings.ToLower(skill.Name), input) ||
			strings.Contains(input, strings.ToLower(skill.Name)) {
			return skill.Name + ": " + skill.Description
		}
	}

	return ""
}

// AddSkill adds a skill definition.
func (m SkillsModel) AddSkill(skill SkillDefinition) SkillsModel {
	m.skills = append(m.skills, skill)
	return m
}

// RemoveSkill removes a skill by name.
func (m SkillsModel) RemoveSkill(name string) SkillsModel {
	var filtered []SkillDefinition
	for _, s := range m.skills {
		if s.Name != name {
			filtered = append(filtered, s)
		}
	}
	m.skills = filtered
	delete(m.executingSkills, name)
	return m
}

// GetSkillByName returns a skill by name.
func (m SkillsModel) GetSkillByName(name string) (SkillDefinition, bool) {
	for _, s := range m.skills {
		if s.Name == name {
			return s, true
		}
	}
	return SkillDefinition{}, false
}
