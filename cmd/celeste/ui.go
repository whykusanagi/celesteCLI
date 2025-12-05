package main

import (
	"fmt"
	"os"
	"strings"
	"time"
)

// MessageType represents different types of messages for consistent styling
type MessageType string

const (
	INFO    MessageType = "INFO"
	SUCCESS MessageType = "SUCCESS"
	WARN    MessageType = "WARN"
	ERROR   MessageType = "ERROR"
	DEBUG   MessageType = "DEBUG"
)

// OperationMode represents different CLI operation modes with distinct visual styles
type OperationMode string

const (
	NORMAL    OperationMode = "NORMAL"
	TAROT     OperationMode = "TAROT"
	NSFW      OperationMode = "NSFW"
	TWITTER   OperationMode = "TWITTER"
	STREAMING OperationMode = "STREAMING"
)

// SeparatorStyle defines different visual separator styles
type SeparatorStyle string

const (
	HEAVY     SeparatorStyle = "HEAVY"     // ═══ (double line)
	LIGHT     SeparatorStyle = "LIGHT"     // ─── (single line)
	DASHED    SeparatorStyle = "DASHED"    // ╌╌╌ (dashed)
	CORRUPTED SeparatorStyle = "CORRUPTED" // ≈≈≈ (wavy/corrupted)
)

// ANSI Color codes
const (
	// Text colors
	ColorDefault = "\033[0m"
	ColorBlack   = "\033[30m"
	ColorRed     = "\033[31m"
	ColorGreen   = "\033[32m"
	ColorYellow  = "\033[33m"
	ColorBlue    = "\033[34m"
	ColorMagenta = "\033[35m"
	ColorCyan    = "\033[36m"
	ColorWhite   = "\033[37m"

	// Bright colors
	ColorBrightRed     = "\033[91m"
	ColorBrightGreen   = "\033[92m"
	ColorBrightYellow  = "\033[93m"
	ColorBrightBlue    = "\033[94m"
	ColorBrightMagenta = "\033[95m"
	ColorBrightCyan    = "\033[96m"

	// Text styles
	Bold      = "\033[1m"
	Dim       = "\033[2m"
	Italic    = "\033[3m"
	Underline = "\033[4m"
	Blink     = "\033[5m"
)

// getColorForMode returns the appropriate color for the current operation mode
func getColorForMode(mode OperationMode) string {
	switch mode {
	case TAROT:
		return ColorBrightMagenta
	case NSFW:
		return ColorBrightYellow
	case TWITTER:
		return ColorBrightBlue
	case STREAMING:
		return ColorBrightGreen
	case NORMAL:
		fallthrough
	default:
		return ColorCyan
	}
}

// getColorForMessageType returns the appropriate color for a message type
func getColorForMessageType(t MessageType) string {
	switch t {
	case SUCCESS:
		return ColorGreen
	case WARN:
		return ColorYellow
	case ERROR:
		return ColorBrightRed
	case DEBUG:
		return ColorCyan
	case INFO:
		fallthrough
	default:
		return ColorCyan
	}
}

// getEmojiForMessageType returns the emoji for a message type
func getEmojiForMessageType(t MessageType) string {
	switch t {
	case SUCCESS:
		return "✅"
	case WARN:
		return "⚠️"
	case ERROR:
		return "❌"
	case DEBUG:
		return "🔍"
	case INFO:
		fallthrough
	default:
		return "📋"
	}
}

// PrintMessage prints a standardized message with type-specific formatting
func PrintMessage(msgType MessageType, msg string) {
	emoji := getEmojiForMessageType(msgType)
	color := getColorForMessageType(msgType)

	fmt.Fprintf(os.Stderr, "%s%s %s%s\n", color, emoji, msg, ColorDefault)
}

// PrintMessagef prints a formatted message similar to Printf but with type styling
func PrintMessagef(msgType MessageType, format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	PrintMessage(msgType, msg)
}

// PrintPhase prints a phase indicator showing progress through operation steps
// Example: [✓] Config loaded  [✓] Loaded  [●] Processing...  [ ] Formatting
func PrintPhase(current, total int, phase string) {
	phases := ""

	// Build phase indicators
	for i := 1; i <= total; i++ {
		if i < current {
			phases += "✓ "
		} else if i == current {
			phases += fmt.Sprintf("%s%s●%s ", ColorBrightCyan, Bold, ColorDefault)
		} else {
			phases += "○ "
		}
	}

	fmt.Fprintf(os.Stderr, "%s[%s] %s%s\n", ColorCyan, phases, phase, ColorDefault)
}

