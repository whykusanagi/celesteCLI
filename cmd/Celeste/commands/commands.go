// Package commands handles slash commands for Celeste CLI.
// Commands provide direct user control over modes, endpoints, and configuration.
package commands

import (
	"fmt"
	"strings"
)

// Command represents a parsed slash command.
type Command struct {
	Name string
	Args []string
	Raw  string
}

// CommandResult represents the result of executing a command.
type CommandResult struct {
	Success      bool
	Message      string
	ShouldRender bool // Whether to show in chat history
	StateChange  *StateChange
}

// StateChange represents a change in application state.
type StateChange struct {
	EndpointChange *string
	NSFWMode       *bool
	Model          *string
	ClearHistory   bool
}

// Parse parses a message to check if it's a command.
// Returns nil if not a command.
func Parse(input string) *Command {
	input = strings.TrimSpace(input)

	if !strings.HasPrefix(input, "/") {
		return nil
	}

	// Split by whitespace
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return nil
	}

	cmd := &Command{
		Name: strings.TrimPrefix(parts[0], "/"),
		Raw:  input,
	}

	if len(parts) > 1 {
		cmd.Args = parts[1:]
	}

	return cmd
}

// Execute executes a command and returns the result.
func Execute(cmd *Command) *CommandResult {
	switch strings.ToLower(cmd.Name) {
	case "nsfw":
		return handleNSFW(cmd)
	case "safe":
		return handleSafe(cmd)
	case "endpoint":
		return handleEndpoint(cmd)
	case "model":
		return handleModel(cmd)
	case "config":
		return handleConfig(cmd)
	case "clear":
		return handleClear(cmd)
	case "help":
		return handleHelp(cmd)
	default:
		return &CommandResult{
			Success:      false,
			Message:      fmt.Sprintf("Unknown command: /%s. Type /help for available commands.", cmd.Name),
			ShouldRender: true,
		}
	}
}

// handleNSFW handles the /nsfw command.
func handleNSFW(cmd *Command) *CommandResult {
	enabled := true
	return &CommandResult{
		Success:      true,
		Message:      "🔥 NSFW Mode Enabled\n\nSwitched to Venice.ai endpoint for uncensored content.\nAll requests will use Venice.ai until you run /safe.\n\nNote: Image generation will use uncensored models.",
		ShouldRender: true,
		StateChange: &StateChange{
			NSFWMode: &enabled,
		},
	}
}

// handleSafe handles the /safe command.
func handleSafe(cmd *Command) *CommandResult {
	disabled := false
	return &CommandResult{
		Success:      true,
		Message:      "✅ Safe Mode Enabled\n\nSwitched back to OpenAI endpoint.\nContent will follow OpenAI usage policies.",
		ShouldRender: true,
		StateChange: &StateChange{
			NSFWMode: &disabled,
		},
	}
}

// handleEndpoint handles the /endpoint command.
func handleEndpoint(cmd *Command) *CommandResult {
	if len(cmd.Args) == 0 {
		return &CommandResult{
			Success:      false,
			Message:      "Usage: /endpoint <name>\n\nAvailable endpoints:\n  • openai\n  • venice\n  • grok\n  • elevenlabs\n  • google (for Vertex AI)\n\nExample: /endpoint venice",
			ShouldRender: true,
		}
	}

	endpoint := strings.ToLower(cmd.Args[0])
	validEndpoints := map[string]string{
		"openai":     "OpenAI",
		"venice":     "Venice.ai",
		"grok":       "xAI Grok",
		"elevenlabs": "ElevenLabs",
		"google":     "Google Vertex AI",
	}

	if displayName, ok := validEndpoints[endpoint]; ok {
		return &CommandResult{
			Success:      true,
			Message:      fmt.Sprintf("🔄 Switched to %s\n\nAll requests will use this endpoint until changed.", displayName),
			ShouldRender: true,
			StateChange: &StateChange{
				EndpointChange: &endpoint,
			},
		}
	}

	return &CommandResult{
		Success:      false,
		Message:      fmt.Sprintf("Unknown endpoint: %s\n\nAvailable: openai, venice, grok, elevenlabs, google", endpoint),
		ShouldRender: true,
	}
}

