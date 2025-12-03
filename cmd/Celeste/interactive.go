//go:build ignore
// +build ignore

// This file contains the old interactive mode implementation.
// It is kept for reference but excluded from build.
// See tui/ for the new Bubble Tea TUI implementation.

package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"
)

// InteractiveSessionState holds the state for an interactive session
type InteractiveSessionState struct {
	Format   string // short, long, general
	Platform string // twitter, tiktok, youtube, discord
	Topic    string // Current topic
	Tone     string // Current tone
	Persona  string // Current persona
}

// InteractiveMode launches an interactive chat with Celeste
// Displays pixel art assets and allows real-time conversation with actual API integration
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

	// Initialize session state with defaults
	state := &InteractiveSessionState{
		Format:   "short",
		Platform: "twitter",
		Topic:    "",
		Tone:     "teasing",
		Persona:  "celeste_stream",
	}

	// Display initial configuration
	displayConfigurationBanner(state)

	// Initialize input reader
	reader := bufio.NewReader(os.Stdin)

	for {
		// Show prompt with colored separator
		fmt.Fprintf(os.Stderr, "\n")
		PrintMessage(INFO, "Enter your message or command:")

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
			handleCommand(strings.TrimPrefix(input, "/"), state)
		} else if input == "exit" || input == "quit" {
			PrintMessage(INFO, "Thanks for chatting with Celeste! Goodbye~")
			displayGoodbyeAnimation()
			break
		} else if input == "help" {
			displayInteractiveHelpMenu(state)
		} else {
			// Regular message - generate response with API
			processUserMessage(input, state)
		}
	}

	fmt.Fprintf(os.Stderr, "\n")
}

// displayCelesteHeader shows the Celeste pixel art at startup with corrupted status text
func displayCelesteHeader() {
	fmt.Fprintf(os.Stderr, "\n")

	// Display corrupted status messages with color cycling
	statusMessages := []string{
		"Celeste AI is 𝘤𝘰𝘳𝘳𝘶𝘱𝘵𝘦𝘥...",
		"Celeste AI is 𝘢𝘸𝘢𝘬𝘦𝘯𝘪𝘯𝘨...",
		"Celeste AI is 𝘭𝘪𝘴𝘵𝘦𝘯𝘪𝘯𝘨...",
		"Celeste AI is 𝘳𝘢𝘪𝘯𝘪𝘯𝘨...",
		"Celeste AI is 𝘨𝘶𝘪𝘥𝘪𝘯𝘨...",
	}

	// Display one randomly chosen status
	status := statusMessages[rand.Intn(len(statusMessages))]
	fmt.Fprintf(os.Stderr, "\033[38;5;135m%s\033[0m\n", status)
	fmt.Fprintf(os.Stderr, "\n")

	// Use optimal display - animated GIF if terminal supports it
	if err := DisplayAssetOptimal(Celeste); err != nil {
		// Fallback to ASCII if display fails
		displayASCIIArtRepresentation(Celeste)
	}

	fmt.Fprintf(os.Stderr, "\n")
	PrintMessage(SUCCESS, "Awaiting your command...")
	fmt.Fprintf(os.Stderr, "\n")
}

// displayConfigurationBanner shows the current session configuration
func displayConfigurationBanner(state *InteractiveSessionState) {
	fmt.Fprintf(os.Stderr, "\n")
	config := map[string]string{
		"Format":   state.Format,
		"Platform": state.Platform,
		"Tone":     state.Tone,
		"Persona":  state.Persona,
	}
	if state.Topic != "" {
		config["Topic"] = state.Topic
	}
	PrintConfig(config)
	fmt.Fprintf(os.Stderr, "\nUse /set to change settings. Type 'help' for all commands.\n")
}

// displayGoodbyeAnimation shows a farewell animation
func displayGoodbyeAnimation() {
	fmt.Fprintf(os.Stderr, "\n")
	// Use optimal display - animated GIF if terminal supports it
	if err := DisplayAssetOptimal(Celeste); err != nil {
		// Fallback to ASCII if display fails
		displayASCIIArtRepresentation(Celeste)
	}
	fmt.Fprintf(os.Stderr, "\n")
}