// PrintPhaseSimple prints a single phase as [✓], [●], or [ ]
func PrintPhaseSimple(status string, phaseText string) {
	var indicator string
	var color string

	switch status {
	case "done":
		indicator = "✓"
		color = ColorGreen
	case "active":
		indicator = "●"
		color = ColorBrightCyan
	default:
		indicator = " "
		color = ColorDefault
	}

	fmt.Fprintf(os.Stderr, "%s[%s%s%s] %s\n", ColorCyan, color, indicator, ColorDefault, phaseText)
}

// PrintSuccess prints a success message with operation details
// Example: ✅ Complete in 2.34s | 145 tokens | Streamed to 280 chars
func PrintSuccess(operation string, duration time.Duration, metadata map[string]string) {
	emoji := "✅"
	parts := []string{fmt.Sprintf("%s in %.2fs", operation, duration.Seconds())}

	// Add metadata if provided
	for key, value := range metadata {
		parts = append(parts, fmt.Sprintf("%s: %s", key, value))
	}

	message := strings.Join(parts, " | ")
	fmt.Fprintf(os.Stderr, "%s%s %s%s\n", ColorGreen, emoji, message, ColorDefault)
}

// PrintError prints an error message with optional hint/resolution
func PrintError(operation string, err error, hint string) {
	fmt.Fprintf(os.Stderr, "%s❌ %s: %v%s\n", ColorBrightRed, operation, err, ColorDefault)

	if hint != "" {
		fmt.Fprintf(os.Stderr, "%s💡 Hint: %s%s\n", ColorYellow, hint, ColorDefault)
	}
}

// PrintErrorBox prints an error in a formatted box with resolution instructions
func PrintErrorBox(title string, error string, hints []string, docLink string) {
	const width = 50

	// Top border
	fmt.Fprintf(os.Stderr, "%s╔", ColorBrightRed)
	fmt.Fprintf(os.Stderr, "%s", strings.Repeat("═", width-2))
	fmt.Fprintf(os.Stderr, "╗%s\n", ColorDefault)

	// Title
	fmt.Fprintf(os.Stderr, "%s║ ❌ %s%s\n", ColorBrightRed, padRight(title, width-5), ColorDefault)

	// Separator
	fmt.Fprintf(os.Stderr, "%s╟", ColorBrightRed)
	fmt.Fprintf(os.Stderr, "%s", strings.Repeat("─", width-2))
	fmt.Fprintf(os.Stderr, "╢%s\n", ColorDefault)

	// Error message
	fmt.Fprintf(os.Stderr, "%s║ %s%s\n", ColorDefault, padRight(error, width-2), ColorDefault)

	// Hints
	if len(hints) > 0 {
		fmt.Fprintf(os.Stderr, "%s║ %s\n", ColorDefault, padRight("HOW TO FIX:", width-2))
		for _, hint := range hints {
			// Wrap hint if too long
			wrappedHint := wrapText(hint, width-4)
			for _, line := range wrappedHint {
				fmt.Fprintf(os.Stderr, "%s║ %s%s\n", ColorYellow, padRight(line, width-2), ColorDefault)
			}
		}
	}

	// Documentation link
	if docLink != "" {
		fmt.Fprintf(os.Stderr, "%s║ %s\n", ColorDefault, padRight("", width-2))
		linkText := fmt.Sprintf("📖 Docs: %s", docLink)
		fmt.Fprintf(os.Stderr, "%s║ %s%s\n", ColorCyan, padRight(linkText, width-2), ColorDefault)
	}

	// Bottom border
	fmt.Fprintf(os.Stderr, "%s╚", ColorBrightRed)
	fmt.Fprintf(os.Stderr, "%s", strings.Repeat("═", width-2))
	fmt.Fprintf(os.Stderr, "╝%s\n", ColorDefault)
}

// PrintConfig displays the active configuration in a formatted header
func PrintConfig(config map[string]string) {
	const width = 50

	// Top border
	fmt.Fprintf(os.Stderr, "%s┏", ColorBrightCyan)
	fmt.Fprintf(os.Stderr, "%s━", strings.Repeat("━", width-2))
	fmt.Fprintf(os.Stderr, "┓%s\n", ColorDefault)

	// Title
	fmt.Fprintf(os.Stderr, "%s┃ %sActive Configuration%s %s\n", ColorBrightCyan, Bold, ColorDefault, padRight("", width-22))

	// Config items
	for key, value := range config {
		line := fmt.Sprintf("  %s: %s", key, value)
		fmt.Fprintf(os.Stderr, "%s┃ %s%s\n", ColorBrightCyan, padRight(line, width-2), ColorDefault)
	}

	// Bottom border
	fmt.Fprintf(os.Stderr, "%s┗", ColorBrightCyan)
	fmt.Fprintf(os.Stderr, "%s━", strings.Repeat("━", width-2))
	fmt.Fprintf(os.Stderr, "┛%s\n", ColorDefault)
	fmt.Fprintf(os.Stderr, "\n")
}

