package commands

import (
	"fmt"
	"strings"

	"github.com/whykusanagi/celesteCLI/cmd/celeste/providers"
)

// HandleProvidersCommand handles the /providers command and its subcommands.
// Usage:
//   /providers               - List all providers
//   /providers --tools       - Show only tool-capable providers
//   /providers info <name>   - Show detailed capabilities
//   /providers current       - Show current provider info
func HandleProvidersCommand(cmd *Command, ctx *CommandContext) *CommandResult {
	// Parse subcommand
	if len(cmd.Args) == 0 {
		return listAllProviders(ctx)
	}

	subcommand := cmd.Args[0]

	switch subcommand {
	case "--tools":
		return listToolProviders(ctx)
	case "info":
		if len(cmd.Args) < 2 {
			return &CommandResult{
				Success:      false,
				Message:      "❌ Usage: /providers info <provider_name>",
				ShouldRender: true,
			}
		}
		return showProviderInfo(cmd.Args[1], ctx)
	case "current":
		return showCurrentProvider(ctx)
	default:
		// Check if it's a provider name (for backwards compatibility with "/providers <name>")
		if _, ok := providers.GetProvider(subcommand); ok {
			return showProviderInfo(subcommand, ctx)
		}
		return &CommandResult{
			Success:      false,
			Message:      fmt.Sprintf("❌ Unknown /providers subcommand: %s\n\nAvailable: --tools, info <name>, current", subcommand),
			ShouldRender: true,
		}
	}
}

// listAllProviders displays all registered providers with their capabilities
func listAllProviders(ctx *CommandContext) *CommandResult {
	var output strings.Builder

	// Corrupt header
	output.WriteString("═══════════════════════════════════════════════\n")
	output.WriteString("           可用的 AI PROVIDERS\n")
	output.WriteString("═══════════════════════════════════════════════\n\n")

	allProviders := providers.ListProviders()

	for _, name := range allProviders {
		caps, ok := providers.GetProvider(name)
		if !ok {
			continue
		}

		// Provider status indicator
		status := "✓"
		if ctx.Provider == name {
			status = "▶" // Current provider
		}

		// Tool support indicator
		toolSupport := "[NO TOOLS]"
		if caps.SupportsFunctionCalling {
			toolSupport = "[TOOLS]"
		}

		// Build provider line
		output.WriteString(fmt.Sprintf("%s %-15s %-12s", status, name, toolSupport))

		// Default/preferred model
		if caps.PreferredToolModel != "" {
			output.WriteString(fmt.Sprintf(" %s (preferred)", caps.PreferredToolModel))
		} else if caps.DefaultModel != "" {
			output.WriteString(fmt.Sprintf(" %s (default)", caps.DefaultModel))
		}

		// Special notes
		if caps.BaseURL != "" && strings.Contains(caps.BaseURL, "digitalocean") {
			output.WriteString(" [cloud-only]")
		} else if strings.Contains(name, "vertex") {
			output.WriteString(" [OAuth required]")
		} else if strings.Contains(name, "elevenlabs") {
			output.WriteString(" [voice]")
		} else if strings.Contains(name, "openrouter") {
			output.WriteString(" [aggregator]")
		}

		output.WriteString("\n")
	}

	// Current provider info
	if ctx.Provider != "" {
		output.WriteString(fmt.Sprintf("\nCurrent: %s", ctx.Provider))
		if caps, ok := providers.GetProvider(ctx.Provider); ok {
			if caps.SupportsFunctionCalling {
				output.WriteString(" (function calling enabled)")
			}
		}
		output.WriteString("\n")
	}

	output.WriteString("\nUse: /providers info <name> for details\n")

	return &CommandResult{
		Success:      true,
		Message:      output.String(),
		ShouldRender: true,
	}
}

// listToolProviders displays only providers that support function calling
func listToolProviders(ctx *CommandContext) *CommandResult {
	var output strings.Builder

	output.WriteString("═══════════════════════════════════════════════\n")
	output.WriteString("      TOOL-CAPABLE AI PROVIDERS\n")
	output.WriteString("═══════════════════════════════════════════════\n\n")

	toolProviders := providers.GetToolCallingProviders()

	if len(toolProviders) == 0 {
		output.WriteString("No providers with function calling support found.\n")
	} else {
		for _, name := range toolProviders {
			caps, ok := providers.GetProvider(name)
			if !ok {
				continue
			}

			// Current provider indicator
			status := " "
			if ctx.Provider == name {
				status = "▶"
			}

			output.WriteString(fmt.Sprintf("%s %-15s", status, name))

			// Show preferred tool model
			if caps.PreferredToolModel != "" {
				output.WriteString(fmt.Sprintf(" %s", caps.PreferredToolModel))
			} else if caps.DefaultModel != "" {
				output.WriteString(fmt.Sprintf(" %s", caps.DefaultModel))
			}

			output.WriteString("\n")
		}
	}

	output.WriteString(fmt.Sprintf("\nTotal: %d tool-capable providers\n", len(toolProviders)))

	return &CommandResult{
		Success:      true,
		Message:      output.String(),
		ShouldRender: true,
	}
}

