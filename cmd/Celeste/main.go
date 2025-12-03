// Celeste CLI - Interactive AI Assistant with Bubble Tea TUI
// This file provides the new main entry point using Bubble Tea.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/whykusanagi/celesteCLI/cmd/Celeste/config"
	"github.com/whykusanagi/celesteCLI/cmd/Celeste/llm"
	"github.com/whykusanagi/celesteCLI/cmd/Celeste/prompts"
	"github.com/whykusanagi/celesteCLI/cmd/Celeste/skills"
	"github.com/whykusanagi/celesteCLI/cmd/Celeste/tui"
)

// Version information
const (
	Version = "3.0.0"
	Build   = "bubbletea-tui"
)

// Global config name (set by -config flag)
var configName string

func main() {
	// Check for -config flag before command
	args := os.Args[1:]
	for i := 0; i < len(args); i++ {
		if args[i] == "-config" && i+1 < len(args) {
			configName = args[i+1]
			// Remove -config and its value from args
			args = append(args[:i], args[i+2:]...)
			break
		} else if strings.HasPrefix(args[i], "-config=") {
			configName = strings.TrimPrefix(args[i], "-config=")
			args = append(args[:i], args[i+1:]...)
			break
		}
	}

	// Parse command line
	if len(args) < 1 {
		printUsage()
		os.Exit(0)
	}

	command := args[0]
	cmdArgs := args[1:]

	switch command {
	case "chat":
		runChatTUI()
	case "config":
		runConfigCommand(cmdArgs)
	case "message", "msg":
		if len(cmdArgs) < 1 {
			fmt.Fprintln(os.Stderr, "Usage: celeste message <text>")
			os.Exit(1)
		}
		runSingleMessage(strings.Join(cmdArgs, " "))
	case "skills":
		runSkillsCommand(cmdArgs)
	case "session", "sessions":
		runSessionCommand(cmdArgs)
	case "help", "-h", "--help":
		printUsage()
	case "version", "-v", "--version":
		fmt.Printf("Celeste CLI %s (%s)\n", Version, Build)
	default:
		// Treat unknown command as a message
		runSingleMessage(strings.Join(args, " "))
	}
}

// printUsage prints the CLI usage information.
func printUsage() {
	fmt.Println(`
✨ Celeste CLI - Interactive AI Assistant

Usage:
  celeste [-config <name>] <command> [arguments]

Global Flags:
  -config <name>          Use named config (loads ~/.celeste/config.<name>.json)

Commands:
  chat                    Launch interactive TUI mode
  message <text>          Send a single message and exit
  config                  View/modify configuration
  skills                  List and manage skills
  session                 Manage conversation sessions
  help                    Show this help message
  version                 Show version information

Interactive Commands (in chat mode):
  help                    Show available commands
  clear                   Clear chat history
  config                  Show current configuration
  tools, debug            Show available skills
  exit, quit, q           Exit the application

Keyboard Shortcuts:
  Ctrl+C                  Exit immediately
  PgUp/PgDown            Scroll chat history
  Shift+↑/↓              Scroll chat history
  ↑/↓                    Navigate input history

Configuration:
  celeste config --show                  Show current config
  celeste config --list                  List all config profiles
  celeste config --init <name>           Create a new config profile
  celeste config --set-key <key>         Set API key
  celeste config --set-url <url>         Set API URL
  celeste config --set-model <model>     Set model
  celeste config --skip-persona <bool>   Skip persona prompt injection

Skills:
  celeste skills --list                  List available skills
  celeste skills --init                  Create default skill files

Sessions:
  celeste session --list                 List saved sessions
  celeste session --load <id>            Load a session
  celeste session --clear                Clear all sessions

Environment Variables:
  CELESTE_API_KEY         API key (overrides config)
  CELESTE_API_ENDPOINT    API endpoint (overrides config)
  VENICE_API_KEY          Venice.ai API key for NSFW mode
  TAROT_AUTH_TOKEN        Tarot function auth token

Examples:
  celeste chat                           Start with default config
  celeste -config openai chat            Start with OpenAI config
  celeste -config grok chat              Start with Grok/xAI config
  celeste config --list                  List available configs
  celeste config --init openai           Create OpenAI config template
`)
}