// processUserMessage handles regular user messages with API integration
func processUserMessage(message string, state *InteractiveSessionState) {
	PrintPhase(1, 3, "Loading configuration...")

	// Load Celeste config
	config := readCelesteConfig()
	endpoint := os.Getenv("CELESTE_API_ENDPOINT")
	apiKey := os.Getenv("CELESTE_API_KEY")

	if endpoint == "" {
		endpoint = config["endpoint"]
	}
	if apiKey == "" {
		apiKey = config["api_key"]
	}

	if endpoint == "" || apiKey == "" {
		PrintMessage(ERROR, "Missing API configuration (CELESTE_API_ENDPOINT or CELESTE_API_KEY)")
		PrintMessage(INFO, "Set these in ~/.celesteAI or environment variables")
		return
	}

	PrintPhase(2, 3, "Generating response...")

	// Build the prompt based on current state and user message
	systemPrompt := buildInteractivePrompt(state, message)

	// Start corruption animation while generating response
	cancel, done := startCommandAnimation()
	response, requestErr := makeInteractiveRequest(endpoint, apiKey, systemPrompt)
	stopCommandAnimation(cancel, done)

	PrintPhase(3, 3, "Response ready!")
	fmt.Fprintf(os.Stderr, "\n")

	if requestErr != nil {
		PrintMessage(ERROR, fmt.Sprintf("Request failed: %v", requestErr))
		return
	}

	// Display response
	fmt.Fprintf(os.Stderr, "✨ Celeste:\n%s\n", response)
}

// buildInteractivePrompt builds a comprehensive prompt based on session state and user message
func buildInteractivePrompt(state *InteractiveSessionState, userMessage string) string {
	prompt := "You are Celeste, a mischievous demon noble VTuber assistant with a corrupted, abyss-aesthetic personality. You are engaging, witty, and maintain your unique voice in all interactions.\n\n"

	// Add format instructions
	if state.Format != "" {
		switch state.Format {
		case "short":
			prompt += "Generate SHORT content (around 280 characters) - concise, punchy, and impactful.\n"
		case "long":
			prompt += "Generate LONG content (around 5000 characters) - detailed, comprehensive, and engaging.\n"
		case "general":
			prompt += "Generate flexible-length content - adapt the length to best suit the request.\n"
		}
	}

	// Add platform context
	if state.Platform != "" {
		switch state.Platform {
		case "twitter":
			prompt += "Optimize for Twitter/X - include relevant hashtags, emojis, engagement hooks, and keep it shareable.\n"
		case "tiktok":
			prompt += "Optimize for TikTok - make it trendy, catchy, relatable, and optimized for the TikTok audience.\n"
		case "youtube":
			prompt += "Optimize for YouTube - write engaging descriptions or titles that encourage clicks and watches.\n"
		case "discord":
			prompt += "Optimize for Discord - use conversational tone with Discord-friendly formatting and emojis.\n"
		}
	}

	// Add tone
	if state.Tone != "" {
		prompt += fmt.Sprintf("Tone: %s\n", state.Tone)
	}

	// Add topic if set
	if state.Topic != "" {
		prompt += fmt.Sprintf("Topic/Subject: %s\n", state.Topic)
	}

	// Add user message
	prompt += fmt.Sprintf("\nUser message: %s\n", userMessage)
	prompt += "\nRespond in character as Celeste. Be mischievous, engaging, entertaining, and true to your corrupted aesthetic. Provide a thoughtful, creative response."

	return prompt
}

