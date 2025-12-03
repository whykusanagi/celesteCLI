// Package tui provides the Bubble Tea-based terminal UI for Celeste CLI.
// This file contains the simulated streaming/typing effect with corruption.
package tui

import (
	"math/rand"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// SimulatedTyping handles simulated typing effect for non-streaming responses.
type SimulatedTyping struct {
	content       string
	displayedLen  int
	typingSpeed   int  // characters per second
	glitchChance  float64
	enabled       bool
	done          bool
}

// Corrupted text elements - ported from old Go CLI animation.go
var (
	// Japanese phrases for corruption effect
	japaneseGlitch = []string{
		"ニャー", "かわいい", "変態", "えっち", "デレデレ",
		"きゃー", "あはは", "うふふ", "やだ", "ばか",
	}

	// Romaji/text corruption
	romajiGlitch = []string{
		"nyaa~", "ara ara~", "fufufu~", "kyaa~", "baka~",
		"<3", "uwu", "owo", ">w<", "^w^",
	}

	// Symbol corruption
	symbolGlitch = []string{
		"★", "☆", "♥", "♡", "✧", "✦", "◆", "◇", "●", "○",
		"█", "▓", "▒", "░", "▄", "▀", "▌", "▐",
	}

	// ANSI corruption - special characters
	corruptChars = []rune{
		'█', '▓', '▒', '░', '▄', '▀', '▌', '▐',
		'╔', '╗', '╚', '╝', '═', '║', '╠', '╣',
		'▲', '▼', '◄', '►', '◊', '○', '●', '◘',
	}
)

// NewSimulatedTyping creates a new simulated typing effect.
func NewSimulatedTyping(content string, typingSpeed int, glitchChance float64) *SimulatedTyping {
	if typingSpeed <= 0 {
		typingSpeed = 40 // Default 40 chars/sec
	}
	if glitchChance < 0 {
		glitchChance = 0.02 // 2% chance
	}
	if glitchChance > 1 {
		glitchChance = 1
	}

	return &SimulatedTyping{
		content:      content,
		displayedLen: 0,
		typingSpeed:  typingSpeed,
		glitchChance: glitchChance,
		enabled:      true,
	}
}

// IsEnabled returns whether simulated typing is enabled.
func (s *SimulatedTyping) IsEnabled() bool {
	return s.enabled
}

// SetEnabled enables or disables simulated typing.
func (s *SimulatedTyping) SetEnabled(enabled bool) {
	s.enabled = enabled
}

// IsDone returns whether typing simulation is complete.
func (s *SimulatedTyping) IsDone() bool {
	return s.done || s.displayedLen >= len(s.content)
}

// GetDisplayed returns the currently displayed content.
func (s *SimulatedTyping) GetDisplayed() string {
	if !s.enabled || s.displayedLen >= len(s.content) {
		return s.content
	}
	return s.content[:s.displayedLen]
}

// GetDisplayedWithCorruption returns displayed content with potential corruption.
func (s *SimulatedTyping) GetDisplayedWithCorruption() string {
	displayed := s.GetDisplayed()
	if !s.enabled || s.IsDone() {
		return displayed
	}

	// Add corruption effect to the "cursor" position
	if rand.Float64() < s.glitchChance {
		corruption := getRandomCorruption()
		return displayed + corruption
	}

	return displayed
}

// Advance advances the typing simulation by one tick.
// Returns the number of characters advanced.
func (s *SimulatedTyping) Advance() int {
	if s.IsDone() {
		return 0
	}

	// Calculate characters to advance based on speed
	// typingSpeed is chars/sec, we tick ~30 times/sec
	charsPerTick := max(1, s.typingSpeed/30)

	oldLen := s.displayedLen
	s.displayedLen = min(s.displayedLen+charsPerTick, len(s.content))

	if s.displayedLen >= len(s.content) {
		s.done = true
	}

	return s.displayedLen - oldLen
}

// Reset resets the typing simulation.
func (s *SimulatedTyping) Reset(content string) {
	s.content = content
	s.displayedLen = 0
	s.done = false
}

// TickCmd returns a command that sends a typing tick.
func TypingTickCmd() tea.Cmd {
	return tea.Tick(time.Millisecond*33, func(t time.Time) tea.Msg { // ~30fps
		return TypingTickMsg{}
	})
}

// TypingTickMsg is sent for typing animation ticks.
type TypingTickMsg struct{}

// getRandomCorruption returns a random corruption string.
func getRandomCorruption() string {
	r := rand.Float64()
	if r < 0.3 {
		return japaneseGlitch[rand.Intn(len(japaneseGlitch))]
	} else if r < 0.6 {
		return romajiGlitch[rand.Intn(len(romajiGlitch))]
	} else if r < 0.8 {
		return symbolGlitch[rand.Intn(len(symbolGlitch))]
	} else {
		// Random corrupt char
		return string(corruptChars[rand.Intn(len(corruptChars))])
	}
}

// CorruptText adds corruption effects to a string.
// Used for loading states and other animated text.
func CorruptText(text string, intensity float64) string {
	if intensity <= 0 {
		return text
	}

	runes := []rune(text)
	result := make([]rune, len(runes))

	for i, r := range runes {
		if rand.Float64() < intensity {
			result[i] = corruptChars[rand.Intn(len(corruptChars))]
		} else {
			result[i] = r
		}
	}

	return string(result)
}

// ThinkingAnimation returns animated "thinking" text.
func ThinkingAnimation(frame int) string {
	frames := []string{
		"Thinking" + CorruptText("...", 0.3),
		"Thinking" + CorruptText("...", 0.5),
		"Thinking" + CorruptText("...", 0.7),
		"Thinking" + CorruptText("...", 0.5),
	}
	return frames[frame%len(frames)]
}

// StreamingSpinner returns an animated spinner for streaming.
func StreamingSpinner(frame int) string {
	// Corrupted-style spinner
	frames := []string{
		"◐", "◓", "◑", "◒",
	}
	spinner := frames[frame%len(frames)]

	// Add occasional glitch
	if rand.Float64() < 0.1 {
		spinner = symbolGlitch[rand.Intn(len(symbolGlitch))]
	}

	return spinner
}

// DetectDump checks if response appears to be a "dump" (all at once)
// rather than true streaming. Used to decide if simulation is needed.
func DetectDump(firstChunkSize, totalSize int, elapsed time.Duration) bool {
	// If >80% of content arrives in first chunk within 500ms, it's a dump
	if totalSize <= 0 {
		return false
	}

	ratio := float64(firstChunkSize) / float64(totalSize)
	return ratio > 0.8 && elapsed < 500*time.Millisecond
}

// Helper functions
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

