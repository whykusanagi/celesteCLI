// Package commands handles slash commands for Celeste CLI.
// Commands provide direct user control over modes, endpoints, and configuration.
package commands

import (
	"context"
	"fmt"
	"strings"

	"github.com/whykusanagi/celesteCLI/cmd/Celeste/providers"
)

// Command represents a parsed slash command.
type Command struct {
	Name string
	Args []string
	Raw  string
}

// CommandContext provides context for command execution.
type CommandContext struct {
	NSFWMode      bool
	Provider      string // Current provider (grok, openai, venice, etc.)
	CurrentModel  string // Current model in use
	APIKey        string // API key for model listing
	BaseURL       string // Base URL for API calls
	SkillsEnabled bool   // Whether skills/functions are currently enabled
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
	ImageModel     *string
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
func Execute(cmd *Command, ctx *CommandContext) *CommandResult {
	if ctx == nil {
		ctx = &CommandContext{}
	}

	switch strings.ToLower(cmd.Name) {
	case "nsfw":
		return handleNSFW(cmd)
	case "safe":
		return handleSafe(cmd)
	case "endpoint":
		return handleEndpoint(cmd)
	case "model":
		return handleModel(cmd)
	case "image-model", "set-model", "list-models":
		return handleSetModel(cmd, ctx)
	case "config":
		return handleConfig(cmd)
	case "clear":
		return handleClear(cmd)
	case "help":
		return handleHelp(cmd, ctx)
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
	defaultImageModel := "lustify-sdxl"
	return &CommandResult{
		Success:      true,
		Message:      "🔥 NSFW Mode Enabled\n\nSwitched to Venice.ai endpoint for uncensored content.\nImage Model: lustify-sdxl\n\nUse /set-model <model> to change image model.\nUse /help to see available models and commands.",
		ShouldRender: true,
		StateChange: &StateChange{
			NSFWMode:   &enabled,
			ImageModel: &defaultImageModel,
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

// handleSetModel handles the /set-model and /list-models commands.
// Context-aware: image models in NSFW mode, chat models otherwise.
func handleSetModel(cmd *Command, ctx *CommandContext) *CommandResult {
	// NSFW mode: Handle image models (backward compatibility with Venice pattern)
	if ctx.NSFWMode {
		return handleImageModel(cmd, ctx)
	}

	// Chat mode: Handle chat models with provider awareness
	return handleChatModel(cmd, ctx)
}

// handleImageModel handles image model selection in NSFW mode (Venice pattern).
func handleImageModel(cmd *Command, ctx *CommandContext) *CommandResult {
	if len(cmd.Args) == 0 || cmd.Name == "list-models" {
		return &CommandResult{
			Success:      false,
			Message:      "Available Image Models:\n\n  • lustify-sdxl (default NSFW)\n  • wai-Illustrious (anime)\n  • hidream (dream-like)\n  • nano-banana-pro\n  • venice-sd35 (Stable Diffusion 3.5)\n  • lustify-v7\n\nUsage: /set-model <model-name>\nExample: /set-model wai-Illustrious\n\nOr use shortcuts: anime:, dream:, image:",
			ShouldRender: true,
		}
	}

	imageModel := cmd.Args[0]

	// Validate model name
	validModels := map[string]string{
		"lustify-sdxl":    "NSFW image generation",
		"wai-illustrious": "Anime style",
		"hidream":         "Dream-like quality",
		"nano-banana-pro": "Alternative model",
		"venice-sd35":     "Stable Diffusion 3.5",
		"lustify-v7":      "Lustify v7",
		"qwen-image":      "Qwen vision model",
	}

	modelLower := strings.ToLower(imageModel)
	if desc, ok := validModels[modelLower]; ok {
		return &CommandResult{
			Success:      true,
			Message:      fmt.Sprintf("🎨 Image model changed to: %s\n%s\n\nThis will be used for all image: prompts until changed.", imageModel, desc),
			ShouldRender: true,
			StateChange: &StateChange{
				ImageModel: &imageModel,
			},
		}
	}

	return &CommandResult{
		Success:      false,
		Message:      fmt.Sprintf("Unknown model: %s\n\nUse /set-model without arguments to see available models.", imageModel),
		ShouldRender: true,
	}
}

// handleChatModel handles chat model selection with provider capabilities.
func handleChatModel(cmd *Command, ctx *CommandContext) *CommandResult {
	// Get provider capabilities
	caps, ok := providers.GetProvider(ctx.Provider)
	if !ok {
		return &CommandResult{
			Success:      false,
			Message:      fmt.Sprintf("Unknown provider: %s\n\nUse /endpoint to switch providers.", ctx.Provider),
			ShouldRender: true,
		}
	}

	// No args or /list-models: Show available models
	if len(cmd.Args) == 0 || cmd.Name == "list-models" {
		return listAvailableModels(ctx, caps)
	}

	// Check for --force flag
	forceModel := false
	modelName := cmd.Args[0]
	if len(cmd.Args) > 1 && cmd.Args[1] == "--force" {
		forceModel = true
	}

	// Create model service to validate
	modelService := providers.NewModelService(ctx.APIKey, ctx.BaseURL, ctx.Provider)
	modelInfo, err := modelService.ValidateModel(context.Background(), modelName)

	if err != nil {
		// Model not found, but allow if --force
		if forceModel {
			return &CommandResult{
				Success:      true,
				Message:      fmt.Sprintf("🤖 Model changed to: %s\n⚠️  Model validation unavailable", modelName),
				ShouldRender: true,
				StateChange: &StateChange{
					Model: &modelName,
				},
			}
		}

		return &CommandResult{
			Success:      false,
			Message:      fmt.Sprintf("❌ Model '%s' not found for provider %s\n\nUse /set-model to see available models.\nUse /set-model %s --force to set anyway.", modelName, caps.Name, modelName),
			ShouldRender: true,
		}
	}

	// Model found - check tool support
	if !modelInfo.SupportsTools && ctx.SkillsEnabled {
		if !forceModel {
			return &CommandResult{
				Success:      false,
				Message:      fmt.Sprintf("⚠️  Model '%s' does not support function calling.\n\n%s\n\nSkills will be disabled with this model.\n\n✓ Use /set-model %s for skills support\n  Or proceed with /set-model %s --force", modelName, modelInfo.Description, caps.PreferredToolModel, modelName),
				ShouldRender: true,
			}
		}

		// Forced non-tool model
		return &CommandResult{
			Success:      true,
			Message:      fmt.Sprintf("🤖 Model changed to: %s\n⚠️  Skills disabled - model does not support function calling\n\n%s", modelName, modelInfo.Description),
			ShouldRender: true,
			StateChange: &StateChange{
				Model: &modelName,
			},
		}
	}

	// Model supports tools or skills aren't required
	checkmark := ""
	if modelInfo.SupportsTools {
		checkmark = " ✓"
	}

	return &CommandResult{
		Success:      true,
		Message:      fmt.Sprintf("🤖 Model changed to: %s%s\n\n%s", modelName, checkmark, modelInfo.Description),
		ShouldRender: true,
		StateChange: &StateChange{
			Model: &modelName,
		},
	}
}

// listAvailableModels fetches and displays available models for current provider.
func listAvailableModels(ctx *CommandContext, caps providers.ProviderCapabilities) *CommandResult {
	modelService := providers.NewModelService(ctx.APIKey, ctx.BaseURL, ctx.Provider)

	models, err := modelService.ListModels(context.Background())
	if err != nil {
		// Fallback to common models help
		return &CommandResult{
			Success:      false,
			Message:      fmt.Sprintf("Failed to fetch models from %s\n\n%s\n\nCommon models:\n%s\n\nUsage: /set-model <model-id>", caps.Name, err, getCommonModelsHelp(ctx.Provider)),
			ShouldRender: true,
		}
	}

	// Format model list with capability indicators
	formattedList := providers.FormatModelList(models, true)

	// Add header and usage
	message := fmt.Sprintf("Available Models for %s:\n\n%s\nUsage: /set-model <model-id>", caps.Name, formattedList)

	// Add recommendation
	if caps.PreferredToolModel != "" {
		message += fmt.Sprintf("\n\n💡 Recommended: %s (optimized for skills)", caps.PreferredToolModel)
	}

	return &CommandResult{
		Success:      true,
		Message:      message,
		ShouldRender: true,
	}
}

// getCommonModelsHelp returns static model suggestions when API fails.
func getCommonModelsHelp(provider string) string {
	switch provider {
	case "grok":
		return "  • grok-4-1-fast (recommended for skills)\n  • grok-4-1\n  • grok-beta"
	case "openai":
		return "  • gpt-4o-mini (recommended)\n  • gpt-4o\n  • gpt-4-turbo"
	case "venice":
		return "  • venice-uncensored (no skills)\n  • llama-3.3-70b\n  • qwen3-235b"
	case "anthropic":
		return "  • claude-sonnet-4-5-20250929\n  • claude-opus-4-5-20251101"
	case "vertex":
		return "  • gemini-1.5-pro\n  • gemini-1.5-flash"
	case "openrouter":
		return "  • openai/gpt-4o-mini\n  • anthropic/claude-sonnet-4-5"
	default:
		return "  (provider-specific models)"
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
func handleHelp(cmd *Command, ctx *CommandContext) *CommandResult {
	var helpText string

	if ctx.NSFWMode {
		// NSFW Mode Help
		helpText = `🔥 NSFW Mode - Venice.ai Uncensored

Media Generation Commands:
  image: <prompt>              Generate images with current model
                               Example: image: cyberpunk cityscape at night

  anime: <prompt>              Generate anime-style images (wai-Illustrious)
                               Example: anime: magical girl with sword

  dream: <prompt>              High-quality dream-like images (hidream)
                               Example: dream: surreal landscape

  image[model]: <prompt>       Use specific model for one generation
                               Example: image[nano-banana-pro]: futuristic city

  upscale: <path>              Upscale and enhance existing image
                               Example: upscale: ~/photo.jpg

Model Management:
  /set-model <model>           Set default image generation model
                               Example: /set-model wai-Illustrious
                               Run without args to see all models

Chat Commands:
  /safe                        Return to safe mode (OpenAI)
  /clear                       Clear conversation history
  /help                        Show this help message

Current Configuration:
  • Endpoint: Venice.ai (https://api.venice.ai/api/v1)
  • Chat Model: venice-uncensored (no function calling)
  • Image Model: Use /set-model to configure
  • Downloads: ~/Downloads
  • Quality: 40 steps, CFG 12.0, PNG format

Available Image Models:
  • lustify-sdxl - NSFW image generation (default)
  • wai-Illustrious - Anime style
  • hidream - Dream-like quality
  • nano-banana-pro - Alternative model
  • venice-sd35 - Stable Diffusion 3.5
  • lustify-v7 - Lustify v7
  • qwen-image - Qwen vision model

Image Quality Parameters (defaults):
  • Steps: 40 (1-50, higher = more detail)
  • CFG Scale: 12.0 (0-20, higher = stronger prompt adherence)
  • Size: 1024x1024 (up to 1280x1280)
  • Format: PNG (lossless)
  • Safe Mode: Disabled (no NSFW blurring)

Configure downloads_dir in ~/.celeste/skills.json to change save location.

Tip: Ask the uncensored LLM to write detailed NSFW prompts, then use
"image: [paste prompt]" to generate from that description!`
	} else {
		// Safe Mode Help
		helpText = `Available Commands:

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
  /nsfw              → Enable uncensored mode with media generation
  /endpoint google   → Switch to Google Vertex AI
  /model gpt-4o      → Use GPT-4o model
  /safe              → Return to safe mode

Skills Available: 18 function-calling tools
  • Weather, currency, timezone conversion
  • Hashing, encoding, UUID generation
  • Twitch live checks, YouTube videos
  • Reminders, notes, tarot readings
  • QR codes, passwords

Tip: You can also add keywords like "nsfw" or "uncensored" at the end
of your message for automatic routing while staying in control.`
	}

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