// runChatTUI launches the interactive Bubble Tea TUI.
func runChatTUI() {
	// Load configuration (named or default)
	cfg, err := config.LoadNamed(configName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	// Show which config is being used
	if configName != "" {
		fmt.Fprintf(os.Stderr, "Using config: %s\n", configName)
	}

	// Validate API key
	if cfg.APIKey == "" {
		fmt.Fprintln(os.Stderr, "No API key configured.")
		if configName != "" {
			fmt.Fprintf(os.Stderr, "Edit %s or set CELESTE_API_KEY\n", config.NamedConfigPath(configName))
		} else {
			fmt.Fprintln(os.Stderr, "Set CELESTE_API_KEY environment variable or run: celeste config --set-key <key>")
		}
		os.Exit(1)
	}

	// Initialize skill registry
	registry := skills.NewRegistry()
	if err := registry.LoadSkills(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to load skills: %v\n", err)
	}

	// Register built-in skills
	configLoader := config.NewConfigLoader(cfg)
	skills.RegisterBuiltinSkills(registry, configLoader)

	// Initialize LLM client
	llmConfig := &llm.Config{
		APIKey:            cfg.APIKey,
		BaseURL:           cfg.BaseURL,
		Model:             cfg.Model,
		Timeout:           cfg.GetTimeout(),
		SkipPersonaPrompt: cfg.SkipPersonaPrompt,
		SimulateTyping:    cfg.SimulateTyping,
		TypingSpeed:       cfg.TypingSpeed,
	}
	client := llm.NewClient(llmConfig, registry)

	// Set system prompt if not skipping
	if !cfg.SkipPersonaPrompt {
		client.SetSystemPrompt(prompts.GetSystemPrompt(false))
	}

	// Create TUI client adapter
	tuiClient := &TUIClientAdapter{
		client:   client,
		registry: registry,
	}

	// Initialize logging for skill calls
	if err := tui.InitLogging(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to init logging: %v\n", err)
	}
	defer tui.CloseLogging()

	// Create and run TUI
	app := tui.NewApp(tuiClient)
	p := tea.NewProgram(app, tea.WithAltScreen(), tea.WithMouseCellMotion())

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running TUI: %v\n", err)
		os.Exit(1)
	}

	// Print log path on exit
	if logPath := tui.GetLogPath(); logPath != "" {
		fmt.Printf("\nSkill call log: %s\n", logPath)
	}
}

// TUIClientAdapter adapts the LLM client for the TUI.
type TUIClientAdapter struct {
	client   *llm.Client
	registry *skills.Registry
}

// SendMessage implements tui.LLMClient.
func (a *TUIClientAdapter) SendMessage(messages []tui.ChatMessage, tools []tui.SkillDefinition) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		// Log the request
		tui.LogLLMRequest(len(messages), len(tools))

		var fullContent string
		var toolCalls []llm.ToolCallResult

		err := a.client.SendMessageStream(ctx, messages, tools, func(chunk llm.StreamChunk) {
			fullContent += chunk.Content
			if chunk.IsFinal {
				toolCalls = chunk.ToolCalls
			}
		})

		if err != nil {
			tui.LogInfo(fmt.Sprintf("LLM error: %v", err))
			return tui.StreamErrorMsg{Err: err}
		}

		// Log the response
		tui.LogLLMResponse(len(fullContent), len(toolCalls) > 0)

		// Handle tool calls
		if len(toolCalls) > 0 {
			tc := toolCalls[0]
			tui.LogInfo(fmt.Sprintf("LLM requested tool call: %s (ID: %s)", tc.Name, tc.ID))
			
			// Convert all tool calls to ToolCallInfo
			toolCallInfos := make([]tui.ToolCallInfo, len(toolCalls))
			for i, t := range toolCalls {
				toolCallInfos[i] = tui.ToolCallInfo{
					ID:        t.ID,
					Name:      t.Name,
					Arguments: t.Arguments,
				}
			}
			
			return tui.SkillCallMsg{
				Call: tui.FunctionCall{
					Name:      tc.Name,
					Arguments: parseArgs(tc.Arguments),
					Status:    "executing",
					Timestamp: time.Now(),
				},
				ToolCallID:      tc.ID, // Store tool call ID for sending result back
				AssistantContent: fullContent, // Store any assistant content before tool call
				ToolCalls:        toolCallInfos,
			}
		}

		return tui.StreamDoneMsg{
			FullContent:  fullContent,
			FinishReason: "stop",
		}
	}
}

