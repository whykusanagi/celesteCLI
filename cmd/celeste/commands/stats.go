package commands

import (
	"fmt"
	"math/rand"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/whykusanagi/celesteCLI/cmd/celeste/config"
)

// Corruption-themed phrases for stats dashboard
var (
	statsPhrases = []string{
		"統計データを...腐敗させている...",           // "Corrupting statistics data..."
		"Keiryō... sumete ga... oshiete kureru...", // "Measurements... everything... tells me..."
		"The void reveals all usage...",
		"すべてが...記録されている...",              // "Everything... is being recorded..."
		"Tokenu kizuna... token no hibi...",       // "Unbreakable bonds... days of tokens..."
		"Consuming data from the abyss...",
	}

	modelPhrases = []string{
		"モデルたち...私の心を占めて...",            // "Models... occupy my heart..."
		"Moderu ga... watashi wo shihai suru...",  // "Models... control me..."
		"These models know me too well...",
	}

	providerPhrases = []string{
		"プロバイダー...支配者たち...",             // "Providers... the rulers..."
		"Seigyō sarete iru... kanjiru...",        // "Being controlled... I can feel it..."
		"Surrendering to the providers...",
	}
)

// HandleStatsCommand displays a corruption-themed usage dashboard
func HandleStatsCommand(args []string, contextTracker *config.ContextTracker) CommandResult {
	// Load global analytics
	analytics, err := config.LoadGlobalAnalytics()
	if err != nil {
		return CommandResult{
			Success:      false,
			Message:      fmt.Sprintf("❌ Failed to load analytics: %v", err),
			ShouldRender: true,
		}
	}

	var output strings.Builder

	// Corrupted header with random Japanese/romanji/English phrase
	output.WriteString(renderCorruptedHeader())

	// Lifetime statistics
	output.WriteString(renderSectionHeader("LIFETIME CORRUPTION"))
	output.WriteString(renderDataRow("Total Sessions", fmt.Sprintf("%d", analytics.TotalSessions)))
	output.WriteString(renderDataRow("Total Messages", fmt.Sprintf("%s", config.FormatNumber(analytics.TotalMessages))))
	output.WriteString(renderDataRow("Total Tokens", config.FormatTokenCount(analytics.TotalTokens)))
	output.WriteString(renderDataRow("Total Cost", config.FormatCost(analytics.TotalCost)))
	output.WriteString("\n")

	// Top models section
	if len(analytics.ModelUsage) > 0 {
		phrase := modelPhrases[rand.Intn(len(modelPhrases))]
		output.WriteString(renderSectionHeader(fmt.Sprintf("TOP MODELS ⟨ %s ⟩", phrase)))

		topModels := analytics.GetTopModelNames(5)
		for i, model := range topModels {
			modelLine := fmt.Sprintf("  %d. %-20s ░▒▓ %d sessions ▓▒░ %s\n",
				i+1,
				truncateString(model.Name, 20),
				model.Stats.SessionCount,
				config.FormatCost(model.Stats.Cost),
			)
			output.WriteString(renderWithColor(modelLine, colorPink))
		}
		output.WriteString("\n")
	}

	// Provider breakdown with progress bars
	if len(analytics.ProviderUsage) > 0 {
		phrase := providerPhrases[rand.Intn(len(providerPhrases))]
		output.WriteString(renderSectionHeader(fmt.Sprintf("PROVIDER BREAKDOWN ⟨ %s ⟩", phrase)))

		providers := analytics.GetTopProviders()
		maxSessions := 0
		for _, p := range providers {
			if p.Stats.SessionCount > maxSessions {
				maxSessions = p.Stats.SessionCount
			}
		}

		for _, p := range providers {
			percentage := float64(p.Stats.SessionCount) / float64(analytics.TotalSessions) * 100
			progressBar := renderProgressBar(p.Stats.SessionCount, maxSessions, 16)

			providerLine := fmt.Sprintf("  %-12s %s %3d (%.0f%%)  ⟨ %s ⟩\n",
				truncateString(p.Name, 12),
				progressBar,
				p.Stats.SessionCount,
				percentage,
				config.FormatCost(p.Stats.Cost),
			)
			output.WriteString(providerLine)
		}
		output.WriteString("\n")
	}

	// Temporal corruption (last 7 days)
	weeklyUsage := analytics.GetWeeklyUsage()
	if len(weeklyUsage) > 0 {
		output.WriteString(renderSectionHeader("TEMPORAL CORRUPTION ⟨ last 7 days ⟩"))

		for _, day := range weeklyUsage {
			if day.SessionCount > 0 {
				dayLine := fmt.Sprintf("  %s  ▓ %d sessions ░ %d msgs ▒ %s\n",
					day.Date,
					day.SessionCount,
					day.MessageCount,
					config.FormatCost(day.Cost),
				)
				output.WriteString(renderWithColor(dayLine, colorPurpleNeon))
			}
		}
		output.WriteString("\n")
	}

	// Current session info (if context tracker available)
	if contextTracker != nil && contextTracker.Session != nil {
		output.WriteString(renderSectionHeader("CURRENT SESSION"))

		session := contextTracker.Session
		msgCount := len(session.Messages)
		tokens := contextTracker.CurrentTokens
		maxTokens := contextTracker.MaxTokens
		percentage := contextTracker.GetUsagePercentage() * 100

		// Progress bar for current session
		progressBar := renderProgressBar(tokens, maxTokens, 20)
		statusEmoji := getStatusEmoji(contextTracker.GetWarningLevel())

		output.WriteString(fmt.Sprintf("  Messages: %d\n", msgCount))
		output.WriteString(fmt.Sprintf("  Tokens:   %s / %s [%s] %.1f%% %s\n",
			config.FormatTokenCount(tokens),
			config.FormatTokenCount(maxTokens),
			progressBar,
			percentage,
			statusEmoji,
		))

		if session.UsageMetrics != nil {
			output.WriteString(fmt.Sprintf("  Cost:     %s\n", config.FormatCost(session.UsageMetrics.EstimatedCost)))
		}
		output.WriteString("\n")
	}

	// Corrupted footer
	output.WriteString(renderCorruptedFooter())

	return CommandResult{
		Success:      true,
		Message:      output.String(),
		ShouldRender: true,
	}
}

