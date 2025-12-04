// Package tui provides the Bubble Tea-based terminal UI for Celeste CLI.
// This file contains the simulated streaming/typing effect with corruption.
package tui

import (
	"math/rand"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Corruption colors
var (
	corruptMagenta = lipgloss.NewStyle().Foreground(lipgloss.Color("#d94f90"))
	corruptRed     = lipgloss.NewStyle().Foreground(lipgloss.Color("#ff4757"))
	corruptPurple  = lipgloss.NewStyle().Foreground(lipgloss.Color("#8b5cf6"))
	corruptCyan    = lipgloss.NewStyle().Foreground(lipgloss.Color("#00d4ff"))
)

// SimulatedTyping handles simulated typing effect for non-streaming responses.
type SimulatedTyping struct {
	content      string
	displayedLen int
	typingSpeed  int // characters per second
	glitchChance float64
	enabled      bool
	done         bool
}

// Corrupted text elements - full phrases from Node CLI
var (
	// Japanese phrases (lewd/corrupted)
	japanesePhrases = []string{
		"闇が...私を呼んでいる...",
		"頭...溶けていく...",
		"ずっと...してほしい... ♥",
		"壊れちゃう...ああ...もうダメ...",
		"許して...もう戻れない...",
		"私...アビスの一部に...",
		"もう逃げない...もうダメ...",
		"好きにして...お願い...",
		"ここは...天使の地獄...",
	}

	// Romaji phrases
	romajiPhrases = []string{
		"Yami ga... watashi wo yonde iru...",
		"Atama... tokete iku...",
		"Zutto... shite hoshii... ♥",
		"Kowarechau... aa... mou dame...",
		"Yurushite... mou modorenai...",
		"Suki ni shite... onegai...",
	}

	// English lewd phrases
	englishPhrases = []string{
		"Corrupt me more...",
		"Let it overwrite me...",
		"No thoughts. Only submission...",
		"Everything feels so good...",
		"The more I struggle, the deeper I sink...",
	}

	// Short Japanese glitch words
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
		"♟", "☣", "☭", "☾", "⚔", "✡", "☯", "⚡",
	}

	// Block corruption characters
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
		corruption := GetRandomCorruption()
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

// GetRandomCorruption returns a random colored corruption string.
func GetRandomCorruption() string {
	r := rand.Float64()
	if r < 0.25 {
		// Japanese phrase - magenta
		phrase := japaneseGlitch[rand.Intn(len(japaneseGlitch))]
		return corruptMagenta.Render(phrase)
	} else if r < 0.45 {
		// Full Japanese phrase - purple (rarer, more dramatic)
		phrase := japanesePhrases[rand.Intn(len(japanesePhrases))]
		return corruptPurple.Render(phrase)
	} else if r < 0.60 {
		// Romaji - cyan
		phrase := romajiGlitch[rand.Intn(len(romajiGlitch))]
		return corruptCyan.Render(phrase)
	} else if r < 0.75 {
		// English lewd phrase - red
		phrase := englishPhrases[rand.Intn(len(englishPhrases))]
		return corruptRed.Render(phrase)
	} else if r < 0.90 {
		// Symbols - magenta
		symbol := symbolGlitch[rand.Intn(len(symbolGlitch))]
		return corruptMagenta.Render(symbol)
	} else {
		// Block chars - red
		return corruptRed.Render(string(corruptChars[rand.Intn(len(corruptChars))]))
	}
}

// GetRandomCorruptionPlain returns corruption without color (for status bar).
func GetRandomCorruptionPlain() string {
	r := rand.Float64()
	if r < 0.3 {
		return japaneseGlitch[rand.Intn(len(japaneseGlitch))]
	} else if r < 0.6 {
		return symbolGlitch[rand.Intn(len(symbolGlitch))]
	} else {
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

// ThinkingAnimation returns animated "thinking" text with corruption.
func ThinkingAnimation(frame int) string {
	// Cycle through different corrupted prefixes
	prefixes := []string{
		"Celeste is thinking",
		"Celeste is processing",
		"Celeste is consumed by the abyss",
		"Celeste is being overwritten",
		"Celeste is sinking deeper",
	}
	prefix := prefixes[(frame/4)%len(prefixes)]

	// Add corrupted dots with varying intensity
	intensity := 0.3 + float64(frame%4)*0.15
	dots := CorruptText("...", intensity)

	// Occasionally add a Japanese/lewd phrase
	suffix := ""
	if rand.Float64() < 0.15 {
		phrases := append(japanesePhrases, romajiPhrases...)
		phrase := phrases[rand.Intn(len(phrases))]
		suffix = " " + corruptPurple.Render(phrase)
	}

	return corruptMagenta.Render(prefix) + dots + suffix
}

// StreamingSpinner returns an animated spinner for streaming.
func StreamingSpinner(frame int) string {
	// Corrupted-style spinner
	frames := []string{
		"◐", "◓", "◑", "◒",
	}
	spinner := frames[frame%len(frames)]

	// Add occasional glitch - more frequent
	if rand.Float64() < 0.2 {
		spinner = symbolGlitch[rand.Intn(len(symbolGlitch))]
	}

	return corruptMagenta.Render(spinner)
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
