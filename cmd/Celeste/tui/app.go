// Package tui provides the Bubble Tea-based terminal UI for Celeste CLI.
// This file contains the main application model and layout logic.
package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/whykusanagi/celesteCLI/cmd/Celeste/commands"
	"github.com/whykusanagi/celesteCLI/cmd/Celeste/config"
	"github.com/whykusanagi/celesteCLI/cmd/Celeste/venice"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Typing speed: ~25 chars/sec for smooth, visible corruption effects
const charsPerTick = 2
const typingTickInterval = 80 * time.Millisecond

// AppModel is the root model for the Celeste TUI application.
type AppModel struct {
	// Sub-components
	header HeaderModel
	chat   ChatModel
	input  InputModel
	skills SkillsModel
	status StatusModel

	// Application state
	width      int
	height     int
	ready      bool
	nsfwMode   bool
	streaming  bool
	endpoint   string // Current endpoint (openai, venice, grok, etc.)
	model      string // Current model name
	imageModel string // Current image generation model (for NSFW mode)

	// Simulated typing state
	typingContent string // Full content to type
	typingPos     int    // Current position in content
	animFrame     int    // Animation frame counter

	// Pending tool call tracking
	pendingToolCallID string // Track tool call ID for sending result back to LLM

	// LLM client (injected)
	llmClient LLMClient

	// Session persistence (optional)
	sessionManager SessionManager
	currentSession Session
}

// LLMClient interface for sending messages to the LLM.
type LLMClient interface {
	SendMessage(messages []ChatMessage, tools []SkillDefinition) tea.Cmd
	GetSkills() []SkillDefinition
	ExecuteSkill(name string, args map[string]any, toolCallID string) tea.Cmd
}

// EndpointSwitcher interface for clients that support dynamic endpoint switching.
type EndpointSwitcher interface {
	SwitchEndpoint(endpoint string) error
	ChangeModel(model string) error
}

// SkillDefinition represents a skill/function that can be called.
type SkillDefinition struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

// VeniceConfigData holds Venice.ai configuration from skills.json.
type VeniceConfigData struct {
	APIKey     string
	BaseURL    string
	Model      string // Chat model
	ImageModel string // Image generation model
}

// loadVeniceConfig loads Venice configuration from ~/.celeste/skills.json.
func loadVeniceConfig() (VeniceConfigData, error) {
	// Load skills config
	skillsConfig, err := config.LoadSkillsConfig()
	if err != nil {
		return VeniceConfigData{}, fmt.Errorf("failed to load skills config: %w", err)
	}

	// Create config loader
	loader := config.NewConfigLoader(skillsConfig)

	// Get Venice config via loader
	veniceConfig, err := loader.GetVeniceConfig()
	if err != nil {
		return VeniceConfigData{}, err
	}

	return VeniceConfigData{
		APIKey:     veniceConfig.APIKey,
		BaseURL:    veniceConfig.BaseURL,
		Model:      veniceConfig.Model,
		ImageModel: veniceConfig.ImageModel,
	}, nil
}

// NewApp creates a new TUI application model.
func NewApp(llmClient LLMClient) AppModel {
	skills := []SkillDefinition{}
	if llmClient != nil {
		skills = llmClient.GetSkills()
	}

	return AppModel{
		header:    NewHeaderModel(),
		chat:      NewChatModel(),
		input:     NewInputModel(),
		skills:    NewSkillsModel(skills),
		status:    NewStatusModel(),
		llmClient: llmClient,
	}
}

// Init implements tea.Model.
func (m AppModel) Init() tea.Cmd {
	return tea.Batch(
		m.input.Init(),
		tea.EnterAltScreen,
	)
}