// PrintHeader displays an operation header with mode and details
func PrintHeader(mode OperationMode, details map[string]string) {
	modeColor := getColorForMode(mode)

	// Mode badge
	fmt.Fprintf(os.Stderr, "%s[%s%s%s]", ColorDefault, modeColor, mode, ColorDefault)

	// Details
	for key, value := range details {
		fmt.Fprintf(os.Stderr, " %s[%s: %s]%s", ColorCyan, key, value, ColorDefault)
	}

	fmt.Fprintf(os.Stderr, "\n")
}

// PrintSeparator prints a visual separator with optional text
func PrintSeparator(style SeparatorStyle) {
	const width = 60
	var char string

	switch style {
	case HEAVY:
		char = "═"
	case LIGHT:
		char = "─"
	case DASHED:
		char = "╌"
	case CORRUPTED:
		char = "≈"
	default:
		char = "─"
	}

	fmt.Fprintf(os.Stderr, "%s%s%s\n", ColorCyan, strings.Repeat(char, width), ColorDefault)
}

// PrintSeparatorWithText prints a separator with centered text
func PrintSeparatorWithText(style SeparatorStyle, text string) {
	const width = 60
	var char string

	switch style {
	case HEAVY:
		char = "═"
	case LIGHT:
		char = "─"
	case DASHED:
		char = "╌"
	case CORRUPTED:
		char = "≈"
	default:
		char = "─"
	}

	// Calculate spacing
	textWithSpaces := fmt.Sprintf(" %s ", text)
	remainingWidth := width - len(textWithSpaces)
	leftPad := remainingWidth / 2
	rightPad := remainingWidth - leftPad

	line := strings.Repeat(char, leftPad) + textWithSpaces + strings.Repeat(char, rightPad)

	fmt.Fprintf(os.Stderr, "%s%s%s\n", ColorBrightCyan, line, ColorDefault)
}

// PrintResponseReady prints a visual indicator that response is ready
func PrintResponseReady() {
	fmt.Fprintf(os.Stderr, "\n")
	PrintSeparatorWithText(HEAVY, "✨ Response Ready ✨")
	fmt.Fprintf(os.Stderr, "\n")
}

// PrintModeIndicator prints a color-coded mode indicator
func PrintModeIndicator(mode OperationMode) {
	modeColor := getColorForMode(mode)
	symbol := "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

	fmt.Fprintf(os.Stderr, "%s[%s%s%s] %s\n", ColorDefault, modeColor, mode, ColorDefault, symbol)
}

// PrintConfigStatus displays configuration validation status
func PrintConfigStatus(checks map[string]bool) {
	fmt.Fprintf(os.Stderr, "%sConfiguration Status:%s\n", ColorCyan, ColorDefault)

	for check, passed := range checks {
		if passed {
			fmt.Fprintf(os.Stderr, "%s  ✓ %s%s\n", ColorGreen, check, ColorDefault)
		} else {
			fmt.Fprintf(os.Stderr, "%s  ✗ %s%s\n", ColorBrightRed, check, ColorDefault)
		}
	}
}

// Helper functions

// padRight pads a string to a specific width on the right
func padRight(s string, width int) string {
	if len(s) >= width {
		return s[:width]
	}
	return s + strings.Repeat(" ", width-len(s))
}

// wrapText wraps text to a specific width, returning lines
func wrapText(s string, width int) []string {
	var lines []string
	words := strings.Fields(s)
	var currentLine string

	for _, word := range words {
		if len(currentLine)+len(word)+1 > width {
			if currentLine != "" {
				lines = append(lines, currentLine)
				currentLine = word
			} else {
				lines = append(lines, word)
			}
		} else {
			if currentLine == "" {
				currentLine = word
			} else {
				currentLine += " " + word
			}
		}
	}

	if currentLine != "" {
		lines = append(lines, currentLine)
	}

	return lines
}

// ClearLine clears the current line (used for animation cleanup)
func ClearLine() {
	fmt.Fprintf(os.Stderr, "\r\033[K")
}