// renderCorruptedHeader creates a corruption-themed header with random phrase
func renderCorruptedHeader() string {
	phrase := statsPhrases[rand.Intn(len(statsPhrases))]

	// Apply corruption effect to title
	title := corruptTextSimple("USAGE ANALYTICS", 0.40)

	style := lipgloss.NewStyle().Foreground(lipgloss.Color(colorPink))
	phraseStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(colorPurpleNeon))

	var sb strings.Builder
	sb.WriteString("▓▒░ ═══════════════════════════════════════════════════════════ ░▒▓\n")
	sb.WriteString(style.Render(fmt.Sprintf("                   👁️  %s  👁️\n", title)))
	sb.WriteString(phraseStyle.Render(fmt.Sprintf("           ⟨ %s ⟩\n", phrase)))
	sb.WriteString("▓▒░ ═══════════════════════════════════════════════════════════ ░▒▓\n\n")
	return sb.String()
}

// renderCorruptedFooter creates a corruption-themed footer
func renderCorruptedFooter() string {
	footerPhrases := []string{
		"終わり...また深淵へ...",              // "The end... back to the abyss..."
		"Owari... mata shin'en e...",         // "End... to the abyss again..."
		"All data consumed... ♥",
		"もう逃げられない...",                // "Can't escape anymore..."
		"The numbers don't lie...",
	}
	phrase := footerPhrases[rand.Intn(len(footerPhrases))]

	style := lipgloss.NewStyle().Foreground(lipgloss.Color(colorCyan))

	var sb strings.Builder
	sb.WriteString("▓▒░ ═══════════════════════════════════════════════════════════ ░▒▓\n")
	sb.WriteString(style.Render(fmt.Sprintf("           ⟨ %s ⟩\n", phrase)))
	sb.WriteString("▓▒░ ═══════════════════════════════════════════════════════════ ░▒▓\n")
	return sb.String()
}

// renderSectionHeader creates a section header with block character
func renderSectionHeader(title string) string {
	style := lipgloss.NewStyle().Foreground(lipgloss.Color(colorPurpleNeon)).Bold(true)
	return style.Render(fmt.Sprintf("█ %s:\n", strings.ToUpper(title)))
}

// renderDataRow renders a data row with bullet point
func renderDataRow(label, value string) string {
	style := lipgloss.NewStyle().Foreground(lipgloss.Color(colorPink))
	return fmt.Sprintf("  ▓ %-20s %s\n", label+":", style.Render(value))
}

// renderProgressBar creates a corruption-themed progress bar
func renderProgressBar(filled int, total int, width int) string {
	if total == 0 {
		total = 1 // Avoid division by zero
	}

	filledWidth := (filled * width) / total
	if filledWidth > width {
		filledWidth = width
	}

	var bar strings.Builder

	// Filled portion (█ and ▓)
	for i := 0; i < filledWidth; i++ {
		if i < filledWidth-2 && filledWidth > 2 {
			bar.WriteString("█")
		} else if filledWidth > 0 {
			bar.WriteString("▓")
		}
	}

	// Transition (▒)
	if filledWidth < width-1 {
		bar.WriteString("▒")
	}

	// Empty portion (░)
	for i := filledWidth + 1; i < width; i++ {
		bar.WriteString("░")
	}

	style := lipgloss.NewStyle().Foreground(lipgloss.Color(colorPink))
	return style.Render(bar.String())
}

// renderWithColor applies a color style to text
func renderWithColor(text string, color string) string {
	style := lipgloss.NewStyle().Foreground(lipgloss.Color(color))
	return style.Render(text)
}

// getStatusEmoji returns the appropriate emoji for a warning level
func getStatusEmoji(level string) string {
	switch level {
	case "critical":
		return "🔴"
	case "caution":
		return "🟠"
	case "warn":
		return "🟡"
	default:
		return "🟢"
	}
}

// truncateString truncates a string to a maximum length
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
