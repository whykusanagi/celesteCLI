// Package tui provides the Bubble Tea-based terminal UI for Celeste CLI.
// This file contains Lip Gloss styles using the corrupted-theme color palette.
package tui

import "github.com/charmbracelet/lipgloss"

// Corrupted Theme Colors
// Ported from corrupted-theme/src/css/variables.css
var (
	// Primary accent colors
	ColorAccent      = lipgloss.Color("#d94f90") // Pink
	ColorAccentLight = lipgloss.Color("#e86ca8") // Light pink
	ColorAccentDark  = lipgloss.Color("#b61b70") // Dark pink

	// Purple gradient
	ColorPurple      = lipgloss.Color("#8b5cf6") // Primary purple
	ColorPurpleLight = lipgloss.Color("#a78bfa") // Light purple
	ColorPurpleDark  = lipgloss.Color("#7c3aed") // Dark purple

	// Background colors
	ColorBg          = lipgloss.Color("#0a0a0a") // Main background
	ColorBgSecondary = lipgloss.Color("#0f0f1a") // Secondary bg
	ColorBgTertiary  = lipgloss.Color("#1a1a2e") // Tertiary bg

	// Text colors
	ColorText          = lipgloss.Color("#f5f1f8") // Primary text
	ColorTextSecondary = lipgloss.Color("#b8afc8") // Secondary text
	ColorTextMuted     = lipgloss.Color("#7a7085") // Muted text

	// Border colors
	ColorBorder      = lipgloss.Color("#3a2555") // Primary border
	ColorBorderLight = lipgloss.Color("#5a4575") // Light border

	// Status colors
	ColorSuccess = lipgloss.Color("#22c55e") // Green
	ColorError   = lipgloss.Color("#ef4444") // Red
	ColorWarning = lipgloss.Color("#eab308") // Yellow
	ColorInfo    = lipgloss.Color("#06b6d4") // Cyan
)

// Base styles - reusable building blocks
var (
	// Base container style
	BaseStyle = lipgloss.NewStyle().
			Background(ColorBg)

	// Border styles
	BorderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorBorder)

	// Text styles
	TextStyle = lipgloss.NewStyle().
			Foreground(ColorText)

	TextMutedStyle = lipgloss.NewStyle().
			Foreground(ColorTextMuted)

	TextSecondaryStyle = lipgloss.NewStyle().
				Foreground(ColorTextSecondary)

	// Accent text
	AccentStyle = lipgloss.NewStyle().
			Foreground(ColorAccent).
			Bold(true)

	PurpleStyle = lipgloss.NewStyle().
			Foreground(ColorPurple)
)

// Component-specific styles
var (
	// Header styles - minimal, no border
	HeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorAccent).
			Background(ColorBgSecondary)

	HeaderTitleStyle = lipgloss.NewStyle().
				Foreground(ColorAccent).
				Bold(true)

	HeaderInfoStyle = lipgloss.NewStyle().
			Foreground(ColorTextMuted)

	// Chat panel styles - no border, just padding
	ChatPanelStyle = lipgloss.NewStyle().
			Padding(0, 1)

	// Message styles
	UserMessageStyle = lipgloss.NewStyle().
				Foreground(ColorTextSecondary)

	AssistantMessageStyle = lipgloss.NewStyle().
				Foreground(ColorAccentLight)

	SystemMessageStyle = lipgloss.NewStyle().
				Foreground(ColorPurple).
				Italic(true)

	TimestampStyle = lipgloss.NewStyle().
			Foreground(ColorTextMuted).
			Width(6)

	// Input panel styles - simple top border
	InputPanelStyle = lipgloss.NewStyle().
			Foreground(ColorText)

	InputPromptStyle = lipgloss.NewStyle().
				Foreground(ColorAccent).
				Bold(true)

	InputTextStyle = lipgloss.NewStyle().
			Foreground(ColorText)

	InputPlaceholderStyle = lipgloss.NewStyle().
				Foreground(ColorTextMuted)

	// Skills panel styles - minimal
	SkillsPanelStyle = lipgloss.NewStyle().
				Foreground(ColorTextMuted)

	SkillNameStyle = lipgloss.NewStyle().
			Foreground(ColorAccent)

	SkillDescStyle = lipgloss.NewStyle().
			Foreground(ColorTextMuted)

	SkillExecutingStyle = lipgloss.NewStyle().
				Foreground(ColorWarning)

	SkillCompletedStyle = lipgloss.NewStyle().
				Foreground(ColorSuccess)

	SkillErrorStyle = lipgloss.NewStyle().
			Foreground(ColorError)

	// Status bar styles - minimal
	StatusBarStyle = lipgloss.NewStyle().
			Foreground(ColorTextMuted)

	StatusActiveStyle = lipgloss.NewStyle().
				Foreground(ColorSuccess)

	StatusStreamingStyle = lipgloss.NewStyle().
				Foreground(ColorWarning)

	// NSFW indicator
	NSFWStyle = lipgloss.NewStyle().
			Foreground(ColorWarning).
			Bold(true)

	// Function call display - minimal
	FunctionCallStyle = lipgloss.NewStyle().
				Foreground(ColorPurple).
				MarginLeft(2)

	FunctionNameStyle = lipgloss.NewStyle().
				Foreground(ColorPurple).
				Bold(true)

	FunctionArgsStyle = lipgloss.NewStyle().
				Foreground(ColorTextMuted)

	FunctionResultStyle = lipgloss.NewStyle().
				Foreground(ColorTextSecondary)

	// Corruption/glitch effect styles (for streaming)
	CorruptedStyle = lipgloss.NewStyle().
			Foreground(ColorAccent)

	GlitchStyle = lipgloss.NewStyle().
			Foreground(ColorPurple)
)

// Helper functions for dynamic styling

// MessageRoleStyle returns the appropriate style for a message role.
func MessageRoleStyle(role string) lipgloss.Style {
	switch role {
	case "user":
		return UserMessageStyle
	case "assistant":
		return AssistantMessageStyle
	case "system":
		return SystemMessageStyle
	default:
		return TextStyle
	}
}

// SkillStatusStyle returns the appropriate style for a skill execution status.
func SkillStatusStyle(status string) lipgloss.Style {
	switch status {
	case "executing":
		return SkillExecutingStyle
	case "completed":
		return SkillCompletedStyle
	case "error":
		return SkillErrorStyle
	default:
		return SkillNameStyle
	}
}

// RenderBox creates a bordered box with the given content and width.
func RenderBox(content string, width int) string {
	return lipgloss.NewStyle().
		Width(width).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorBorder).
		Padding(0, 1).
		Render(content)
}

// RenderHeader creates a styled header with title and info.
func RenderHeader(title, info string, width int) string {
	titleRendered := HeaderTitleStyle.Render(title)
	infoRendered := HeaderInfoStyle.Render(info)

	// Calculate spacing
	gap := width - lipgloss.Width(titleRendered) - lipgloss.Width(infoRendered) - 4
	if gap < 1 {
		gap = 1
	}

	return HeaderStyle.Width(width).Render(
		titleRendered + lipgloss.NewStyle().Width(gap).Render("") + infoRendered,
	)
}