// Update implements tea.Model.
func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "pgup", "pgdown", "shift+up", "shift+down":
			// Scrolling keys go to chat
			var cmd tea.Cmd
			m.chat, cmd = m.chat.Update(msg)
			cmds = append(cmds, cmd)
		default:
			// Other keys go to input
			var cmd tea.Cmd
			m.input, cmd = m.input.Update(msg)
			cmds = append(cmds, cmd)
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true

		// Calculate component heights - compact layout
		headerHeight := 1
		inputHeight := 2
		skillsHeight := 3 // Reduced from 5
		statusHeight := 1
		chatHeight := m.height - headerHeight - inputHeight - skillsHeight - statusHeight

		// Ensure minimum chat height
		if chatHeight < 5 {
			chatHeight = 5
			skillsHeight = 2 // Further reduce skills if needed
		}

		// Update component sizes
		m.header = m.header.SetWidth(m.width)
		m.chat = m.chat.SetSize(m.width, chatHeight)
		m.input = m.input.SetWidth(m.width)
		m.skills = m.skills.SetSize(m.width, skillsHeight)
		m.status = m.status.SetWidth(m.width)

	case SendMessageMsg:
		content := strings.TrimSpace(msg.Content)

		// Check if it's a slash command first
		if cmd := commands.Parse(content); cmd != nil {
			// Create context with current state
			ctx := &commands.CommandContext{
				NSFWMode: m.nsfwMode,
			}
			result := commands.Execute(cmd, ctx)

			// Show command result message if needed
			if result.ShouldRender {
				m.chat = m.chat.AddSystemMessage(result.Message)
			}

			// Apply state changes
			if result.StateChange != nil {
				if result.StateChange.EndpointChange != nil {
					m.endpoint = *result.StateChange.EndpointChange
					m.header = m.header.SetEndpoint(m.endpoint)
					m.status = m.status.SetText(fmt.Sprintf("Switched to %s", m.endpoint))

					// Actually switch the LLM client endpoint
					if switcher, ok := m.llmClient.(EndpointSwitcher); ok {
						if err := switcher.SwitchEndpoint(m.endpoint); err != nil {
							m.status = m.status.SetText(fmt.Sprintf("Error switching endpoint: %v", err))
						}
					}

					// Persist session state
					m.persistSession()
				}
				if result.StateChange.NSFWMode != nil {
					m.nsfwMode = *result.StateChange.NSFWMode
					m.header = m.header.SetNSFWMode(m.nsfwMode)
					// When NSFW mode is enabled, automatically switch to Venice
					if m.nsfwMode {
						m.endpoint = "venice"
						m.header = m.header.SetEndpoint(m.endpoint)

						// Actually switch the LLM client to Venice
						if switcher, ok := m.llmClient.(EndpointSwitcher); ok {
							if err := switcher.SwitchEndpoint(m.endpoint); err != nil {
								m.status = m.status.SetText(fmt.Sprintf("Error switching to Venice: %v", err))
							}
						}
					}

					// Persist session state
					m.persistSession()
				}
				if result.StateChange.Model != nil {
					m.model = *result.StateChange.Model
					m.header = m.header.SetModel(m.model)
					m.status = m.status.SetText(fmt.Sprintf("Model changed to %s", m.model))

					// Actually change the model
					if switcher, ok := m.llmClient.(EndpointSwitcher); ok {
						if err := switcher.ChangeModel(m.model); err != nil {
							m.status = m.status.SetText(fmt.Sprintf("Error changing model: %v", err))
						}
					}

					// Persist session state
					m.persistSession()
				}
				if result.StateChange.ImageModel != nil {
					m.imageModel = *result.StateChange.ImageModel
					m.header = m.header.SetImageModel(m.imageModel)
					m.status = m.status.SetText(fmt.Sprintf("🎨 Image model: %s", m.imageModel))

					// Persist session state
					m.persistSession()
				}
				if result.StateChange.ClearHistory {
					m.chat = m.chat.Clear()
				}
			}

			return m, nil
		}

		// Handle legacy text commands (for backward compatibility)
		lowerContent := strings.ToLower(content)
		switch lowerContent {
		case "exit", "quit", "q", ":q", ":quit", ":exit":
			return m, tea.Quit
		case "clear":
			m.chat = m.chat.Clear()
			m.status = m.status.SetText("Chat cleared")
			return m, nil
		case "help":
			// Use context-aware /help command instead of static helpText()
			helpCmd := &commands.Command{Name: "help"}
			ctx := &commands.CommandContext{NSFWMode: m.nsfwMode}
			result := commands.Execute(helpCmd, ctx)
			if result.Success {
				m.chat = m.chat.AddSystemMessage(result.Message)
			}
			return m, nil
		case "tools", "skills", "debug":
			// Show tools/skills debug info
			skills := m.skills.GetDefinitions()
			debugMsg := fmt.Sprintf("📋 Available Tools (%d):\n", len(skills))
			for _, s := range skills {
				debugMsg += fmt.Sprintf("  • %s: %s\n", s.Name, s.Description)
			}
			debugMsg += "\n⚠️  Note: DigitalOcean GenAI Agents may not support function calling.\n"
			debugMsg += "Tool calls only work with OpenAI-compatible APIs that support the 'tools' parameter.\n"
			debugMsg += fmt.Sprintf("\nLog file: %s", GetLogPath())
			m.chat = m.chat.AddSystemMessage(debugMsg)
			return m, nil
		}

		// Check for routing hints (hashtags or keywords at end)
		suggestedEndpoint := commands.DetectRoutingHints(content)
		if suggestedEndpoint != "" && suggestedEndpoint != m.endpoint {
			// Auto-route based on hints
			m.endpoint = suggestedEndpoint
			m.header = m.header.SetEndpoint(m.endpoint)
			m.header = m.header.SetAutoRouted(true)
			m.status = m.status.SetText(fmt.Sprintf("🔀 Auto-routed to %s", suggestedEndpoint))

			// Actually switch the LLM client endpoint
			if switcher, ok := m.llmClient.(EndpointSwitcher); ok {
				if err := switcher.SwitchEndpoint(m.endpoint); err != nil {
					m.status = m.status.SetText(fmt.Sprintf("Error auto-routing: %v", err))
				}
			}

			// Persist session state
			m.persistSession()
		} else {
			m.header = m.header.SetAutoRouted(false)
		}

		// Check for Venice media commands in NSFW mode
		if m.nsfwMode {
			LogInfo(fmt.Sprintf("Checking for media command in: '%s'", content))
			mediaType, prompt, params, isMediaCmd := venice.ParseMediaCommand(content)
			LogInfo(fmt.Sprintf("ParseMediaCommand result: isMediaCmd=%v, mediaType=%s, prompt='%s'", isMediaCmd, mediaType, prompt))

			if isMediaCmd {
				// Handle media generation directly (bypass LLM)
				LogInfo(fmt.Sprintf("✓ Detected %s media command, bypassing LLM", mediaType))
				m.chat = m.chat.AddUserMessage(content)
				m.chat = m.chat.AddAssistantMessage(fmt.Sprintf("🎨 Generating %s... please wait", mediaType))
				m.status = m.status.SetText(fmt.Sprintf("⏳ Venice.ai %s generation in progress...", mediaType))

				// Trigger async media generation
				cmds = append(cmds, func() tea.Msg {
					return GenerateMediaMsg{
						MediaType:  mediaType,
						Prompt:     prompt,
						Params:     params,
						ImageModel: m.imageModel, // Pass current image model from app state
					}
				})
				return m, tea.Batch(cmds...)
			} else {
				LogInfo("No media command detected, sending to LLM chat")
			}
		}

		// Add user message to chat
		m.chat = m.chat.AddUserMessage(content)
		m.streaming = true
		m.status = m.status.SetStreaming(true)
		m.status = m.status.SetText(StreamingSpinner(0) + " " + ThinkingAnimation(0))

		// Send to LLM and start animation
		if m.llmClient != nil {
			// In NSFW mode, don't send skills (Venice uncensored doesn't support function calling)
			var toolsToSend []SkillDefinition
			if !m.nsfwMode {
				toolsToSend = m.skills.GetDefinitions()
			}

			cmds = append(cmds, m.llmClient.SendMessage(m.chat.GetMessages(), toolsToSend))
			// Start animation tick for waiting state
			cmds = append(cmds, tea.Tick(typingTickInterval*2, func(t time.Time) tea.Msg {
				return TickMsg{Time: t}
			}))
		}

	case GenerateMediaMsg:
		// Generate media asynchronously via Venice.ai
		LogInfo(fmt.Sprintf("→ Starting %s generation with prompt: '%s'", msg.MediaType, msg.Prompt))
		cmds = append(cmds, func() tea.Msg {
			// Load Venice config from skills.json
			LogInfo("Loading Venice config from skills.json")
			veniceConfig, err := loadVeniceConfig()
			if err != nil {
				LogInfo(fmt.Sprintf("❌ Failed to load Venice config: %v", err))
				return MediaResultMsg{
					Success:   false,
					Error:     fmt.Sprintf("Failed to load Venice config: %v", err),
					MediaType: msg.MediaType,
				}
			}
			LogInfo(fmt.Sprintf("✓ Loaded Venice config: baseURL=%s, imageModel=%s", veniceConfig.BaseURL, veniceConfig.ImageModel))

			// Create config with appropriate model for the media type
			modelToUse := veniceConfig.Model // Default to chat model
			if msg.MediaType == "image" {
				// Use app's image model if set, otherwise fall back to config
				if msg.ImageModel != "" {
					modelToUse = msg.ImageModel
					LogInfo(fmt.Sprintf("Using app image model: %s", modelToUse))
				} else {
					modelToUse = veniceConfig.ImageModel
					LogInfo(fmt.Sprintf("Using config image model: %s", modelToUse))
				}
			}
			LogInfo(fmt.Sprintf("Using model: %s for %s generation", modelToUse, msg.MediaType))

			config := venice.Config{
				APIKey:  veniceConfig.APIKey,
				BaseURL: veniceConfig.BaseURL,
				Model:   modelToUse,
			}

			var response *venice.MediaResponse
			var genErr error

			LogInfo(fmt.Sprintf("Calling Venice.ai API for %s generation", msg.MediaType))
			switch msg.MediaType {
			case "image":
				response, genErr = venice.GenerateImage(config, msg.Prompt, msg.Params)
			case "video":
				response, genErr = venice.GenerateVideo(config, msg.Prompt, msg.Params)
			case "upscale":
				if path, ok := msg.Params["path"].(string); ok {
					response, genErr = venice.UpscaleImage(config, path, msg.Params)
				} else {
					genErr = fmt.Errorf("no image path provided for upscale")
				}
			case "image-to-video":
				if path, ok := msg.Params["path"].(string); ok {
					response, genErr = venice.ImageToVideo(config, path, msg.Params)
				} else {
					genErr = fmt.Errorf("no image path provided for image-to-video")
				}
			default:
				genErr = fmt.Errorf("unknown media type: %s", msg.MediaType)
			}

			if genErr != nil {
				LogInfo(fmt.Sprintf("❌ Media generation error: %v", genErr))
				return MediaResultMsg{
					Success:   false,
					Error:     genErr.Error(),
					MediaType: msg.MediaType,
				}
			}

			if response == nil {
				LogInfo("❌ Response is nil")
				return MediaResultMsg{
					Success:   false,
					Error:     "No response from Venice API",
					MediaType: msg.MediaType,
				}
			}

			if !response.Success {
				LogInfo(fmt.Sprintf("❌ Response failed: %s", response.Error))
				return MediaResultMsg{
					Success:   false,
					Error:     response.Error,
					MediaType: msg.MediaType,
				}
			}

			LogInfo(fmt.Sprintf("✓ Media generation successful: URL=%s, Path=%s", response.URL, response.Path))
			return MediaResultMsg{
				Success:   true,
				URL:       response.URL,
				Path:      response.Path,
				MediaType: msg.MediaType,
			}
		})

	case MediaResultMsg:
		// Handle media generation result
		LogInfo(fmt.Sprintf("Received MediaResultMsg: success=%v, mediaType=%s", msg.Success, msg.MediaType))
		if msg.Success {
			var resultText string
			if msg.URL != "" {
				LogInfo(fmt.Sprintf("✓ Media generation SUCCESS: URL=%s", msg.URL))
				resultText = fmt.Sprintf("✅ %s generated successfully!\n\n🔗 URL: %s", msg.MediaType, msg.URL)
			} else if msg.Path != "" {
				LogInfo(fmt.Sprintf("✓ Media generation SUCCESS: Path=%s", msg.Path))
				resultText = fmt.Sprintf("✅ %s generated successfully!\n\n💾 Saved to: %s", msg.MediaType, msg.Path)
			} else {
				LogInfo("✓ Media generation SUCCESS (no URL/Path)")
				resultText = fmt.Sprintf("✅ %s generated successfully!", msg.MediaType)
			}

			// Update the last assistant message with the result
			m.chat = m.chat.SetLastAssistantContent(resultText)
			m.status = m.status.SetText(fmt.Sprintf("✓ %s complete", msg.MediaType))
		} else {
			LogInfo(fmt.Sprintf("✗ Media generation FAILED: %s", msg.Error))
			errorText := fmt.Sprintf("❌ %s generation failed: %s", msg.MediaType, msg.Error)
			m.chat = m.chat.SetLastAssistantContent(errorText)
			m.status = m.status.SetText(fmt.Sprintf("✗ %s failed", msg.MediaType))
		}
		m.streaming = false
		m.status = m.status.SetStreaming(false)

	case StreamChunkMsg:
		m.chat = m.chat.AppendToLastAssistant(msg.Chunk.Content)
		if msg.Chunk.IsFirst {
			m.chat = m.chat.AddAssistantMessage("")
		}
		cmds = append(cmds, nil) // Keep processing

	case StreamDoneMsg:
		if msg.FullContent != "" {
			// Check for content policy refusal
			if commands.IsContentPolicyRefusal(msg.FullContent) && m.endpoint != "venice" {
				// Detected refusal - offer to switch to Venice
				m.chat = m.chat.AddSystemMessage(
					"⚠️  Content policy refusal detected.\n\n" +
						"💡 Tip: Use /nsfw to switch to Venice.ai for uncensored responses,\n" +
						"or add 'nsfw' at the end of your message for auto-routing.",
				)
				m.streaming = false
				m.status = m.status.SetStreaming(false)
				m.status = m.status.SetText("Content policy refusal - use /nsfw")

				// Still show the original response
				m.typingContent = msg.FullContent
				m.typingPos = 0
				m.chat = m.chat.AddAssistantMessage("")
				cmds = append(cmds, tea.Tick(typingTickInterval, func(t time.Time) tea.Msg {
					return TickMsg{Time: t}
				}))
			} else {
				// Normal response - start simulated typing
				m.typingContent = msg.FullContent
				m.typingPos = 0
				m.chat = m.chat.AddAssistantMessage("") // Start with empty message
				m.status = m.status.SetText("Typing...")
				// Schedule first typing tick
				cmds = append(cmds, tea.Tick(typingTickInterval, func(t time.Time) tea.Msg {
					return TickMsg{Time: t}
				}))
			}
		} else {
			m.streaming = false
			m.status = m.status.SetStreaming(false)
			m.status = m.status.SetText(fmt.Sprintf("Done (%s)", msg.FinishReason))
		}

	case StreamErrorMsg:
		m.streaming = false
		m.status = m.status.SetStreaming(false)
		m.status = m.status.SetText(fmt.Sprintf("Error: %v", msg.Err))
		m.chat = m.chat.AddSystemMessage(fmt.Sprintf("Error: %v", msg.Err))

	case SkillCallMsg:
		// Log the skill call for debugging
		LogSkillCall(msg.Call.Name, msg.Call.Arguments)
		LogInfo(fmt.Sprintf("Starting execution of skill: %s", msg.Call.Name))
		m.skills = m.skills.SetExecuting(msg.Call.Name)
		m.chat = m.chat.AddFunctionCall(msg.Call)
		m.status = m.status.SetText(fmt.Sprintf("⚡ Executing: %s", msg.Call.Name))

		// Store tool call ID for sending result back to LLM
		m.pendingToolCallID = msg.ToolCallID

		// Add assistant message with tool_calls to conversation (required by OpenAI API)
		// The assistant message must precede the tool result message
		// Convert ToolCallInfo to the format needed
		m.chat = m.chat.AddAssistantMessageWithToolCalls(msg.AssistantContent, msg.ToolCalls)

		// Execute the skill asynchronously
		if m.llmClient != nil {
			cmds = append(cmds, m.llmClient.ExecuteSkill(msg.Call.Name, msg.Call.Arguments, msg.ToolCallID))
		}

	case SkillResultMsg:
		// Log the skill result
		LogSkillResult(msg.Name, msg.Result, msg.Err)
		if msg.Err != nil {
			m.skills = m.skills.SetError(msg.Name, msg.Err)
			m.chat = m.chat.UpdateFunctionResult(msg.Name, fmt.Sprintf("Error: %v", msg.Err))

			// IMPORTANT: Send error result back to LLM so conversation can continue
			// The LLM needs to receive a tool result message even for errors
			if m.llmClient != nil && msg.ToolCallID != "" {
				// Format error as JSON for LLM to interpret
				// Escape quotes and newlines in error message
				errorMsg := strings.ReplaceAll(msg.Err.Error(), `"`, `\"`)
				errorMsg = strings.ReplaceAll(errorMsg, "\n", "\\n")
				errorResult := fmt.Sprintf(`{"error": true, "message": "%s", "skill": "%s"}`, errorMsg, msg.Name)

				// Add tool result as a "tool" message to chat (even for errors)
				m.chat = m.chat.AddToolResult(msg.ToolCallID, msg.Name, errorResult)

				// Send updated conversation back to LLM for interpretation
				m.streaming = true
				m.status = m.status.SetStreaming(true)
				m.status = m.status.SetText(StreamingSpinner(0) + " " + ThinkingAnimation(0))

				// In NSFW mode, don't send skills
				var toolsToSend []SkillDefinition
				if !m.nsfwMode {
					toolsToSend = m.skills.GetDefinitions()
				}
				cmds = append(cmds, m.llmClient.SendMessage(m.chat.GetMessages(), toolsToSend))

				// Start animation tick
				cmds = append(cmds, tea.Tick(typingTickInterval*2, func(t time.Time) tea.Msg {
					return TickMsg{Time: t}
				}))

				// Clear pending tool call ID
				m.pendingToolCallID = ""
			}
		} else {
			m.skills = m.skills.SetCompleted(msg.Name)
			m.chat = m.chat.UpdateFunctionResult(msg.Name, msg.Result)

			// Handle NSFW mode toggle
			if msg.Name == "nsfw_mode" && strings.Contains(msg.Result, "enabled") {
				m.nsfwMode = true
				m.header = m.header.SetNSFWMode(true)
			} else if msg.Name == "nsfw_mode" && strings.Contains(msg.Result, "disabled") {
				m.nsfwMode = false
				m.header = m.header.SetNSFWMode(false)
			}

			// For successful skill results, send result back to LLM for interpretation
			// Add tool result message to conversation and send to LLM
			if m.llmClient != nil && msg.ToolCallID != "" {
				// Add tool result as a "tool" message to chat
				m.chat = m.chat.AddToolResult(msg.ToolCallID, msg.Name, msg.Result)

				// Send updated conversation back to LLM for interpretation
				m.streaming = true
				m.status = m.status.SetStreaming(true)
				m.status = m.status.SetText(StreamingSpinner(0) + " " + ThinkingAnimation(0))

				// In NSFW mode, don't send skills
				var toolsToSend []SkillDefinition
				if !m.nsfwMode {
					toolsToSend = m.skills.GetDefinitions()
				}
				cmds = append(cmds, m.llmClient.SendMessage(m.chat.GetMessages(), toolsToSend))

				// Start animation tick
				cmds = append(cmds, tea.Tick(typingTickInterval*2, func(t time.Time) tea.Msg {
					return TickMsg{Time: t}
				}))

				// Clear pending tool call ID
				m.pendingToolCallID = ""
			}
		}

	case SimulateTypingMsg:
		// For simulated streaming (when endpoint dumps all at once)
		displayed := msg.Content[:msg.CharsToShow]
		m.chat = m.chat.SetLastAssistantContent(displayed)
		if msg.CharsToShow < len(msg.Content) {
			// Schedule next typing tick
			cmds = append(cmds, Tick(typingDelay))
		} else {
			m.streaming = false
			m.status = m.status.SetStreaming(false)
		}

	case TickMsg:
		m.animFrame++

		// Handle simulated typing
		if m.typingContent != "" && m.typingPos < len(m.typingContent) {
			// Advance typing position
			m.typingPos += charsPerTick
			if m.typingPos > len(m.typingContent) {
				m.typingPos = len(m.typingContent)
			}

			// Update chat with current typed content + corruption at cursor
			displayed := m.typingContent[:m.typingPos]
			if m.typingPos < len(m.typingContent) {
				// Add corruption effect at typing cursor
				displayed += GetRandomCorruption()
			}
			m.chat = m.chat.SetLastAssistantContent(displayed)

			// Update status with corrupted animation
			m.status = m.status.SetText(StreamingSpinner(m.animFrame) + " " + ThinkingAnimation(m.animFrame))

			if m.typingPos < len(m.typingContent) {
				// Schedule next typing tick
				cmds = append(cmds, tea.Tick(typingTickInterval, func(t time.Time) tea.Msg {
					return TickMsg{Time: t}
				}))
			} else {
				// Typing complete - show final content without corruption
				m.chat = m.chat.SetLastAssistantContent(m.typingContent)
				m.typingContent = ""
				m.typingPos = 0
				m.streaming = false
				m.status = m.status.SetStreaming(false)
				m.status = m.status.SetText("Ready")
			}
		} else if m.streaming {
			// Just streaming (waiting for response) - show animated status
			m.status = m.status.SetText(StreamingSpinner(m.animFrame) + " " + ThinkingAnimation(m.animFrame))
			cmds = append(cmds, tea.Tick(typingTickInterval*2, func(t time.Time) tea.Msg {
				return TickMsg{Time: t}
			}))
		}

	case NSFWToggleMsg:
		m.nsfwMode = msg.Enabled
		m.header = m.header.SetNSFWMode(msg.Enabled)

	case ErrorMsg:
		m.status = m.status.SetText(fmt.Sprintf("Error: %v", msg.Err))
	}

	return m, tea.Batch(cmds...)
}

