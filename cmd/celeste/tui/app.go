// Package tui provides the Bubble Tea-based terminal UI for Celeste CLI.
// This file contains the main application model and layout logic.
package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/whykusanagi/celesteCLI/cmd/celeste/commands"
	"github.com/whykusanagi/celesteCLI/cmd/celeste/config"
	"github.com/whykusanagi/celesteCLI/cmd/celeste/providers"
	"github.com/whykusanagi/celesteCLI/cmd/celeste/venice"
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
	width         int
	height        int
	ready         bool
	nsfwMode      bool
	streaming     bool
	endpoint      string // Current endpoint (openai, venice, grok, etc.)
	safeEndpoint  string // Endpoint to return to when leaving NSFW mode
	model         string // Current model name
	imageModel    string // Current image generation model (for NSFW mode)
	provider      string // Current provider (grok, openai, venice, etc.) - detected from endpoint
	skillsEnabled bool   // Whether skills/function calling is available
	version       string // Application version (e.g., "1.0.1")
	build         string // Build identifier (e.g., "bubbletea-tui")

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

	// Context tracking (NEW)
	contextTracker     *config.ContextTracker
	showContextWarning bool
	lastWarningLevel   string

	// Interactive selector
	selector       SelectorModel
	selectorActive bool
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
		// If selector is active, route all keys to it
		if m.selectorActive {
			var cmd tea.Cmd
			m.selector, cmd = m.selector.Update(msg)
			return m, cmd
		}

		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "ctrl+k":
			// Toggle skill call logs visibility
			m.chat = m.chat.ToggleSkillCalls()
			m.status = m.status.SetText("Skill calls toggled")
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

			// Update skills panel with current input for contextual help
			m.skills = m.skills.SetCurrentInput(m.input.Value())
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true

		// Calculate component heights - RPG menu layout
		headerHeight := 1
		inputHeight := 2
		skillsHeight := 12 // Increased for RPG-style menu with contextual help
		statusHeight := 1
		chatHeight := m.height - headerHeight - inputHeight - skillsHeight - statusHeight

		// Ensure minimum chat height
		if chatHeight < 5 {
			chatHeight = 5
			skillsHeight = m.height - headerHeight - inputHeight - statusHeight - chatHeight
			if skillsHeight < 6 {
				skillsHeight = 6 // Minimum for RPG menu
			}
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
			// Handle Phase 4 commands that require app state (contextTracker, currentSession)
			switch cmd.Name {
			case "stats":
				// Pass animation frame for flickering corruption effects
				argsWithFrame := append([]string{"--frame", fmt.Sprintf("%d", m.animFrame)}, cmd.Args...)
				result := commands.HandleStatsCommand(argsWithFrame, m.contextTracker)
				if result.ShouldRender {
					m.chat = m.chat.AddSystemMessage(result.Message)
				}
				return m, nil

			case "export":
				// Get pointer to current session for export
				var sessionPtr *config.Session
				if sess, ok := m.currentSession.(*config.Session); ok {
					sessionPtr = sess
				}
				result := commands.HandleExportCommand(cmd.Args, sessionPtr)
				if result.ShouldRender {
					m.chat = m.chat.AddSystemMessage(result.Message)
				}
				return m, nil

			case "context":
				result := commands.HandleContextCommand(cmd.Args, m.contextTracker)
				if result.ShouldRender {
					m.chat = m.chat.AddSystemMessage(result.Message)
				}
				return m, nil
			}

			// For other commands, use normal execution flow
			// Create context with current state (needed for model listing/validation)
			// Try to get config from LLMClient (available if it's the adapter from main.go)
			// If not available, commands will fall back to static model lists
			ctx := &commands.CommandContext{
				NSFWMode:      m.nsfwMode,
				Provider:      m.provider,
				CurrentModel:  m.model,
				APIKey:        "", // Will be populated if config accessible
				BaseURL:       "", // Will be populated if config accessible
				SkillsEnabled: m.skillsEnabled,
				Version:       m.version,
				Build:         m.build,
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

					// Detect provider from endpoint name
					// Provider detection will use endpoint name mapping
					m.provider = m.endpoint

					// Update skills availability and auto-select best model
					if caps, ok := providers.GetProvider(m.provider); ok {
						m.skillsEnabled = caps.SupportsFunctionCalling

						// AUTO-SELECT: Choose best tool-calling model for this provider
						if caps.PreferredToolModel != "" {
							m.model = caps.PreferredToolModel
							m.header = m.header.SetModel(m.model)
							LogInfo(fmt.Sprintf("Auto-selected model: %s (optimized for tool calling)", m.model))

							// Update LLM client model
							if switcher, ok := m.llmClient.(EndpointSwitcher); ok {
								if err := switcher.ChangeModel(m.model); err != nil {
									LogInfo(fmt.Sprintf("Error changing model: %v", err))
								}
							}
						} else if caps.DefaultModel != "" {
							m.model = caps.DefaultModel
							m.header = m.header.SetModel(m.model)
							LogInfo(fmt.Sprintf("Using default model: %s", m.model))

							// Update LLM client model
							if switcher, ok := m.llmClient.(EndpointSwitcher); ok {
								if err := switcher.ChangeModel(m.model); err != nil {
									LogInfo(fmt.Sprintf("Error changing model: %v", err))
								}
							}
						}

						LogInfo(fmt.Sprintf("Provider detected: %s, skills enabled: %v", m.provider, m.skillsEnabled))
					}

					m.header = m.header.SetEndpoint(m.endpoint)
					m.header = m.header.SetSkillsEnabled(m.skillsEnabled) // Update UI indicator
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

					// When NSFW mode is enabled, save current endpoint and switch to Venice
					if m.nsfwMode {
						// Save the current "safe" endpoint
						m.safeEndpoint = m.endpoint
						m.endpoint = "venice"
						m.header = m.header.SetEndpoint(m.endpoint)

						// Actually switch the LLM client to Venice
						if switcher, ok := m.llmClient.(EndpointSwitcher); ok {
							if err := switcher.SwitchEndpoint(m.endpoint); err != nil {
								m.status = m.status.SetText(fmt.Sprintf("Error switching to Venice: %v", err))
							}
						}
					} else {
						// When NSFW mode is disabled, restore the safe endpoint
						if m.safeEndpoint != "" {
							m.endpoint = m.safeEndpoint
						} else {
							// Fallback to default if no safe endpoint saved
							m.endpoint = "openai"
						}
						m.header = m.header.SetEndpoint(m.endpoint)

						// Actually switch the LLM client back
						if switcher, ok := m.llmClient.(EndpointSwitcher); ok {
							if err := switcher.SwitchEndpoint(m.endpoint); err != nil {
								m.status = m.status.SetText(fmt.Sprintf("Error switching endpoint: %v", err))
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

				if result.StateChange.MenuState != nil {
					m.skills = m.skills.SetMenuState(*result.StateChange.MenuState)
				}

				// Handle session actions
				if result.StateChange.SessionAction != nil {
					m = m.handleSessionAction(result.StateChange.SessionAction)
				}

				// Handle selector request
				if result.StateChange.ShowSelector != nil {
					// Convert commands.SelectorItem to tui.SelectorItem
					tuiItems := make([]SelectorItem, len(result.StateChange.ShowSelector.Items))
					for i, item := range result.StateChange.ShowSelector.Items {
						tuiItems[i] = SelectorItem{
							ID:          item.ID,
							DisplayName: item.DisplayName,
							Description: item.Description,
							Badge:       item.Badge,
						}
					}

					// Activate selector
					m.selector = NewSelectorModel(result.StateChange.ShowSelector.Title, tuiItems)
					m.selector = m.selector.SetHeight(m.height - 4)
					m.selector = m.selector.SetWidth(m.width)
					m.selectorActive = true
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

	case ShowSelectorMsg:
		// Activate the selector
		m.selector = NewSelectorModel(msg.Title, msg.Items)
		m.selector = m.selector.SetHeight(m.height - 4) // Leave room for borders/footer
		m.selector = m.selector.SetWidth(m.width)
		m.selectorActive = true

	case SelectorResultMsg:
		// Handle selector result
		m.selectorActive = false

		if msg.Cancelled {
			// User cancelled - show cancellation message
			m.chat = m.chat.AddSystemMessage("Selection cancelled")
			m.status = m.status.SetText("Selection cancelled")
		} else if msg.Selected != nil {
			// User selected an item - trigger model change
			modelName := msg.Selected.ID

			// Use the switcher interface to change model
			if switcher, ok := m.llmClient.(EndpointSwitcher); ok {
				if err := switcher.ChangeModel(modelName); err != nil {
					m.chat = m.chat.AddSystemMessage(fmt.Sprintf("❌ Failed to change model: %v", err))
					m.status = m.status.SetText(fmt.Sprintf("Error: %v", err))
				} else {
					m.model = modelName
					m.header = m.header.SetModel(modelName)
					m.chat = m.chat.AddSystemMessage(fmt.Sprintf("🤖 Model changed to: %s", modelName))
					m.status = m.status.SetText(fmt.Sprintf("Model changed to: %s", modelName))

					// Persist the change
					m.persistSession()
				}
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

	// If selector is active, show it full-screen
	if m.selectorActive {
		return m.selector.View()
	}

	// Build the layout vertically
	var sections []string

	// Header (fixed, 1 line)
	sections = append(sections, m.header.View())

	// Chat panel (flexible height)
	sections = append(sections, m.chat.View())

	// Input panel (fixed, 3 lines)
	sections = append(sections, m.input.View())

	// Skills panel (fixed, 5 lines) - update config before rendering
	// Calculate skills count and disabled reason
	skillsCount := len(m.skills.GetDefinitions())
	disabledReason := ""
	if !m.skillsEnabled {
		if m.nsfwMode {
			disabledReason = "NSFW Mode - Venice doesn't support tools"
		} else {
			disabledReason = "Current model doesn't support function calling"
		}
	}

	m.skills = m.skills.SetConfig(m.endpoint, m.model, m.skillsEnabled, m.nsfwMode, skillsCount, disabledReason)
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
// Uses interface{} for return types to avoid circular dependencies.
type SessionManager interface {
	NewSession() interface{}
	Save(session interface{}) error
	Load(id string) (interface{}, error)
	List() ([]interface{}, error)
	MergeSessions(session1, session2 interface{}) interface{}
}

// Session interface for session data (avoid circular import).
// Uses interface{} for complex types to avoid circular dependencies.
type Session interface {
	SetEndpoint(endpoint string)
	GetEndpoint() string
	SetModel(model string)
	GetModel() string
	SetNSFWMode(enabled bool)
	GetNSFWMode() bool
	ClearMessages()
	GetMessagesRaw() interface{}     // Returns []SessionMessage
	SetMessagesRaw(msgs interface{}) // Accepts []SessionMessage
	SummarizeRaw() interface{}       // Returns SessionSummary
}

// SessionMessage represents a message stored in session (matches config.SessionMessage).
type SessionMessage struct {
	Role      string
	Content   string
	Timestamp time.Time
}

// SessionSummary represents session metadata (matches config.SessionSummary).
// Duplicated here to avoid circular import with config package.
type SessionSummary struct {
	ID           string
	Name         string
	MessageCount int
	CreatedAt    time.Time
	UpdatedAt    time.Time
	FirstMessage string
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

			// Initialize context tracker with session and model
			// Convert Session interface to *config.Session for ContextTracker
			if configSession, ok := session.(*config.Session); ok {
				m.contextTracker = config.NewContextTracker(configSession, model)
				// Update header with initial context usage
				if m.contextTracker.MaxTokens > 0 {
					m.header = m.header.SetContextUsage(m.contextTracker.CurrentTokens, m.contextTracker.MaxTokens)
				}
			}
		}
		m.nsfwMode = session.GetNSFWMode()
		m.header = m.header.SetNSFWMode(m.nsfwMode)
	}

	return m
}

// SetVersion sets the application version and build information.
func (m AppModel) SetVersion(version, build string) AppModel {
	m.version = version
	m.build = build
	return m
}

// WithMessages restores chat history from session messages.
func (m AppModel) WithMessages(messages []ChatMessage) AppModel {
	for _, msg := range messages {
		switch msg.Role {
		case "user":
			m.chat = m.chat.AddUserMessage(msg.Content)
		case "assistant":
			m.chat = m.chat.AddAssistantMessage(msg.Content)
		case "tool":
			m.chat = m.chat.AddToolResult(msg.ToolCallID, msg.Name, msg.Content)
		}
	}
	return m
}

// WithEndpoint restores the endpoint/provider from a loaded session.
func (m AppModel) WithEndpoint(endpoint string) AppModel {
	if endpoint != "" {
		m.endpoint = endpoint
		m.provider = endpoint // Provider matches endpoint name
		m.header = m.header.SetEndpoint(endpoint)
		LogInfo(fmt.Sprintf("✓ Restored endpoint from session: %s", endpoint))
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

	// Convert TUI ChatMessages to config SessionMessages
	chatMsgs := m.chat.GetMessages()
	sessionMsgs := make([]SessionMessage, 0, len(chatMsgs))
	for _, msg := range chatMsgs {
		// Skip system messages (UI-only, not part of LLM conversation)
		if msg.Role == "system" {
			continue
		}
		sessionMsgs = append(sessionMsgs, SessionMessage{
			Role:      msg.Role,
			Content:   msg.Content,
			Timestamp: msg.Timestamp,
		})
	}
	m.currentSession.SetMessagesRaw(sessionMsgs)

	// Save asynchronously (ignore errors for now)
	go func() {
		_ = m.sessionManager.Save(m.currentSession)
	}()
}

// handleSessionAction handles session management actions.
func (m AppModel) handleSessionAction(action *commands.SessionAction) AppModel {
	if m.sessionManager == nil {
		m.chat = m.chat.AddSystemMessage("❌ Session manager not available")
		return m
	}

	switch action.Action {
	case "new":
		// Save current session first
		m.persistSession()

		// Create new session
		newSession := m.sessionManager.NewSession()
		if s, ok := newSession.(Session); ok {
			// TODO: Set name through metadata if action.Name is provided
			// config.Session doesn't currently have a SetName method
			m.currentSession = s

			// Clear chat
			m.chat = m.chat.Clear()

			// Show success with short ID
			if summary := s.SummarizeRaw(); summary != nil {
				m.chat = m.chat.AddSystemMessage("📝 New session created")
			}
		}

	case "resume":
		// Save current session first
		m.persistSession()

		// Load requested session
		if loaded, err := m.sessionManager.Load(action.SessionID); err == nil {
			if s, ok := loaded.(Session); ok {
				m.currentSession = s

				// Clear current chat
				m.chat = m.chat.Clear()

				// Restore messages
				if messagesRaw := s.GetMessagesRaw(); messagesRaw != nil {
					if sessionMsgs, ok := messagesRaw.([]SessionMessage); ok {
						for _, msg := range sessionMsgs {
							switch msg.Role {
							case "user":
								m.chat = m.chat.AddUserMessage(msg.Content)
							case "assistant":
								m.chat = m.chat.AddAssistantMessage(msg.Content)
							}
						}
					}
				}

				// Restore state
				if endpoint := s.GetEndpoint(); endpoint != "" {
					m.endpoint = endpoint
					m.header = m.header.SetEndpoint(m.endpoint)
				}
				m.nsfwMode = s.GetNSFWMode()
				m.header = m.header.SetNSFWMode(m.nsfwMode)

				msgCount := 0
				if msgs := s.GetMessagesRaw(); msgs != nil {
					if sm, ok := msgs.([]SessionMessage); ok {
						msgCount = len(sm)
					}
				}
				m.chat = m.chat.AddSystemMessage(
					fmt.Sprintf("📂 Resumed session (%d messages)", msgCount))
			}
		} else {
			m.chat = m.chat.AddSystemMessage(
				fmt.Sprintf("❌ Failed to load session: %v", err))
		}

	case "list":
		if sessions, err := m.sessionManager.List(); err == nil {
			if len(sessions) == 0 {
				m.chat = m.chat.AddSystemMessage("No saved sessions")
			} else {
				var sb strings.Builder
				sb.WriteString(fmt.Sprintf("\n📋 Saved Sessions (%d):\n\n", len(sessions)))
				for _, sessionRaw := range sessions {
					if s, ok := sessionRaw.(Session); ok {
						if summaryRaw := s.SummarizeRaw(); summaryRaw != nil {
							// Type assert the summary to our local struct
							if summary, ok := summaryRaw.(SessionSummary); ok {
								// Format session entry
								preview := summary.FirstMessage
								if preview == "" {
									preview = "(empty session)"
								} else if len(preview) > 40 {
									preview = preview[:37] + "..."
								}

								// Show ID (last 8 chars), message count, and preview
								shortID := summary.ID
								if len(shortID) > 8 {
									shortID = shortID[len(shortID)-8:]
								}

								sb.WriteString(fmt.Sprintf("• [%s] %d msgs - %s\n",
									shortID, summary.MessageCount, preview))
							}
						}
					}
				}
				sb.WriteString("\nUse /session resume <id> to load a session")
				m.chat = m.chat.AddSystemMessage(sb.String())
			}
		} else {
			m.chat = m.chat.AddSystemMessage(
				fmt.Sprintf("❌ Failed to list sessions: %v", err))
		}

	case "clear":
		// Create new session automatically
		newSession := m.sessionManager.NewSession()
		if s, ok := newSession.(Session); ok {
			m.currentSession = s
		}
		m.chat = m.chat.AddSystemMessage("🗑️  Session cleared, new session started")

	case "merge":
		if toMerge, err := m.sessionManager.Load(action.SessionID); err == nil {
			merged := m.sessionManager.MergeSessions(m.currentSession, toMerge)
			if s, ok := merged.(Session); ok {
				m.currentSession = s

				// Clear and reload with merged messages
				m.chat = m.chat.Clear()
				if messagesRaw := s.GetMessagesRaw(); messagesRaw != nil {
					if sessionMsgs, ok := messagesRaw.([]SessionMessage); ok {
						for _, msg := range sessionMsgs {
							switch msg.Role {
							case "user":
								m.chat = m.chat.AddUserMessage(msg.Content)
							case "assistant":
								m.chat = m.chat.AddAssistantMessage(msg.Content)
							}
						}

						m.chat = m.chat.AddSystemMessage(
							fmt.Sprintf("🔀 Merged sessions (%d total messages)", len(sessionMsgs)))
					}
				}

				// Save merged session
				m.persistSession()
			}
		} else {
			m.chat = m.chat.AddSystemMessage(
				fmt.Sprintf("❌ Failed to merge session: %v", err))
		}

	case "info":
		if m.currentSession != nil {
			msgCount := 0
			if msgs := m.currentSession.GetMessagesRaw(); msgs != nil {
				if sm, ok := msgs.([]SessionMessage); ok {
					msgCount = len(sm)
				}
			}

			var sb strings.Builder
			sb.WriteString("\n📊 Current Session Info:\n\n")
			sb.WriteString(fmt.Sprintf("• Messages: %d\n", msgCount))
			sb.WriteString(fmt.Sprintf("• Model: %s\n", m.model))
			sb.WriteString(fmt.Sprintf("• Endpoint: %s\n", m.endpoint))
			if m.nsfwMode {
				sb.WriteString("• Mode: NSFW\n")
			}

			m.chat = m.chat.AddSystemMessage(sb.String())
		}
	}

	return m
}

// --- Header Model ---

// HeaderModel represents the header bar.
type HeaderModel struct {
	width           int
	nsfwMode        bool
	endpoint        string
	model           string
	imageModel      string // Image generation model (NSFW mode)
	autoRouted      bool   // Whether the last message was auto-routed
	skillsEnabled   bool   // Whether skills/function calling is available
	contextIndicator ContextIndicator // Token usage display
	showContext     bool   // Whether to show context usage
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

// SetSkillsEnabled sets whether skills/function calling is available.
func (m HeaderModel) SetSkillsEnabled(enabled bool) HeaderModel {
	m.skillsEnabled = enabled
	return m
}

// SetAutoRouted sets whether auto-routing occurred.
func (m HeaderModel) SetAutoRouted(routed bool) HeaderModel {
	m.autoRouted = routed
	return m
}

// SetContextUsage updates the context usage display.
func (m HeaderModel) SetContextUsage(current, max int) HeaderModel {
	m.contextIndicator = m.contextIndicator.SetUsage(current, max)
	m.showContext = true
	return m
}

// SetShowContext controls whether context usage is displayed.
func (m HeaderModel) SetShowContext(show bool) HeaderModel {
	m.showContext = show
	return m
}

// GetContextWarningLevel returns the current context warning level.
func (m HeaderModel) GetContextWarningLevel() string {
	return m.contextIndicator.GetWarningLevel()
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
			endpointInfo += " • " + ModelStyle.Render("img:"+m.imageModel)
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
		// Add capability indicator
		modelDisplay := m.model
		if m.skillsEnabled {
			modelDisplay += " ✓" // Checkmark for skills enabled
		} else {
			modelDisplay += " ⚠" // Warning for no skills
		}
		endpointInfo += ModelStyle.Render(modelDisplay)
	}

	// Add context usage indicator if available
	var contextInfo string
	if m.showContext {
		contextInfo = m.contextIndicator.ViewCompact()
	}

	info := HeaderInfoStyle.Render("Press Ctrl+C to exit")
	if endpointInfo != "" {
		info = endpointInfo + " • " + info
	}
	if contextInfo != "" {
		info = info + " • " + contextInfo
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
	width            int
	text             string
	streaming        bool
	frame            int
	warningMessage   string // Context warning message
	warningLevel     string // "warn", "caution", "critical"
	showWarning      bool   // Whether to show warning
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

// ShowContextWarning displays a context warning message.
func (m StatusModel) ShowContextWarning(level string, message string) StatusModel {
	m.warningLevel = level
	m.warningMessage = message
	m.showWarning = true
	return m
}

// ClearContextWarning clears the context warning.
func (m StatusModel) ClearContextWarning() StatusModel {
	m.showWarning = false
	m.warningMessage = ""
	m.warningLevel = ""
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

	// Priority: warnings > streaming > normal text
	if m.showWarning {
		// Show context warning with appropriate color
		warningStyle := m.getWarningStyle()
		status = warningStyle.Render(m.warningMessage)
	} else if m.streaming {
		// Animated spinner
		frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
		spinner := StatusStreamingStyle.Render(frames[m.frame%len(frames)])
		status = spinner + " " + StatusStreamingStyle.Render("Streaming...")
	} else {
		status = StatusActiveStyle.Render("●") + " " + m.text
	}

	return StatusBarStyle.Width(m.width).Render(status)
}

// getWarningStyle returns the appropriate style for the warning level.
func (m StatusModel) getWarningStyle() lipgloss.Style {
	switch m.warningLevel {
	case "critical":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true) // Bright red, bold
	case "caution":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("208")).Bold(true) // Orange, bold
	case "warn":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("226")) // Yellow
	default:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("82")) // Green
	}
}

// --- Helper functions ---

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
