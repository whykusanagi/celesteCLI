package main

import (
	"bufio"
	"context"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"
)

// InteractiveMode launches an interactive chat with Celeste
// Displays pixel art assets and allows real-time conversation
func startInteractiveMode() {
	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "╔════════════════════════════════════════════════════════════════╗\n")
	fmt.Fprintf(os.Stderr, "║                                                                ║\n")
	fmt.Fprintf(os.Stderr, "║             Welcome to Celeste Interactive Mode                ║\n")
	fmt.Fprintf(os.Stderr, "║                                                                ║\n")
	fmt.Fprintf(os.Stderr, "║  Type 'help' for commands, 'exit' to quit                      ║\n")
	fmt.Fprintf(os.Stderr, "║                                                                ║\n")
	fmt.Fprintf(os.Stderr, "╚════════════════════════════════════════════════════════════════╝\n")
	fmt.Fprintf(os.Stderr, "\n")

	// Display Celeste asset on startup
	displayCelesteHeader()

	// Initialize input reader
	reader := bufio.NewReader(os.Stdin)

	for {
		// Show animated prompt with pulsing text
		fmt.Fprintf(os.Stderr, "\n")
		displayAnimatedPrompt()

		// Read user input
		input, err := reader.ReadString('\n')
		if err != nil {
			break
		}

		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}

		// Process command
		if strings.HasPrefix(input, "/") {
			handleCommand(strings.TrimPrefix(input, "/"))
		} else if input == "exit" || input == "quit" {
			PrintMessage(INFO, "Thanks for chatting with Celeste! Goodbye~")
			break
		} else if input == "help" {
			displayHelpMenu()
		} else {
			// Regular message - show thinking animation and process
			processUserMessage(input)
		}
	}

	fmt.Fprintf(os.Stderr, "\n")
}

// displayCelesteHeader shows Kusanagi animation and startup message
func displayCelesteHeader() {
	fmt.Fprintf(os.Stderr, "\n")

	// Load embedded kusanagi GIF animation
	gifData, err := GetKusanagiGIFBytes()
	if err != nil {
		PrintMessage(SUCCESS, "Celeste is ready to chat~")
		fmt.Fprintf(os.Stderr, "\n")
		return
	}

	animator, err := LoadGIFAnimationFromBytes(gifData, 55)
	if err != nil || animator == nil || len(animator.frames) == 0 {
		PrintMessage(SUCCESS, "Celeste is ready to chat~")
		fmt.Fprintf(os.Stderr, "\n")
		return
	}

	// Play full animation loop (all frames)
	for i := 0; i < len(animator.frames); i++ {
		fmt.Fprint(os.Stderr, "\033[H\033[2J") // Clear screen
		output := animator.RenderFrameToBlocks(i)
		fmt.Fprint(os.Stderr, output)
		fmt.Fprint(os.Stderr, "\n")
		os.Stderr.Sync()
		time.Sleep(animator.delays[i])
	}
	fmt.Fprintf(os.Stderr, "\n")
	PrintMessage(SUCCESS, "Celeste is ready to chat~")
	fmt.Fprintf(os.Stderr, "\n")
}


// processUserMessage handles regular user messages with thinking animation
func processUserMessage(message string) {
	PrintPhase(1, 3, "Processing your message...")

	// Show thinking animation
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan bool)

	go startDemonicEyeAnimation(ctx, done, os.Stderr)

	// Simulate processing time
	time.Sleep(2 * time.Second)

	cancel()
	<-done

	PrintPhase(2, 3, "Generating response...")
	time.Sleep(1 * time.Second)

	PrintPhase(3, 3, "Response ready!")
	fmt.Fprintf(os.Stderr, "\n")

	// Simulate a response
	response := fmt.Sprintf("You said: %s\n", message)
	fmt.Fprintf(os.Stderr, "Celeste: %s\n", response)
}

// handleCommand processes special interactive commands
func handleCommand(cmd string) {
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return
	}

	switch parts[0] {
	case "show":
		if len(parts) < 2 {
			PrintMessage(WARN, "Usage: /show [pixel_wink|kusanagi]")
			return
		}
		showAsset(parts[1])

	case "asset", "assets":
		displayAssetInfo()

	case "theme":
		if len(parts) < 2 {
			PrintMessage(WARN, "Usage: /theme [normal|corrupted]")
			return
		}
		setTheme(parts[1])

	case "mode":
		if len(parts) < 2 {
			PrintMessage(WARN, "Usage: /mode [tarot|nsfw|twitter|normal]")
			return
		}
		setMode(parts[1])

	case "clear":
		clearScreen()

	case "status":
		displayStatus()

	default:
		PrintMessage(ERROR, fmt.Sprintf("Unknown command: %s", parts[0]))
		PrintMessage(INFO, "Type 'help' to see available commands")
	}
}

// showAsset displays a specific asset
func showAsset(assetType string) {
	var asset AssetType
	switch strings.ToLower(assetType) {
	case "pixel_wink", "wink", "celeste":
		asset = PixelWink
	case "kusanagi", "abyss", "corrupted":
		asset = Kusanagi
	default:
		PrintMessage(ERROR, fmt.Sprintf("Unknown asset: %s", assetType))
		return
	}

	fmt.Fprintf(os.Stderr, "\n")
	// Use optimal display (animated GIF if supported, ASCII fallback)
	if err := DisplayAssetOptimal(asset); err != nil {
		PrintMessage(ERROR, fmt.Sprintf("Error displaying asset: %v", err))
	}
	fmt.Fprintf(os.Stderr, "\n")
}