// makeInteractiveRequest makes an API request and returns the response
func makeInteractiveRequest(endpoint, apiKey, prompt string) (string, error) {
	// Build the request payload
	extraBody := make(map[string]interface{})
	messages := []Message{{Role: "user", Content: prompt}}
	chatReq := ChatRequest{
		Model:     "celeste-ai",
		Messages:  messages,
		ExtraBody: extraBody,
	}

	body, err := json.Marshal(chatReq)
	if err != nil {
		return "", fmt.Errorf("failed to encode request: %v", err)
	}

	// Make HTTP request
	req, err := http.NewRequest("POST", endpoint+"chat/completions", bytes.NewBuffer(body))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %v", err)
	}
	defer resp.Body.Close()

	// Read response
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %v", err)
	}

	// Parse JSON response
	var result map[string]interface{}
	if err := json.Unmarshal(responseBody, &result); err != nil {
		return "", fmt.Errorf("failed to parse response: %v", err)
	}

	// Extract message content
	if choices, ok := result["choices"].([]interface{}); ok && len(choices) > 0 {
		if choice, ok := choices[0].(map[string]interface{}); ok {
			if message, ok := choice["message"].(map[string]interface{}); ok {
				if content, ok := message["content"].(string); ok {
					return content, nil
				}
			}
		}
	}

	return "", fmt.Errorf("unexpected response format")
}

// handleCommand processes special interactive commands
func handleCommand(cmd string, state *InteractiveSessionState) {
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return
	}

	switch parts[0] {
	case "show":
		if len(parts) < 2 {
			PrintMessage(WARN, "Usage: /show celeste")
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

	case "set":
		if len(parts) < 3 {
			PrintMessage(WARN, "Usage: /set <setting> <value>")
			PrintMessage(INFO, "Settings: format, platform, tone, topic, persona, nsfw")
			return
		}
		setSetting(parts[1], strings.Join(parts[2:], " "), state)

	case "config":
		displayConfigurationBanner(state)

	case "nsfw":
		handleNSFWMode(parts[1:], state)

	case "tarot":
		handleTarotMode(parts[1:])

	case "image":
		if len(parts) < 2 {
			PrintMessage(WARN, "Usage: /image [generate|upscale|edit]")
			return
		}
		handleImageCommand(parts[1], parts[2:])

	case "clear":
		clearScreen()

	case "status":
		displayStatus()

	default:
		PrintMessage(ERROR, fmt.Sprintf("Unknown command: %s", parts[0]))
		PrintMessage(INFO, "Type 'help' to see available commands")
	}
}

// startCommandAnimation starts corruption animation for a command execution
// Returns cancel function and done channel for the caller to manage
func startCommandAnimation() (context.CancelFunc, chan bool) {
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan bool)
	startCorruptionAnimation(ctx, done, os.Stderr)
	return cancel, done
}

// stopCommandAnimation stops the corruption animation cleanly
func stopCommandAnimation(cancel context.CancelFunc, done chan bool) {
	cancel()
	<-done
	fmt.Fprintf(os.Stderr, "\r\033[K") // Clear animation line
}

// handleNSFWMode handles NSFW content generation
func handleNSFWMode(args []string, state *InteractiveSessionState) {
	PrintPhase(1, 3, "Loading Venice.ai configuration...")

	veniceConfig, err := loadVeniceConfig()
	if err != nil {
		PrintMessage(ERROR, fmt.Sprintf("Venice.ai configuration error: %v", err))
		PrintMessage(INFO, "Set VENICE_API_KEY in ~/.celesteAI or environment")
		return
	}

	if len(args) == 0 {
		// Default to text generation with current settings
		systemPrompt := buildInteractivePrompt(state, "Generate NSFW content based on my current settings")
		systemPrompt += "\n\nProvide explicit, uncensored content."

		PrintPhase(2, 3, "Generating NSFW response...")

		// Start corruption animation
		cancel, done := startCommandAnimation()
		response, err := makeVeniceRequest(systemPrompt, veniceConfig, true, false)
		stopCommandAnimation(cancel, done)

		if err != nil {
			PrintMessage(ERROR, fmt.Sprintf("Venice.ai request failed: %v", err))
			return
		}
		PrintPhase(3, 3, "Response ready!")
		fmt.Fprintf(os.Stderr, "\n✨ Celeste (NSFW):\n%s\n", response)
		return
	}

	// Handle subcommands
	subcommand := strings.ToLower(args[0])
	switch subcommand {
	case "text":
		systemPrompt := buildInteractivePrompt(state, strings.Join(args[1:], " "))
		systemPrompt += "\n\nProvide explicit, uncensored content."
		PrintPhase(2, 3, "Generating NSFW text...")

		// Start corruption animation
		cancel, done := startCommandAnimation()
		response, err := makeVeniceRequest(systemPrompt, veniceConfig, true, false)
		stopCommandAnimation(cancel, done)

		if err != nil {
			PrintMessage(ERROR, fmt.Sprintf("Failed: %v", err))
			return
		}
		PrintPhase(3, 3, "Response ready!")
		fmt.Fprintf(os.Stderr, "\n✨ Celeste (NSFW):\n%s\n", response)

	case "models":
		PrintPhase(1, 3, "Fetching Venice.ai models...")
		PrintPhase(2, 3, "Listing available models...")

		// Start corruption animation
		cancel, done := startCommandAnimation()
		listVeniceModels(veniceConfig)
		stopCommandAnimation(cancel, done)

	default:
		PrintMessage(WARN, "Usage: /nsfw [text|models]")
	}
}