// View implements tea.Model.
func (m AppModel) View() string {
	if !m.ready {
		return "\n  Initializing..."
	}

	// Build the layout vertically
	var sections []string

	// Header (fixed, 1 line)
	sections = append(sections, m.header.View())

	// Chat panel (flexible height)
	sections = append(sections, m.chat.View())

	// Input panel (fixed, 3 lines)
	sections = append(sections, m.input.View())

	// Skills panel (fixed, 5 lines)
	sections = append(sections, m.skills.View())

	// Status bar (fixed, 1 line)
	sections = append(sections, m.status.View())

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

// SetLLMClient sets the LLM client.
func (m AppModel) SetLLMClient(client LLMClient) AppModel {
	m.llmClient = client
	if client != nil {
		m.skills = NewSkillsModel(client.GetSkills())
	}
	return m
}

// SessionManager interface for session persistence (avoid circular import).
type SessionManager interface {
	NewSession() Session
	Save(session Session) error
	Load(id string) (Session, error)
}

// Session interface for session data (avoid circular import).
type Session interface {
	SetEndpoint(endpoint string)
	GetEndpoint() string
	SetModel(model string)
	GetModel() string
	SetNSFWMode(enabled bool)
	GetNSFWMode() bool
}

// SetSessionManager sets the session manager for persistence.
func (m AppModel) SetSessionManager(sm SessionManager, session Session) AppModel {
	m.sessionManager = sm
	m.currentSession = session

	// Restore endpoint/model from session if available
	if session != nil {
		if endpoint := session.GetEndpoint(); endpoint != "" {
			m.endpoint = endpoint
			m.header = m.header.SetEndpoint(endpoint)
		}
		if model := session.GetModel(); model != "" {
			m.model = model
			m.header = m.header.SetModel(model)
		}
		m.nsfwMode = session.GetNSFWMode()
		m.header = m.header.SetNSFWMode(m.nsfwMode)
	}

	return m
}

// persistSession saves the current session state.
func (m *AppModel) persistSession() {
	if m.sessionManager == nil || m.currentSession == nil {
		return
	}

	m.currentSession.SetEndpoint(m.endpoint)
	m.currentSession.SetModel(m.model)
	m.currentSession.SetNSFWMode(m.nsfwMode)

	// Save asynchronously (ignore errors for now)
	go m.sessionManager.Save(m.currentSession)
}

// --- Header Model ---

// HeaderModel represents the header bar.
type HeaderModel struct {
	width      int
	nsfwMode   bool
	endpoint   string
	model      string
	imageModel string // Image generation model (NSFW mode)
	autoRouted bool   // Whether the last message was auto-routed
}

// NewHeaderModel creates a new header model.
func NewHeaderModel() HeaderModel {
	return HeaderModel{endpoint: "openai"} // Default endpoint
}

// SetWidth sets the header width.
func (m HeaderModel) SetWidth(width int) HeaderModel {
	m.width = width
	return m
}

// SetNSFWMode sets the NSFW mode indicator.
func (m HeaderModel) SetNSFWMode(enabled bool) HeaderModel {
	m.nsfwMode = enabled
	return m
}

// SetEndpoint sets the current endpoint.
func (m HeaderModel) SetEndpoint(endpoint string) HeaderModel {
	m.endpoint = endpoint
	return m
}

// SetModel sets the current model.
func (m HeaderModel) SetModel(model string) HeaderModel {
	m.model = model
	return m
}

// SetImageModel sets the current image generation model.
func (m HeaderModel) SetImageModel(model string) HeaderModel {
	m.imageModel = model
	return m
}

// SetAutoRouted sets whether auto-routing occurred.
func (m HeaderModel) SetAutoRouted(routed bool) HeaderModel {
	m.autoRouted = routed
	return m
}

// View renders the header.
func (m HeaderModel) View() string {
	title := HeaderTitleStyle.Render("✨ Celeste CLI")

	// Build endpoint/mode indicator
	var endpointInfo string
	if m.nsfwMode {
		endpointInfo = NSFWStyle.Render("🔥 NSFW")
		// Show image model if set
		if m.imageModel != "" {
			endpointInfo += " • " + ModelStyle.Render("img:" + m.imageModel)
		}
	} else if m.endpoint != "" && m.endpoint != "openai" {
		// Show non-default endpoint
		endpointDisplay := map[string]string{
			"venice":     "Venice.ai",
			"grok":       "Grok",
			"elevenlabs": "ElevenLabs",
			"google":     "Google",
		}
		display := endpointDisplay[m.endpoint]
		if display == "" {
			display = m.endpoint
		}
		endpointInfo = EndpointStyle.Render(display)
		if m.autoRouted {
			endpointInfo = "🔀 " + endpointInfo
		}
	}

	// Add model info if set (and not in NSFW mode, as it shows chat model separately)
	if m.model != "" && !m.nsfwMode {
		if endpointInfo != "" {
			endpointInfo += " • "
		}
		endpointInfo += ModelStyle.Render(m.model)
	}

	info := HeaderInfoStyle.Render("Press Ctrl+C to exit")
	if endpointInfo != "" {
		info = endpointInfo + " • " + info
	}

	// Calculate gap
	gap := m.width - lipgloss.Width(title) - lipgloss.Width(info) - 2
	if gap < 1 {
		gap = 1
	}
	spacer := strings.Repeat("─", gap)

	return HeaderStyle.Width(m.width).Render(
		title + spacer + info,
	)
}

// --- Status Model ---

// StatusModel represents the status bar.
type StatusModel struct {
	width     int
	text      string
	streaming bool
	frame     int
}

// NewStatusModel creates a new status model.
func NewStatusModel() StatusModel {
	return StatusModel{text: "Ready"}
}

// SetWidth sets the status bar width.
func (m StatusModel) SetWidth(width int) StatusModel {
	m.width = width
	return m
}

// SetText sets the status text.
func (m StatusModel) SetText(text string) StatusModel {
	m.text = text
	return m
}

// SetStreaming sets the streaming indicator.
func (m StatusModel) SetStreaming(streaming bool) StatusModel {
	m.streaming = streaming
	return m
}

// Update handles tick messages for animation.
func (m StatusModel) Update(msg tea.Msg) (StatusModel, tea.Cmd) {
	if _, ok := msg.(TickMsg); ok {
		m.frame++
	}
	return m, nil
}

// View renders the status bar.
func (m StatusModel) View() string {
	var status string
	if m.streaming {
		// Animated spinner
		frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
		spinner := StatusStreamingStyle.Render(frames[m.frame%len(frames)])
		status = spinner + " " + StatusStreamingStyle.Render("Streaming...")
	} else {
		status = StatusActiveStyle.Render("●") + " " + m.text
	}

	return StatusBarStyle.Width(m.width).Render(status)
}

// --- Helper functions ---

func helpText() string {
	return `
Slash Commands:
  /help              - Show this help
  /clear             - Clear chat history
  /nsfw              - Switch to NSFW mode (Venice.ai, uncensored)
  /safe              - Switch to safe mode (OpenAI)
  /endpoint <name>   - Switch endpoint (openai, venice, grok, elevenlabs, google)
  /model <name>      - Change model (e.g., gpt-4o, llama-3.3-70b)
  /config <name>     - Load a named config profile

Legacy Commands:
  help, clear, exit, quit, tools, skills, debug

Keyboard shortcuts:
  Ctrl+C     - Exit immediately
  PgUp/PgDn  - Scroll chat history
  Shift+↑/↓  - Scroll chat history
  ↑/↓        - Navigate input history

Auto-Routing:
  Add keywords at the end of your message for automatic routing:
  • "nsfw" or "#nsfw" - Routes to Venice.ai
  • "uncensored" - Routes to Venice.ai
  • "explicit" - Routes to Venice.ai

  Example: "Generate an image of a dragon nsfw"
           → Automatically routes to Venice.ai
`
}

// Run starts the TUI application.
func Run(llmClient LLMClient) error {
	p := tea.NewProgram(
		NewApp(llmClient),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	_, err := p.Run()
	return err
}

// Typing delay for simulated streaming (40 chars/sec = 25ms per char)
const typingDelay = 25 * 1000000 // 25ms in nanoseconds