// handleModel handles the /model command.
func handleModel(cmd *Command) *CommandResult {
	if len(cmd.Args) == 0 {
		return &CommandResult{
			Success:      false,
			Message:      "Usage: /model <name>\n\nCommon models:\n  • gpt-4o-mini\n  • gpt-4o\n  • claude-3-5-sonnet\n  • llama-3.3-70b\n\nExample: /model gpt-4o",
			ShouldRender: true,
		}
	}

	model := strings.Join(cmd.Args, " ")
	return &CommandResult{
		Success:      true,
		Message:      fmt.Sprintf("🤖 Model changed to: %s", model),
		ShouldRender: true,
		StateChange: &StateChange{
			Model: &model,
		},
	}
}

// handleConfig handles the /config command.
func handleConfig(cmd *Command) *CommandResult {
	if len(cmd.Args) == 0 {
		return &CommandResult{
			Success:      false,
			Message:      "Usage: /config <name>\n\nLoads a named configuration profile.\nExample: /config grok",
			ShouldRender: true,
		}
	}

	configName := cmd.Args[0]
	return &CommandResult{
		Success:      true,
		Message:      fmt.Sprintf("⚙️  Loaded config profile: %s", configName),
		ShouldRender: true,
		StateChange: &StateChange{
			EndpointChange: &configName,
		},
	}
}

// handleClear handles the /clear command.
func handleClear(cmd *Command) *CommandResult {
	return &CommandResult{
		Success:      true,
		Message:      "🗑️  Conversation cleared",
		ShouldRender: false,
		StateChange: &StateChange{
			ClearHistory: true,
		},
	}
}

// handleHelp handles the /help command.
func handleHelp(cmd *Command) *CommandResult {
	helpText := `Available Commands:

Mode Control:
  /nsfw              Switch to NSFW mode (Venice.ai, uncensored)
  /safe              Switch to safe mode (OpenAI, content policy)

Endpoint Control:
  /endpoint <name>   Switch to a specific endpoint
                     Options: openai, venice, grok, elevenlabs, google
  /config <name>     Load a named config profile
  /model <name>      Change the model (e.g., gpt-4o, llama-3.3-70b)

Session Control:
  /clear             Clear conversation history
  /help              Show this help message

Examples:
  /nsfw              → Enable uncensored mode
  /endpoint google   → Switch to Google Vertex AI
  /model gpt-4o      → Use GPT-4o model
  /safe              → Return to safe mode

Tip: You can also add keywords like "nsfw" or "uncensored" at the end
of your message for automatic routing while staying in control.`

	return &CommandResult{
		Success:      true,
		Message:      helpText,
		ShouldRender: true,
	}
}

// DetectRoutingHints checks if message contains routing hints.
// Returns suggested endpoint or empty string.
func DetectRoutingHints(message string) string {
	lower := strings.ToLower(message)

	// Check for explicit routing hints
	hints := map[string]string{
		"#nsfw":       "venice",
		"#uncensored": "venice",
		"#venice":     "venice",
		"#explicit":   "venice",
		"#mature":     "venice",
	}

	for hint, endpoint := range hints {
		if strings.Contains(lower, hint) {
			return endpoint
		}
	}

	// Check for contextual hints at end of message
	words := strings.Fields(message)
	if len(words) > 0 {
		lastWord := strings.ToLower(words[len(words)-1])
		contextHints := map[string]string{
			"nsfw":       "venice",
			"uncensored": "venice",
			"explicit":   "venice",
			"lewd":       "venice",
			"mature":     "venice",
		}

		if endpoint, ok := contextHints[lastWord]; ok {
			return endpoint
		}
	}

	return ""
}

// IsImageGenerationRequest checks if the message is requesting image generation.
func IsImageGenerationRequest(message string) bool {
	lower := strings.ToLower(message)

	imageKeywords := []string{
		"generate an image",
		"generate image",
		"create an image",
		"create image",
		"make an image",
		"make image",
		"draw",
		"generate a picture",
		"create a picture",
		"generate art",
		"create art",
	}

	for _, keyword := range imageKeywords {
		if strings.Contains(lower, keyword) {
			return true
		}
	}

	return false
}

// IsContentPolicyRefusal checks if the LLM response is a content policy refusal.
func IsContentPolicyRefusal(response string) bool {
	lower := strings.ToLower(response)

	refusalPatterns := []string{
		"i can't",
		"i cannot",
		"i'm not able to",
		"i'm unable to",
		"against my",
		"content policy",
		"usage policy",
		"i don't feel comfortable",
		"inappropriate",
		"i'm designed to be helpful, harmless, and honest",
	}

	for _, pattern := range refusalPatterns {
		if strings.Contains(lower, pattern) {
			return true
		}
	}

	return false
}