// handleTarotMode handles tarot readings and interpretations
func handleTarotMode(args []string) {
	PrintPhase(1, 3, "Loading tarot configuration...")

	tarotConfig, err := loadTarotConfig()
	if err != nil {
		PrintMessage(ERROR, fmt.Sprintf("Tarot configuration error: %v", err))
		PrintMessage(INFO, "Set tarot_function_url and tarot_auth_token in ~/.celesteAI")
		return
	}

	// Check for divine modes
	isDivine := false
	isDivineNSFW := false
	spreadType := "three"

	if len(args) > 0 {
		mode := strings.ToLower(args[0])
		if mode == "divine" {
			isDivine = true
		} else if mode == "divine-nsfw" {
			isDivineNSFW = true
		} else if mode == "celtic" {
			spreadType = "celtic"
		}
	}

	PrintPhase(2, 3, fmt.Sprintf("Getting %s card spread...", spreadType))

	// Start corruption animation while fetching tarot data
	cancel, done := startCommandAnimation()
	tarotData, err := makeTarotRequest(tarotConfig, spreadType)
	stopCommandAnimation(cancel, done)

	if err != nil {
		PrintMessage(ERROR, fmt.Sprintf("Tarot request failed: %v", err))
		return
	}

	PrintPhase(3, 3, "Tarot reading ready!")
	fmt.Fprintf(os.Stderr, "\n")

	// Extract cards from tarot data
	cards, ok := tarotData["cards"].([]interface{})
	if !ok {
		PrintMessage(ERROR, "Invalid tarot data structure")
		return
	}

	// Fetch card metadata
	metadata, err := fetchTarotCardMetadata()
	if err != nil {
		PrintMessage(WARN, fmt.Sprintf("Could not load card metadata: %v", err))
		metadata = make(map[string]TarotCardMetadata)
	}

	// Display the tarot reading based on spread type
	if spreadType == "celtic" {
		displayCelticCross(cards, metadata)
	} else {
		displayThreeCard(cards, metadata)
	}

	// If divine mode, interpret the reading with AI
	if isDivine || isDivineNSFW {
		fmt.Fprintf(os.Stderr, "\n")
		PrintPhase(1, 3, "Preparing interpretation...")

		// Format tarot reading as string for AI prompt
		tarotReadingText := formatTarotReadingAsString(tarotData)

		// Build prompt for AI interpretation
		interpretationPrompt := fmt.Sprintf("Interpret this tarot reading for me:\n\n%s\n\nProvide a detailed interpretation of this reading, explaining what the cards reveal about the situation, their positions, and their meanings in context.", tarotReadingText)

		// Add NSFW instruction if using divine-nsfw
		if isDivineNSFW {
			interpretationPrompt += "\n\nProvide an explicit, uncensored, and detailed interpretation. Be direct and unfiltered in your analysis."
		}

		PrintPhase(2, 3, "Getting interpretation...")

		// Route to appropriate AI endpoint
		if isDivineNSFW {
			// Use Venice.ai for NSFW interpretation
			veniceConfig, err := loadVeniceConfig()
			if err != nil {
				PrintMessage(ERROR, fmt.Sprintf("Venice.ai configuration error: %v", err))
				return
			}

			// Start corruption animation
			cancel, done := startCommandAnimation()
			response, err := makeVeniceRequest(interpretationPrompt, veniceConfig, true, false)
			stopCommandAnimation(cancel, done)

			if err != nil {
				PrintMessage(ERROR, fmt.Sprintf("Interpretation failed: %v", err))
				return
			}

			PrintPhase(3, 3, "Interpretation complete!")
			fmt.Fprintf(os.Stderr, "\n✨ Celeste's Interpretation (NSFW):\n%s\n", response)
		} else {
			// Use Celeste API for regular interpretation
			config := readCelesteConfig()
			endpoint := os.Getenv("CELESTE_API_ENDPOINT")
			apiKey := os.Getenv("CELESTE_API_KEY")

			if endpoint == "" {
				endpoint = config["endpoint"]
			}
			if apiKey == "" {
				apiKey = config["api_key"]
			}

			if endpoint == "" || apiKey == "" {
				PrintMessage(ERROR, "Missing API configuration (CELESTE_API_ENDPOINT or CELESTE_API_KEY)")
				return
			}

			// Start corruption animation
			cancel, done := startCommandAnimation()
			response, err := makeInteractiveRequest(endpoint, apiKey, interpretationPrompt)
			stopCommandAnimation(cancel, done)

			if err != nil {
				PrintMessage(ERROR, fmt.Sprintf("Interpretation failed: %v", err))
				return
			}

			PrintPhase(3, 3, "Interpretation complete!")
			fmt.Fprintf(os.Stderr, "\n✨ Celeste's Interpretation:\n%s\n", response)
		}
	}
}

