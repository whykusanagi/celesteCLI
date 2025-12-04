// Package tui provides the Bubble Tea-based terminal UI for Celeste CLI.
// This file contains the skills panel component.
package tui

import (
	"github.com/charmbracelet/lipgloss"
)

// SkillsModel represents the skills panel.
type SkillsModel struct {
	skills          []SkillDefinition
	executingSkills map[string]string // name -> status
	width           int
	height          int
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

// View renders the skills panel - compact single-line format.
func (m SkillsModel) View() string {
	if len(m.skills) == 0 {
		return SkillsPanelStyle.
			Width(m.width).
			Height(m.height).
			Render(TextMutedStyle.Render("No skills"))
	}

	// Compact: just show skill names in a line
	var skillNames []string
	for _, skill := range m.skills {
		status, ok := m.executingSkills[skill.Name]
		var name string
		if !ok {
			name = SkillNameStyle.Render(skill.Name)
		} else {
			switch status {
			case "executing":
				name = SkillExecutingStyle.Render("⏳" + skill.Name)
			case "completed":
				name = SkillCompletedStyle.Render("✓" + skill.Name)
			case "error":
				name = SkillErrorStyle.Render("✗" + skill.Name)
			default:
				name = SkillNameStyle.Render(skill.Name)
			}
		}
		skillNames = append(skillNames, name)
	}

	// Title + skills on same line
	title := AccentStyle.Render("Skills:") + " "

	// Join skill names with separator
	var joined string
	for i, name := range skillNames {
		if i > 0 {
			joined += TextMutedStyle.Render(" • ")
		}
		joined += name
	}

	content := title + TextMutedStyle.Render("[") + joined + TextMutedStyle.Render("]")

	return SkillsPanelStyle.
		Width(m.width).
		Height(m.height).
		Render(content)
}

// renderSkillCard renders a single skill card.
func (m SkillsModel) renderSkillCard(skill SkillDefinition, width int) string {
	// Status indicator
	status, ok := m.executingSkills[skill.Name]
	var indicator string
	var nameStyle lipgloss.Style

	if !ok {
		indicator = TextMutedStyle.Render("●")
		nameStyle = SkillNameStyle
	} else {
		switch status {
		case "executing":
			indicator = SkillExecutingStyle.Render("⏳")
			nameStyle = SkillExecutingStyle
		case "completed":
			indicator = SkillCompletedStyle.Render("✓")
			nameStyle = SkillCompletedStyle
		case "error":
			indicator = SkillErrorStyle.Render("✗")
			nameStyle = SkillErrorStyle
		default:
			indicator = TextMutedStyle.Render("●")
			nameStyle = SkillNameStyle
		}
	}

	// Truncate name if needed
	name := skill.Name
	if len(name) > width-4 {
		name = name[:width-7] + "..."
	}

	// Truncate description
	desc := skill.Description
	if len(desc) > width-2 {
		desc = desc[:width-5] + "..."
	}

	header := indicator + " " + nameStyle.Render(name)
	description := SkillDescStyle.Render(desc)

	card := lipgloss.JoinVertical(lipgloss.Left, header, description)

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
