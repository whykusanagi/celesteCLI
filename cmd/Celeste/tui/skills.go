// Package tui provides the Bubble Tea-based terminal UI for Celeste CLI.
// This file contains the skills panel component.
package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// SkillsModel represents the skills panel (RPG-style menu).
type SkillsModel struct {
	skills          []SkillDefinition
	executingSkills map[string]string // name -> status
	width           int
	height          int
	// Config info for RPG menu
	endpoint      string
	model         string
	skillsEnabled bool
	nsfwMode      bool
	// Context-aware help
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
func (m SkillsModel) SetConfig(endpoint, model string, skillsEnabled, nsfwMode bool) SkillsModel {
	m.endpoint = endpoint
	m.model = model
	m.skillsEnabled = skillsEnabled
	m.nsfwMode = nsfwMode
	return m
}

// SetCurrentInput updates what the user is currently typing for context-aware help.
func (m SkillsModel) SetCurrentInput(input string) SkillsModel {
	m.currentInput = input
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

// View renders the skills panel - RPG-style menu with contextual help.
func (m SkillsModel) View() string {
	var sections []string

	// === SYSTEM STATUS (Compact) ===
	statusLine := TextMutedStyle.Render(m.endpoint) + TextMutedStyle.Render(" • ") +
		ModelStyle.Render(m.model)

	if m.nsfwMode {
		statusLine += TextMutedStyle.Render(" • ") + NSFWStyle.Render("🔥")
	}

	if m.skillsEnabled {
		statusLine += TextMutedStyle.Render(" • Skills ") + SkillCompletedStyle.Render("✓")
	}

	sections = append(sections, AccentStyle.Render("⚡ STATUS"), statusLine)

	// === COMMANDS (Compact list) ===
	sections = append(sections, "", AccentStyle.Render("📋 COMMANDS"))

	commandsList := TextMutedStyle.Render("/help /clear /config /nsfw /safe /endpoint")
	sections = append(sections, commandsList)

	// === SKILLS (Compact, show executing ones with status) ===
	if m.skillsEnabled && len(m.skills) > 0 {
		sections = append(sections, "", AccentStyle.Render("✨ SKILLS"))

		// Show executing skills first
		var executing []string
		for _, skill := range m.skills {
			if status, ok := m.executingSkills[skill.Name]; ok && status == "executing" {
				executing = append(executing, SkillExecutingStyle.Render("⏳"+skill.Name))
			}
		}

		if len(executing) > 0 {
			sections = append(sections, strings.Join(executing, " "))
		}

		// Show all skills compactly
		var skillNames []string
		for _, skill := range m.skills {
			if _, ok := m.executingSkills[skill.Name]; !ok {
				skillNames = append(skillNames, TextMutedStyle.Render(skill.Name))
			}
		}

		if len(skillNames) > 0 {
			// Split into multiple lines if too long
			line := ""
			for i, name := range skillNames {
				if i > 0 {
					line += TextMutedStyle.Render(" • ")
				}
				line += name

				// Break into multiple lines if needed
				if len(line) > 70 || i == len(skillNames)-1 {
					sections = append(sections, line)
					line = ""
				}
			}
		}
	}

	// === CONTEXTUAL HELP (Shows description when user types) ===
	if m.currentInput != "" {
		desc := m.getContextualHelp(m.currentInput)
		if desc != "" {
			sections = append(sections, "", AccentStyle.Render("💡 HELP"))
			sections = append(sections, TextSecondaryStyle.Render(desc))
		}
	}

	// Join all sections
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

// renderSkillCard renders a single skill card.
func (m SkillsModel) renderSkillCard(skill SkillDefinition, width int) string {
	// Status indicator
	status, ok := m.executingSkills[skill.Name]
	var renderedName string

	if !ok {
		renderedName = SkillNameStyle.Render(skill.Name)
	} else {
		switch status {
		case "executing":
			// Show corruption effect while executing
			renderedName = RenderCorruptedSkill(skill.Name)
		case "completed":
			// Return to normal after completion
			renderedName = SkillNameStyle.Render(skill.Name)
		case "error":
			renderedName = SkillErrorStyle.Render("✗" + skill.Name)
		default:
			renderedName = SkillNameStyle.Render(skill.Name)
		}
	}

	// Truncate description
	desc := skill.Description
	if len(desc) > width-2 {
		desc = desc[:width-5] + "..."
	}

	description := SkillDescStyle.Render(desc)

	card := lipgloss.JoinVertical(lipgloss.Left, renderedName, description)

	return lipgloss.NewStyle().
		Width(width).
		Margin(0, 1, 0, 0).
		Render(card)
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