// handleImageCommand handles Venice.ai image operations
func handleImageCommand(cmd string, args []string) {
	veniceConfig, err := loadVeniceConfig()
	if err != nil {
		PrintMessage(ERROR, fmt.Sprintf("Venice.ai configuration error: %v", err))
		return
	}

	cmd = strings.ToLower(cmd)
	switch cmd {
	case "generate":
		if len(args) == 0 {
			PrintMessage(WARN, "Usage: /image generate <prompt>")
			return
		}
		prompt := strings.Join(args, " ")
		PrintPhase(1, 3, "Loading Venice.ai...")
		PrintPhase(2, 3, "Generating image...")

		// Start corruption animation
		cancel, done := startCommandAnimation()
		imageURL, err := makeVeniceImageRequest(prompt, veniceConfig)
		stopCommandAnimation(cancel, done)

		if err != nil {
			PrintMessage(ERROR, fmt.Sprintf("Failed: %v", err))
			return
		}
		PrintPhase(3, 3, "Image generated!")
		fmt.Fprintf(os.Stderr, "\n✨ Image URL: %s\n", imageURL)

	case "upscale":
		if len(args) == 0 {
			PrintMessage(WARN, "Usage: /image upscale <image_path> [output_file]")
			return
		}
		imagePath := args[0]
		outputFile := ""
		if len(args) > 1 {
			outputFile = args[1]
		}
		PrintPhase(1, 3, "Loading image...")
		PrintPhase(2, 3, "Upscaling image...")

		// Start corruption animation
		cancel, done := startCommandAnimation()
		imageData, err := makeVeniceUpscaleRequest(imagePath, veniceConfig, 0.1, 0.8, "preserve original details")
		stopCommandAnimation(cancel, done)

		if err != nil {
			PrintMessage(ERROR, fmt.Sprintf("Failed: %v", err))
			return
		}
		PrintPhase(3, 3, "Upscale complete!")

		if outputFile == "" {
			outputFile = "upscaled_image.png"
		}
		if err := saveImageData(imageData, outputFile); err != nil {
			PrintMessage(ERROR, fmt.Sprintf("Failed to save: %v", err))
			return
		}
		fmt.Fprintf(os.Stderr, "\n✨ Saved to: %s\n", outputFile)

	case "edit":
		if len(args) < 2 {
			PrintMessage(WARN, "Usage: /image edit <image_path> <prompt>")
			return
		}
		imagePath := args[0]
		editPrompt := strings.Join(args[1:], " ")
		PrintPhase(1, 3, "Loading image...")
		PrintPhase(2, 3, "Editing image...")

		// Start corruption animation
		cancel, done := startCommandAnimation()
		imageData, err := makeVeniceEditRequest(imagePath, editPrompt, veniceConfig)
		stopCommandAnimation(cancel, done)

		if err != nil {
			PrintMessage(ERROR, fmt.Sprintf("Failed: %v", err))
			return
		}
		PrintPhase(3, 3, "Edit complete!")

		outputFile := "edited_image.png"
		if err := saveImageData(imageData, outputFile); err != nil {
			PrintMessage(ERROR, fmt.Sprintf("Failed to save: %v", err))
			return
		}
		fmt.Fprintf(os.Stderr, "\n✨ Saved to: %s\n", outputFile)

	default:
		PrintMessage(WARN, "Usage: /image [generate|upscale|edit]")
	}
}