// GetSkills implements tui.LLMClient.
func (a *TUIClientAdapter) GetSkills() []tui.SkillDefinition {
	return a.client.GetSkills()
}

// ExecuteSkill implements tui.LLMClient.
func (a *TUIClientAdapter) ExecuteSkill(name string, args map[string]any, toolCallID string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		startTime := time.Now()
		tui.LogInfo(fmt.Sprintf("Executing skill '%s' with timeout: 30s", name))

		// Convert args to JSON
		argsJSON, err := json.Marshal(args)
		if err != nil {
			tui.LogInfo(fmt.Sprintf("Failed to marshal args for '%s': %v", name, err))
			return tui.SkillResultMsg{
				Name:       name,
				Result:     "",
				Err:        fmt.Errorf("failed to marshal arguments: %w", err),
				ToolCallID: toolCallID,
			}
		}

		// Execute the skill
		result, err := a.client.ExecuteSkill(ctx, name, string(argsJSON))
		
		elapsed := time.Since(startTime)
		if err != nil {
			tui.LogInfo(fmt.Sprintf("Skill '%s' failed after %v: %v", name, elapsed, err))
			return tui.SkillResultMsg{
				Name:       name,
				Result:     "",
				Err:        err,
				ToolCallID: toolCallID,
			}
		}

		// Format result as string
		var resultStr string
		if result.Success {
			switch v := result.Result.(type) {
			case string:
				resultStr = v
			case map[string]interface{}:
				b, _ := json.Marshal(v)
				resultStr = string(b)
			default:
				b, _ := json.Marshal(result.Result)
				resultStr = string(b)
			}
			tui.LogInfo(fmt.Sprintf("Skill '%s' completed successfully in %v", name, elapsed))
		} else {
			resultStr = fmt.Sprintf("Error: %s", result.Error)
			tui.LogInfo(fmt.Sprintf("Skill '%s' returned error after %v: %s", name, elapsed, result.Error))
		}

		return tui.SkillResultMsg{
			Name:       name,
			Result:     resultStr,
			Err:        nil,
			ToolCallID: toolCallID,
		}
	}
}

func parseArgs(argsJSON string) map[string]any {
	var args map[string]any
	json.Unmarshal([]byte(argsJSON), &args)
	if args == nil {
		args = make(map[string]any)
	}
	return args
}