// displayAssetInfo shows information about available assets
func displayAssetInfo() {
	fmt.Fprintf(os.Stderr, "\n")
	PrintMessage(INFO, "Available Assets:")
	fmt.Fprintf(os.Stderr, "\n")

	assets := ListAvailableAssets()
	for i, asset := range assets {
		fmt.Fprintf(os.Stderr, "  %d. %s\n", i+1, asset)
	}

	fmt.Fprintf(os.Stderr, "\n")
	PrintMessage(INFO, "Use: /show [pixel_wink|kusanagi] to display an asset")
	fmt.Fprintf(os.Stderr, "\n")
}

// displayHelpMenu shows available commands
func displayHelpMenu() {
	fmt.Fprintf(os.Stderr, "\n")
	PrintSeparator(HEAVY)
	fmt.Fprintf(os.Stderr, "📚 Celeste Interactive Commands\n")
	PrintSeparator(HEAVY)

	commands := []struct {
		cmd  string
		desc string
	}{
		{"/show [pixel_wink|kusanagi]", "Display pixel art asset"},
		{"/asset", "List all available assets"},
		{"/theme [normal|corrupted]", "Switch visual theme"},
		{"/mode [tarot|nsfw|twitter|normal]", "Change operation mode"},
		{"/status", "Show current status"},
		{"/clear", "Clear screen"},
		{"/help", "Show this help menu"},
		{"exit/quit", "Exit interactive mode"},
	}

	for _, cmd := range commands {
		fmt.Fprintf(os.Stderr, "  %-40s %s\n", cmd.cmd, cmd.desc)
	}

	fmt.Fprintf(os.Stderr, "\n")
	PrintSeparator(LIGHT)
	fmt.Fprintf(os.Stderr, "\n")
}

// setTheme changes the visual theme
func setTheme(theme string) {
	switch strings.ToLower(theme) {
	case "normal", "friendly", "light":
		PrintMessage(SUCCESS, "Switched to friendly theme~")
	case "corrupted", "abyss", "dark":
		PrintMessage(SUCCESS, "Switched to corrupted abyss theme~")
	default:
		PrintMessage(ERROR, fmt.Sprintf("Unknown theme: %s", theme))
	}
}

// setMode changes the operation mode
func setMode(mode string) {
	modes := map[string]string{
		"tarot":   "🔮 Tarot Reading Mode",
		"nsfw":    "⚡ NSFW Mode",
		"twitter": "🐦 Twitter Mode",
		"normal":  "✨ Normal Mode",
	}

	displayMode, exists := modes[strings.ToLower(mode)]
	if !exists {
		PrintMessage(ERROR, fmt.Sprintf("Unknown mode: %s. Available: tarot, nsfw, twitter, normal", mode))
		return
	}

	PrintMessage(SUCCESS, fmt.Sprintf("Switched to %s", displayMode))
	fmt.Fprintf(os.Stderr, "\n")
}

// clearScreen clears the terminal screen
func clearScreen() {
	fmt.Fprint(os.Stderr, "\033[2J")    // Clear screen
	fmt.Fprint(os.Stderr, "\033[H")     // Move cursor to home
	displayCelesteHeader()
}

// displayStatus shows current status and configuration
func displayStatus() {
	status := map[string]string{
		"Mode":      "Interactive Chat",
		"Theme":     "Corrupted Abyss",
		"Assets":    "2 embedded",
		"Status":    "Ready",
		"Time":      time.Now().Format("15:04:05"),
	}

	fmt.Fprintf(os.Stderr, "\n")
	PrintConfig(status)
	fmt.Fprintf(os.Stderr, "\n")
}

// displayAnimatedPrompt shows a pulsing animated prompt text above the input field
func displayAnimatedPrompt() {
	prompts := []string{
		"✨ Tell me something...",
		"💭 What's on your mind?",
		"🌙 Speak to me...",
		"📝 Your message:",
		"💬 Say something...",
		"🎀 Tell Celeste...",
	}

	// Use a simple pulsing effect with color
	pulseFrames := []string{
		"\033[38;5;213m",  // Bright magenta
		"\033[38;5;177m",  // Medium magenta
		"\033[38;5;141m",  // Light purple
		"\033[38;5;177m",  // Back to medium
	}

	selectedPrompt := prompts[rand.Intn(len(prompts))]

	// Show a brief pulsing animation
	// Use \033[2K to clear the entire line before redrawing to ensure smooth animation
	for _, colorCode := range pulseFrames {
		fmt.Fprintf(os.Stderr, "\033[2K\r%s%s\033[0m", colorCode, selectedPrompt)
		time.Sleep(100 * time.Millisecond)
	}

	// Final display in bright magenta
	fmt.Fprintf(os.Stderr, "\033[2K\r\033[38;5;213m%s\033[0m\n", selectedPrompt)
}