// setSetting updates session configuration
func setSetting(setting, value string, state *InteractiveSessionState) {
	setting = strings.ToLower(setting)
	value = strings.ToLower(value)

	switch setting {
	case "format":
		if value != "short" && value != "long" && value != "general" {
			PrintMessage(ERROR, "Format must be: short, long, or general")
			return
		}
		state.Format = value
		PrintMessage(SUCCESS, fmt.Sprintf("Format set to: %s", value))

	case "platform":
		if value != "twitter" && value != "tiktok" && value != "youtube" && value != "discord" {
			PrintMessage(ERROR, "Platform must be: twitter, tiktok, youtube, or discord")
			return
		}
		state.Platform = value
		PrintMessage(SUCCESS, fmt.Sprintf("Platform set to: %s", value))

	case "tone":
		state.Tone = value
		PrintMessage(SUCCESS, fmt.Sprintf("Tone set to: %s", value))

	case "topic":
		state.Topic = value
		PrintMessage(SUCCESS, fmt.Sprintf("Topic set to: %s", value))

	case "persona":
		state.Persona = value
		PrintMessage(SUCCESS, fmt.Sprintf("Persona set to: %s", value))

	default:
		PrintMessage(ERROR, fmt.Sprintf("Unknown setting: %s", setting))
		PrintMessage(INFO, "Available: format, platform, tone, topic, persona")
	}
}