// runConfigCommand handles configuration commands.
func runConfigCommand(args []string) {
	fs := flag.NewFlagSet("config", flag.ExitOnError)
	showConfig := fs.Bool("show", false, "Show current configuration")
	listConfigs := fs.Bool("list", false, "List all config profiles")
	initConfig := fs.String("init", "", "Create a new config profile (openai, grok, elevenlabs, venice)")
	setKey := fs.String("set-key", "", "Set API key")
	setURL := fs.String("set-url", "", "Set API URL")
	setModel := fs.String("set-model", "", "Set model")
	skipPersona := fs.String("skip-persona", "", "Skip persona prompt (true/false)")
	simulateTyping := fs.String("simulate-typing", "", "Simulate typing (true/false)")
	typingSpeed := fs.Int("typing-speed", 0, "Typing speed (chars/sec)")
	
	// Skill configuration flags
	setTarotToken := fs.String("set-tarot-token", "", "Set tarot auth token (saved to skills.json)")
	setVeniceKey := fs.String("set-venice-key", "", "Set Venice.ai API key (saved to skills.json)")
	setTarotURL := fs.String("set-tarot-url", "", "Set tarot function URL (saved to skills.json)")
	setWeatherZip := fs.String("set-weather-zip", "", "Set default weather zip code (saved to skills.json)")
	
	fs.Parse(args)

	// Handle --list
	if *listConfigs {
		configs, err := config.ListConfigs()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error listing configs: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Available config profiles:")
		for _, c := range configs {
			path := config.NamedConfigPath(c)
			if c == "default" {
				path = config.NamedConfigPath("")
			}
			fmt.Printf("  • %s (%s)\n", c, path)
		}
		fmt.Println("\nUsage: celeste -config <name> chat")
		return
	}

	// Handle --init
	if *initConfig != "" {
		if err := createConfigTemplate(*initConfig); err != nil {
			fmt.Fprintf(os.Stderr, "Error creating config: %v\n", err)
			os.Exit(1)
		}
		return
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	changed := false

	if *setKey != "" {
		cfg.APIKey = *setKey
		changed = true
		fmt.Println("API key updated")
	}
	if *setURL != "" {
		cfg.BaseURL = *setURL
		changed = true
		fmt.Printf("API URL set to: %s\n", *setURL)
	}
	if *setModel != "" {
		cfg.Model = *setModel
		changed = true
		fmt.Printf("Model set to: %s\n", *setModel)
	}
	if *skipPersona != "" {
		cfg.SkipPersonaPrompt = strings.ToLower(*skipPersona) == "true"
		changed = true
		fmt.Printf("Skip persona prompt: %v\n", cfg.SkipPersonaPrompt)
	}
	if *simulateTyping != "" {
		cfg.SimulateTyping = strings.ToLower(*simulateTyping) == "true"
		changed = true
		fmt.Printf("Simulate typing: %v\n", cfg.SimulateTyping)
	}
	if *typingSpeed > 0 {
		cfg.TypingSpeed = *typingSpeed
		changed = true
		fmt.Printf("Typing speed: %d chars/sec\n", cfg.TypingSpeed)
	}

	// Handle skill configuration
	skillsChanged := false
	if *setTarotToken != "" {
		cfg.TarotAuthToken = *setTarotToken
		skillsChanged = true
		fmt.Println("Tarot auth token updated (saved to skills.json)")
	}
	if *setVeniceKey != "" {
		cfg.VeniceAPIKey = *setVeniceKey
		skillsChanged = true
		fmt.Println("Venice.ai API key updated (saved to skills.json)")
	}
	if *setTarotURL != "" {
		cfg.TarotFunctionURL = *setTarotURL
		skillsChanged = true
		fmt.Printf("Tarot function URL set to: %s (saved to skills.json)\n", *setTarotURL)
	}
	if *setWeatherZip != "" {
		// Validate zip code format
		zip := *setWeatherZip
		if len(zip) != 5 {
			fmt.Fprintf(os.Stderr, "Error: zip code must be 5 digits\n")
			os.Exit(1)
		}
		for _, c := range zip {
			if c < '0' || c > '9' {
				fmt.Fprintf(os.Stderr, "Error: zip code must contain only digits\n")
				os.Exit(1)
			}
		}
		cfg.WeatherDefaultZipCode = zip
		skillsChanged = true
		fmt.Printf("Default weather zip code set to: %s (saved to skills.json)\n", zip)
	}

	if changed {
		if err := config.Save(cfg); err != nil {
			fmt.Fprintf(os.Stderr, "Error saving config: %v\n", err)
			os.Exit(1)
		}
		if err := config.SaveSecrets(cfg); err != nil {
			fmt.Fprintf(os.Stderr, "Error saving secrets: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Configuration saved")
	}

	if skillsChanged {
		if err := config.SaveSkillsConfig(cfg); err != nil {
			fmt.Fprintf(os.Stderr, "Error saving skills config: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Skills configuration saved to skills.json")
	}

	if *showConfig || !changed {
		fmt.Printf("\nCurrent Configuration:\n")
		fmt.Printf("  API URL:           %s\n", cfg.BaseURL)
		fmt.Printf("  Model:             %s\n", cfg.Model)
		fmt.Printf("  API Key:           %s\n", maskKey(cfg.APIKey))
		fmt.Printf("  Skip Persona:      %v\n", cfg.SkipPersonaPrompt)
		fmt.Printf("  Simulate Typing:   %v\n", cfg.SimulateTyping)
		fmt.Printf("  Typing Speed:      %d chars/sec\n", cfg.TypingSpeed)
		fmt.Printf("  Venice API Key:    %s\n", maskKey(cfg.VeniceAPIKey))
		fmt.Printf("  Tarot Configured:  %v\n", cfg.TarotAuthToken != "")
		fmt.Printf("  Twitter Configured:%v\n", cfg.TwitterBearerToken != "")
		if cfg.WeatherDefaultZipCode != "" {
			fmt.Printf("  Weather Zip Code:  %s\n", cfg.WeatherDefaultZipCode)
		} else {
			fmt.Printf("  Weather Zip Code:  (not set)\n")
		}
	}
}

// createConfigTemplate creates a config file from a template.
func createConfigTemplate(name string) error {
	templates := map[string]*config.Config{
		"openai": {
			BaseURL:           "https://api.openai.com/v1",
			Model:             "gpt-4o-mini",
			Timeout:           60,
			SkipPersonaPrompt: false, // OpenAI needs persona injection
			SimulateTyping:    true,
			TypingSpeed:       25,
		},
		"grok": {
			BaseURL:           "https://api.x.ai/v1",
			Model:             "grok-4-latest",
			Timeout:           60,
			SkipPersonaPrompt: false, // Grok needs persona injection
			SimulateTyping:    true,
			TypingSpeed:       25,
		},
		"elevenlabs": {
			BaseURL:           "https://api.elevenlabs.io/v1",
			Model:             "eleven_multilingual_v2",
			Timeout:           60,
			SkipPersonaPrompt: false,
			SimulateTyping:    true,
			TypingSpeed:       25,
		},
		"venice": {
			BaseURL:           "https://api.venice.ai/api/v1",
			Model:             "venice-uncensored",
			Timeout:           60,
			SkipPersonaPrompt: false, // Venice needs persona injection
			SimulateTyping:    true,
			TypingSpeed:       25,
		},
		"digitalocean": {
			BaseURL:           "https://your-agent.ondigitalocean.app/api/v1",
			Model:             "gpt-4o-mini",
			Timeout:           60,
			SkipPersonaPrompt: true, // DO agents have built-in persona
			SimulateTyping:    true,
			TypingSpeed:       25,
		},
	}

	tmpl, ok := templates[strings.ToLower(name)]
	if !ok {
		return fmt.Errorf("unknown config template '%s'. Available: openai, grok, elevenlabs, venice, digitalocean", name)
	}

	configPath := config.NamedConfigPath(name)
	
	// Check if file already exists
	if _, err := os.Stat(configPath); err == nil {
		return fmt.Errorf("config '%s' already exists at %s", name, configPath)
	}

	// Write config
	data, err := json.MarshalIndent(tmpl, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return err
	}

	fmt.Printf("Created config '%s' at %s\n", name, configPath)
	fmt.Printf("\nEdit the file to add your API key, then run:\n")
	fmt.Printf("  celeste -config %s chat\n", name)
	return nil
}

func maskKey(key string) string {
	if key == "" {
		return "(not set)"
	}
	if len(key) <= 8 {
		return "****"
	}
	return key[:4] + "..." + key[len(key)-4:]
}

// runSkillsCommand handles skill-related commands.
func runSkillsCommand(args []string) {
	fs := flag.NewFlagSet("skills", flag.ExitOnError)
	list := fs.Bool("list", false, "List available skills")
	init := fs.Bool("init", false, "Create default skill files")
	fs.Parse(args)

	if *init {
		if err := skills.CreateDefaultSkillFiles(); err != nil {
			fmt.Fprintf(os.Stderr, "Error creating skill files: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Default skill files created in ~/.celeste/skills/")
		return
	}

	registry := skills.NewRegistry()
	registry.LoadSkills()

	// Register built-in skills (for display)
	cfg, _ := config.Load()
	configLoader := config.NewConfigLoader(cfg)
	skills.RegisterBuiltinSkills(registry, configLoader)

	if *list || len(args) == 0 {
		allSkills := registry.GetAllSkills()
		fmt.Printf("\nAvailable Skills (%d):\n", len(allSkills))
		for _, skill := range allSkills {
			fmt.Printf("\n  %s\n", skill.Name)
			fmt.Printf("    %s\n", skill.Description)
		}
		fmt.Println()
	}
}

// runSessionCommand handles session-related commands.
func runSessionCommand(args []string) {
	fs := flag.NewFlagSet("session", flag.ExitOnError)
	list := fs.Bool("list", false, "List saved sessions")
	load := fs.String("load", "", "Load a session by ID")
	clear := fs.Bool("clear", false, "Clear all sessions")
	fs.Parse(args)

	manager := config.NewSessionManager()

	if *clear {
		if err := manager.Clear(); err != nil {
			fmt.Fprintf(os.Stderr, "Error clearing sessions: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("All sessions cleared")
		return
	}

	if *load != "" {
		session, err := manager.Load(*load)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading session: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Loaded session: %s (%d messages)\n", session.ID, len(session.Messages))
		// In full implementation, this would resume the session in TUI
		return
	}

	if *list || len(args) == 0 {
		sessions, err := manager.List()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error listing sessions: %v\n", err)
			os.Exit(1)
		}

		if len(sessions) == 0 {
			fmt.Println("No saved sessions")
			return
		}

		fmt.Printf("\nSaved Sessions (%d):\n", len(sessions))
		for _, s := range sessions {
			summary := s.Summarize()
			fmt.Printf("\n  ID: %s\n", summary.ID)
			fmt.Printf("    Messages: %d\n", summary.MessageCount)
			fmt.Printf("    Created:  %s\n", summary.CreatedAt.Format("2006-01-02 15:04"))
			fmt.Printf("    Updated:  %s\n", summary.UpdatedAt.Format("2006-01-02 15:04"))
			if summary.FirstMessage != "" {
				fmt.Printf("    Preview:  %s\n", summary.FirstMessage)
			}
		}
		fmt.Println()
	}
}

// runSingleMessage sends a single message and prints the response.
func runSingleMessage(message string) {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	if cfg.APIKey == "" {
		fmt.Fprintln(os.Stderr, "No API key configured.")
		os.Exit(1)
	}

	// Initialize LLM client
	llmConfig := &llm.Config{
		APIKey:            cfg.APIKey,
		BaseURL:           cfg.BaseURL,
		Model:             cfg.Model,
		Timeout:           cfg.GetTimeout(),
		SkipPersonaPrompt: cfg.SkipPersonaPrompt,
	}
	client := llm.NewClient(llmConfig, nil)

	if !cfg.SkipPersonaPrompt {
		client.SetSystemPrompt(prompts.GetSystemPrompt(false))
	}

	// Send message
	ctx, cancel := context.WithTimeout(context.Background(), cfg.GetTimeout())
	defer cancel()

	messages := []tui.ChatMessage{{
		Role:      "user",
		Content:   message,
		Timestamp: time.Now(),
	}}

	result, err := client.SendMessageSync(ctx, messages, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(result.Content)
}

