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

func main() {
	// Parse command line
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(0)
	}

	command := os.Args[1]

	switch command {
	case "chat":
		runChatTUI()
	case "config":
		runConfigCommand(os.Args[2:])
	case "message", "msg":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "Usage: celeste message <text>")
			os.Exit(1)
		}
		runSingleMessage(strings.Join(os.Args[2:], " "))
	case "skills":
		runSkillsCommand(os.Args[2:])
	case "session", "sessions":
		runSessionCommand(os.Args[2:])
	case "help", "-h", "--help":
		printUsage()
	case "version", "-v", "--version":
		fmt.Printf("Celeste CLI %s (%s)\n", Version, Build)
	default:
		// Treat unknown command as a message
		runSingleMessage(strings.Join(os.Args[1:], " "))
	}
}

// printUsage prints the CLI usage information.
func printUsage() {
	fmt.Println(`
✨ Celeste CLI - Interactive AI Assistant

Usage:
  celeste <command> [arguments]

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
  exit, quit, q           Exit the application

Keyboard Shortcuts:
  Ctrl+C                  Exit immediately
  PgUp/PgDown            Scroll chat history
  Shift+↑/↓              Scroll chat history
  ↑/↓                    Navigate input history

Configuration:
  celeste config --show                  Show current config
  celeste config --set-key <key>         Set API key
  celeste config --set-url <url>         Set API URL
  celeste config --set-model <model>     Set model
  celeste config --skip-persona <bool>   Skip persona prompt injection
  celeste config --simulate-typing <bool> Enable/disable typing simulation

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
  celeste chat                           Start interactive mode
  celeste "What's the weather like?"     Quick message
  celeste config --set-key sk-xxx        Set API key
  celeste skills --list                  List skills
`)
}

// runChatTUI launches the interactive Bubble Tea TUI.
func runChatTUI() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	// Validate API key
	if cfg.APIKey == "" {
		fmt.Fprintln(os.Stderr, "No API key configured.")
		fmt.Fprintln(os.Stderr, "Set CELESTE_API_KEY environment variable or run: celeste config --set-key <key>")
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

	// Create and run TUI
	app := tui.NewApp(tuiClient)
	p := tea.NewProgram(app, tea.WithAltScreen(), tea.WithMouseCellMotion())

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running TUI: %v\n", err)
		os.Exit(1)
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

		var fullContent string
		var toolCalls []llm.ToolCallResult

		err := a.client.SendMessageStream(ctx, messages, tools, func(chunk llm.StreamChunk) {
			fullContent += chunk.Content
			if chunk.IsFinal {
				toolCalls = chunk.ToolCalls
			}
		})

		if err != nil {
			return tui.StreamErrorMsg{Err: err}
		}

		// Handle tool calls
		if len(toolCalls) > 0 {
			tc := toolCalls[0]
			return tui.SkillCallMsg{
				Call: tui.FunctionCall{
					Name:      tc.Name,
					Arguments: parseArgs(tc.Arguments),
					Status:    "executing",
					Timestamp: time.Now(),
				},
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
	setKey := fs.String("set-key", "", "Set API key")
	setURL := fs.String("set-url", "", "Set API URL")
	setModel := fs.String("set-model", "", "Set model")
	skipPersona := fs.String("skip-persona", "", "Skip persona prompt (true/false)")
	simulateTyping := fs.String("simulate-typing", "", "Simulate typing (true/false)")
	typingSpeed := fs.Int("typing-speed", 0, "Typing speed (chars/sec)")
	fs.Parse(args)

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
	}
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