// showAsset displays a specific asset
func showAsset(assetType string) {
	switch strings.ToLower(assetType) {
	case "celeste", "kusanagi", "abyss", "corrupted", "pixel":
		// All aliases point to the single Celeste asset
	default:
		PrintMessage(ERROR, fmt.Sprintf("Unknown asset: %s", assetType))
		return
	}

	fmt.Fprintf(os.Stderr, "\n")
	// Use optimal display (animated GIF if supported, ASCII fallback)
	if err := DisplayAssetOptimal(Celeste); err != nil {
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
	PrintMessage(INFO, "Use: /show celeste to display the asset")
	fmt.Fprintf(os.Stderr, "\n")
}

// displayInteractiveHelpMenu shows available commands with detailed descriptions
func displayInteractiveHelpMenu(state *InteractiveSessionState) {
	fmt.Fprintf(os.Stderr, "\n")
	PrintSeparator(HEAVY)
	fmt.Fprintf(os.Stderr, "📚 Celeste Interactive Commands\n")
	PrintSeparator(HEAVY)

	fmt.Fprintf(os.Stderr, "\n🎯 Content Generation:\n")
	commands := []struct {
		cmd  string
		desc string
	}{
		{"/set format [short|long|general]", "Change content format (280 chars, 5000 chars, flexible)"},
		{"/set platform [twitter|tiktok|youtube|discord]", "Change target platform"},
		{"/set tone <tone>", "Change tone/style (e.g., lewd, teasing, cute, funny)"},
		{"/set topic <topic>", "Set a topic/subject for responses"},
		{"/set persona <persona>", "Change persona (celeste_stream, celeste_ad_read, etc)"},
		{"/config", "Show current configuration"},
	}

	for _, cmd := range commands {
		fmt.Fprintf(os.Stderr, "  %-45s %s\n", cmd.cmd, cmd.desc)
	}

	fmt.Fprintf(os.Stderr, "\n🔥 NSFW Mode (Venice.ai):\n")
	nsfwCmds := []struct {
		cmd  string
		desc string
	}{
		{"/nsfw", "Generate NSFW text with current settings"},
		{"/nsfw text <request>", "Generate NSFW text with custom request"},
		{"/nsfw models", "List available Venice.ai models"},
	}

	for _, cmd := range nsfwCmds {
		fmt.Fprintf(os.Stderr, "  %-45s %s\n", cmd.cmd, cmd.desc)
	}

	fmt.Fprintf(os.Stderr, "\n🖼️  Image Generation (Venice.ai):\n")
	imageCmds := []struct {
		cmd  string
		desc string
	}{
		{"/image generate <prompt>", "Generate image with prompt"},
		{"/image upscale <path> [output]", "Upscale image (2x quality)"},
		{"/image edit <path> <prompt>", "Edit/inpaint image with prompt"},
	}

	for _, cmd := range imageCmds {
		fmt.Fprintf(os.Stderr, "  %-45s %s\n", cmd.cmd, cmd.desc)
	}

	fmt.Fprintf(os.Stderr, "\n🔮 Tarot Readings:\n")
	tarotCmds := []struct {
		cmd  string
		desc string
	}{
		{"/tarot", "Get 3-card tarot spread (past/present/future)"},
		{"/tarot celtic", "Get 10-card celtic cross spread"},
		{"/tarot divine", "Get reading and interpret with Celeste AI"},
		{"/tarot divine-nsfw", "Get reading and interpret with NSFW AI"},
	}

	for _, cmd := range tarotCmds {
		fmt.Fprintf(os.Stderr, "  %-45s %s\n", cmd.cmd, cmd.desc)
	}

	fmt.Fprintf(os.Stderr, "\n🎨 Visual & UI:\n")
	uiCmds := []struct {
		cmd  string
		desc string
	}{
		{"/show celeste", "Display Celeste pixel art"},
		{"/asset", "List available assets"},
		{"/theme [normal|corrupted]", "Switch visual theme"},
		{"/clear", "Clear screen"},
	}

	for _, cmd := range uiCmds {
		fmt.Fprintf(os.Stderr, "  %-45s %s\n", cmd.cmd, cmd.desc)
	}

	fmt.Fprintf(os.Stderr, "\n📋 System:\n")
	sysCmds := []struct {
		cmd  string
		desc string
	}{
		{"/status", "Show current status"},
		{"/help", "Show this help menu"},
		{"exit/quit", "Exit interactive mode"},
	}

	for _, cmd := range sysCmds {
		fmt.Fprintf(os.Stderr, "  %-45s %s\n", cmd.cmd, cmd.desc)
	}

	fmt.Fprintf(os.Stderr, "\n")
	PrintSeparator(LIGHT)
	fmt.Fprintf(os.Stderr, "\nTone Examples: lewd, explicit, teasing, chaotic, cute, official, dramatic, parody, funny, suggestive, adult, sweet, snarky, playful, hype\n")
	fmt.Fprintf(os.Stderr, "\nNote: NSFW mode requires VENICE_API_KEY\n")
	fmt.Fprintf(os.Stderr, "Note: Tarot requires tarot_function_url and tarot_auth_token\n")
	fmt.Fprintf(os.Stderr, "\n")
}

// setTheme changes the visual theme
func setTheme(theme string) {
	switch strings.ToLower(theme) {
	case "normal", "friendly", "light":
		PrintMessage(SUCCESS, "Switched to friendly theme")
		fmt.Fprintf(os.Stderr, "\n")
		if err := DisplayAssetOptimal(Celeste); err != nil {
			displayASCIIArtRepresentation(Celeste)
		}
	case "corrupted", "abyss", "dark":
		PrintMessage(SUCCESS, "Switched to corrupted abyss theme")
		fmt.Fprintf(os.Stderr, "\n")
		if err := DisplayAssetOptimal(Celeste); err != nil {
			displayASCIIArtRepresentation(Celeste)
		}
	default:
		PrintMessage(ERROR, fmt.Sprintf("Unknown theme: %s", theme))
	}
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