// showProviderInfo displays detailed information about a specific provider
func showProviderInfo(name string, ctx *CommandContext) *CommandResult {
	caps, ok := providers.GetProvider(name)
	if !ok {
		return &CommandResult{
			Success:      false,
			Message:      fmt.Sprintf("❌ Provider '%s' not found.\n\nAvailable providers:\n%s", name, strings.Join(providers.ListProviders(), ", ")),
			ShouldRender: true,
		}
	}

	var output strings.Builder

	// Header
	output.WriteString("═══════════════════════════════════════════════\n")
	output.WriteString(fmt.Sprintf("           PROVIDER: %s\n", strings.ToUpper(name)))
	output.WriteString("═══════════════════════════════════════════════\n\n")

	// Current provider indicator
	if ctx.Provider == name {
		output.WriteString("▶ CURRENT PROVIDER\n\n")
	}

	// API Endpoint
	if caps.BaseURL != "" {
		output.WriteString(fmt.Sprintf("API Endpoint:  %s\n", caps.BaseURL))
	}

	// Capabilities
	output.WriteString("\nCAPABILITIES:\n")
	output.WriteString(fmt.Sprintf("  Function Calling:    %s\n", boolToStatus(caps.SupportsFunctionCalling)))
	output.WriteString(fmt.Sprintf("  Model Listing:       %s\n", boolToStatus(caps.SupportsModelListing)))
	output.WriteString(fmt.Sprintf("  Token Tracking:      %s\n", boolToStatus(caps.SupportsTokenTracking)))
	output.WriteString(fmt.Sprintf("  OpenAI Compatible:   %s\n", boolToStatus(caps.IsOpenAICompatible)))

	// Models
	output.WriteString("\nMODELS:\n")
	if caps.DefaultModel != "" {
		output.WriteString(fmt.Sprintf("  Default:          %s\n", caps.DefaultModel))
	}
	if caps.PreferredToolModel != "" {
		output.WriteString(fmt.Sprintf("  Preferred (Tool): %s\n", caps.PreferredToolModel))
	}

	// Known limitations
	output.WriteString("\nNOTES:\n")
	switch name {
	case "openai":
		output.WriteString("  • Gold standard for function calling\n")
		output.WriteString("  • Full feature support\n")
	case "grok":
		output.WriteString("  • 2M context window\n")
		output.WriteString("  • Fast and reliable\n")
	case "venice":
		output.WriteString("  • Uncensored mode available\n")
		output.WriteString("  • No function calling support\n")
	case "anthropic":
		output.WriteString("  • Use OpenAI compatibility mode\n")
		output.WriteString("  • Native API support planned\n")
	case "gemini":
		output.WriteString("  • Google AI Studio required\n")
		output.WriteString("  • Function calling support via API\n")
	case "vertex":
		output.WriteString("  • Requires OAuth setup\n")
		output.WriteString("  • GCP project required\n")
	case "openrouter":
		output.WriteString("  • Model aggregator\n")
		output.WriteString("  • Function calling varies by model\n")
	case "digitalocean":
		output.WriteString("  • Agent API (cloud-hosted only)\n")
		output.WriteString("  • Requires DigitalOcean account\n")
	case "elevenlabs":
		output.WriteString("  • Voice synthesis API\n")
		output.WriteString("  • Function calling unknown\n")
	default:
		output.WriteString("  • See provider documentation for details\n")
	}

	// Setup instructions
	if name != ctx.Provider {
		output.WriteString(fmt.Sprintf("\nTo use this provider, update your config:\n"))
		if caps.BaseURL != "" {
			output.WriteString(fmt.Sprintf("  base_url: %s\n", caps.BaseURL))
		}
		if caps.DefaultModel != "" {
			output.WriteString(fmt.Sprintf("  model: %s\n", caps.DefaultModel))
		}
	}

	return &CommandResult{
		Success:      true,
		Message:      output.String(),
		ShouldRender: true,
	}
}

// showCurrentProvider displays information about the currently active provider
func showCurrentProvider(ctx *CommandContext) *CommandResult {
	if ctx.Provider == "" {
		return &CommandResult{
			Success:      true,
			Message:      "⚠️ No provider detected.\n\nProvider will be auto-detected from your BaseURL configuration.",
			ShouldRender: true,
		}
	}

	// Reuse the provider info function
	return showProviderInfo(ctx.Provider, ctx)
}

// Helper functions

func boolToStatus(b bool) string {
	if b {
		return "✓ Yes"
	}
	return "✗ No"
}

func formatNumber(n int) string {
	if n >= 1000000 {
		return fmt.Sprintf("%.1fM", float64(n)/1000000)
	}
	if n >= 1000 {
		return fmt.Sprintf("%.1fK", float64(n)/1000)
	}
	return fmt.Sprintf("%d", n)
}
