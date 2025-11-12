package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/sashabaranov/go-openai"
	"gopkg.in/yaml.v3"
)

// Message represents a chat message.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatRequest represents the request payload for the chat API.
type ChatRequest struct {
	Model     string                 `json:"model"`
	Messages  []Message              `json:"messages"`
	ExtraBody map[string]interface{} `json:"extra_body,omitempty"`
}

// GameMetadata represents game information from IGDB.
type GameMetadata struct {
	ID      int      `json:"id"`
	Name    string   `json:"name"`
	Summary string   `json:"summary"`
	Genres  []string `json:"genres"`
	Website string   `json:"website"`
}

// S3Config holds DigitalOcean Spaces configuration
type S3Config struct {
	Endpoint        string `json:"endpoint"`
	Region          string `json:"region"`
	AccessKeyID     string `json:"access_key_id"`
	SecretAccessKey string `json:"secret_access_key"`
	BucketName      string `json:"bucket_name"`
}

// VeniceConfig holds Venice.ai configuration for NSFW mode
type VeniceConfig struct {
	APIKey   string `json:"api_key"`
	BaseURL  string `json:"base_url"`
	Model    string `json:"model"`
	Upscaler string `json:"upscaler"`
}

// VeniceModel represents a Venice.ai model
type VeniceModel struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Type        string `json:"type"`
}

// VeniceModelsResponse represents the response from Venice.ai models endpoint
type VeniceModelsResponse struct {
	Models []VeniceModel `json:"models"`
}

// ConversationEntry represents a single conversation for storage
type ConversationEntry struct {
	ID          string                 `json:"id"`
	Timestamp   time.Time              `json:"timestamp"`
	UserID      string                 `json:"user_id"`
	ContentType string                 `json:"content_type"`
	Tone        string                 `json:"tone"`
	Game        string                 `json:"game"`
	Persona     string                 `json:"persona"`
	Prompt      string                 `json:"prompt"`
	Response    string                 `json:"response"`
	TokensUsed  map[string]interface{} `json:"tokens_used"`
	Metadata    map[string]interface{} `json:"metadata"`

	// Enhanced fields for OpenSearch RAG
	Intent    string   `json:"intent"`
	Purpose   string   `json:"purpose"`
	Topics    []string `json:"topics"`
	Sentiment string   `json:"sentiment"`
	Platform  string   `json:"platform"`
	Tags      []string `json:"tags"`
	Context   string   `json:"context"`
	Success   bool     `json:"success"`
}

// PersonalityConfig holds the parsed personality.yml configuration
type PersonalityConfig struct {
	Version  string `yaml:"version"`
	Metadata struct {
		Project         string `yaml:"project"`
		Maintainer      string `yaml:"maintainer"`
		BrandVoiceShort string `yaml:"brand_voice_short"`
	} `yaml:"metadata"`
	Persona struct {
		Name       string   `yaml:"name"`
		Aliases    []string `yaml:"aliases"`
		CoreTraits []string `yaml:"core_traits"`
		Tone       struct {
			Default string `yaml:"default"`
			Stream  string `yaml:"stream"`
			AdRead  string `yaml:"ad_read"`
		} `yaml:"tone"`
	} `yaml:"persona"`
	PromptKits map[string]struct {
		System string `yaml:"system"`
		Style  struct {
			AllowEmotes    bool   `yaml:"allow_emotes"`
			SentenceLength string `yaml:"sentence_length"`
			KeepEnergy     string `yaml:"keep_energy"`
		} `yaml:"style"`
		Safety struct {
			Refuse []string `yaml:"refuse"`
		} `yaml:"safety"`
	} `yaml:"prompt_kits"`
}

// loadPersonalityConfig loads and parses the personality.yml file.
func loadPersonalityConfig() (*PersonalityConfig, error) {
	// First try to load from config directory (~/.celeste/personality.yml)
	homeDir, err := os.UserHomeDir()
	if err != nil {
		// Fallback to current directory if home directory can't be determined
		data, err := os.ReadFile("personality.yml")
		if err != nil {
			return nil, fmt.Errorf("failed to read personality.yml: %v", err)
		}
		var config PersonalityConfig
		if err := yaml.Unmarshal(data, &config); err != nil {
			return nil, fmt.Errorf("failed to parse personality.yml: %v", err)
		}
		return &config, nil
	}

	configPath := filepath.Join(homeDir, ".celeste", "personality.yml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		// Fallback to current directory for backward compatibility
		data, err = os.ReadFile("personality.yml")
		if err != nil {
			return nil, fmt.Errorf("failed to read personality.yml from ~/.celeste/personality.yml or current directory: %v", err)
		}
	}

	var config PersonalityConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse personality.yml: %v", err)
	}

	return &config, nil
}

// getPersonalityPrompt returns the appropriate personality prompt based on the persona.
func getPersonalityPrompt(config *PersonalityConfig, persona string) string {
	if kit, exists := config.PromptKits[persona]; exists {
		return kit.System
	}
	// Fallback to default persona
	return config.PromptKits["celeste_stream"].System
}

// fetchIGDBGameInfo queries IGDB for game information and caches results locally.
func fetchIGDBGameInfo(gameName string) (*GameMetadata, error) {
	// Load IGDB credentials
	config := readCelesteConfig()
	clientID := os.Getenv("CELESTE_IGDB_CLIENT_ID")
	clientSecret := os.Getenv("CELESTE_IGDB_CLIENT_SECRET")

	if clientID == "" {
		clientID = config["client_id"]
	}
	if clientSecret == "" {
		clientSecret = config["secret"]
	}

	if clientID == "" || clientSecret == "" {
		return nil, fmt.Errorf("missing IGDB credentials")
	}

	// Get access token
	tokenURL := "https://id.twitch.tv/oauth2/token"
	form := url.Values{}
	form.Set("client_id", clientID)
	form.Set("client_secret", clientSecret)
	form.Set("grant_type", "client_credentials")

	resp, err := http.PostForm(tokenURL, form)
	if err != nil {
		return nil, fmt.Errorf("failed to get IGDB token: %v", err)
	}
	defer resp.Body.Close()

	var tokenResp struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("failed to decode token response: %v", err)
	}

	// Query IGDB
	query := fmt.Sprintf(`fields name,summary,genres.name,website; search "%s"; limit 1;`, gameName)
	req, err := http.NewRequest("POST", "https://api.igdb.com/v4/games", strings.NewReader(query))
	if err != nil {
		return nil, fmt.Errorf("failed to create IGDB request: %v", err)
	}

	req.Header.Set("Client-ID", clientID)
	req.Header.Set("Authorization", "Bearer "+tokenResp.AccessToken)
	req.Header.Set("Content-Type", "text/plain")

	client := &http.Client{}
	igdbResp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to query IGDB: %v", err)
	}
	defer igdbResp.Body.Close()

	var games []struct {
		ID      int    `json:"id"`
		Name    string `json:"name"`
		Summary string `json:"summary"`
		Genres  []struct {
			Name string `json:"name"`
		} `json:"genres"`
		Website string `json:"website"`
	}

	if err := json.NewDecoder(igdbResp.Body).Decode(&games); err != nil {
		return nil, fmt.Errorf("failed to decode IGDB response: %v", err)
	}

	if len(games) == 0 {
		return nil, fmt.Errorf("no game found for: %s", gameName)
	}

	game := games[0]
	genres := make([]string, len(game.Genres))
	for i, genre := range game.Genres {
		genres[i] = genre.Name
	}

	return &GameMetadata{
		ID:      game.ID,
		Name:    game.Name,
		Summary: game.Summary,
		Genres:  genres,
		Website: game.Website,
	}, nil
}

// readCelesteConfig reads the ~/.celesteAI configuration file
func readCelesteConfig() map[string]string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return map[string]string{}
	}
	configPath := filepath.Join(homeDir, ".celesteAI")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return map[string]string{}
	}

	config := make(map[string]string)
	lines := bytes.Split(data, []byte("\n"))
	for _, line := range lines {
		// Skip comments
		if bytes.HasPrefix(bytes.TrimSpace(line), []byte("#")) {
			continue
		}
		parts := bytes.SplitN(line, []byte("="), 2)
		if len(parts) != 2 {
			continue
		}
		key := string(bytes.TrimSpace(parts[0]))
		val := string(bytes.TrimSpace(parts[1]))
		config[key] = val
	}
	return config
}

// TarotConfig holds tarot function configuration
type TarotConfig struct {
	FunctionURL string
	AuthToken   string
}

// loadTarotConfig loads tarot configuration from ~/.celesteAI file
func loadTarotConfig() (*TarotConfig, error) {
	config := &TarotConfig{
		FunctionURL: "https://faas-nyc1-2ef2e6cc.doserverless.co/api/v1/namespaces/fn-30b193db-d334-4dab-b5cd-ab49067f88cc/actions/tarot/logic?blocking=true&result=true",
		AuthToken:   "",
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %v", err)
	}

	celesteConfigPath := filepath.Join(homeDir, ".celesteAI")
	if _, err := os.Stat(celesteConfigPath); err == nil {
		data, err := os.ReadFile(celesteConfigPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read ~/.celesteAI: %v", err)
		}

		lines := bytes.Split(data, []byte("\n"))
		for _, line := range lines {
			// Skip comments
			if bytes.HasPrefix(bytes.TrimSpace(line), []byte("#")) {
				continue
			}
			parts := bytes.SplitN(line, []byte("="), 2)
			if len(parts) != 2 {
				continue
			}
			key := string(bytes.TrimSpace(parts[0]))
			val := string(bytes.TrimSpace(parts[1]))

			switch key {
			case "tarot_function_url":
				config.FunctionURL = val
			case "tarot_auth_token":
				config.AuthToken = val
			}
		}
	}

	// Check environment variables as fallback
	if config.FunctionURL == "" {
		config.FunctionURL = os.Getenv("TAROT_FUNCTION_URL")
	}
	if config.AuthToken == "" {
		config.AuthToken = os.Getenv("TAROT_AUTH_TOKEN")
	}

	if config.AuthToken == "" {
		return nil, fmt.Errorf("missing tarot auth token. Set TAROT_AUTH_TOKEN environment variable or tarot_auth_token in ~/.celesteAI")
	}

	return config, nil
}

// makeTarotRequest makes a request to the tarot function
func makeTarotRequest(config *TarotConfig, spreadType string) (map[string]interface{}, error) {
	// Build request body with spread type
	requestBody := map[string]interface{}{
		"spread_type": spreadType,
	}
	reqBodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to encode request: %v", err)
	}

	req, err := http.NewRequest("POST", config.FunctionURL, bytes.NewBuffer(reqBodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", config.AuthToken)

	// Use a shorter timeout - the function should respond quickly
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		// Check if it's a timeout error
		if urlErr, ok := err.(interface{ Timeout() bool }); ok && urlErr.Timeout() {
			return nil, fmt.Errorf("request timed out after 10 seconds. The tarot function may be slow or unavailable")
		}
		return nil, fmt.Errorf("failed to make request: %v", err)
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("tarot request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var tarotData map[string]interface{}
	if err := json.Unmarshal(body, &tarotData); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v (body: %s)", err, string(body))
	}

	return tarotData, nil
}

// TarotCardMetadata holds metadata for a tarot card
type TarotCardMetadata struct {
	Name     string `json:"name"`
	Upright  string `json:"upright"`
	Reversed string `json:"reversed"`
	Suit     string `json:"suit,omitempty"`
	Number   string `json:"number,omitempty"`
	Element  string `json:"element,omitempty"`
	Planet   string `json:"planet,omitempty"`
	Zodiac   string `json:"zodiac,omitempty"`
	Symbol   string `json:"symbol,omitempty"`
	Color    string `json:"color,omitempty"`
}

// fetchTarotCardMetadata fetches tarot card metadata from S3
func fetchTarotCardMetadata() (map[string]TarotCardMetadata, error) {
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	resp, err := client.Get("https://s3.whykusanagi.xyz/tarot_cards.json")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("failed to fetch tarot metadata: status %d", resp.StatusCode)
	}

	var deck []TarotCardMetadata
	if err := json.NewDecoder(resp.Body).Decode(&deck); err != nil {
		return nil, err
	}

	metadata := make(map[string]TarotCardMetadata)
	for _, card := range deck {
		metadata[card.Name] = card
	}

	return metadata, nil
}

// getCardDecoration returns decorative elements for a card based on its metadata
func getCardDecoration(cardName string, orientation string, metadata map[string]TarotCardMetadata) (string, string) {
	card, exists := metadata[cardName]
	if !exists {
		// Default decorations
		if orientation == "reversed" {
			return "↻", "⚠"
		}
		return "✦", "▲"
	}

	// Orientation indicator
	orientIcon := "▲"
	if orientation == "reversed" {
		orientIcon = "↻"
	}

	// Card-specific symbol or default
	symbol := "✦"
	if card.Symbol != "" {
		symbol = card.Symbol
	} else if card.Suit != "" {
		// Use suit-based symbols
		switch strings.ToLower(card.Suit) {
		case "wands", "wand":
			symbol = "🔥"
		case "cups", "cup":
			symbol = "💧"
		case "swords", "sword":
			symbol = "⚔"
		case "pentacles", "pentacle", "coins", "coin":
			symbol = "💰"
		case "major arcana", "major":
			symbol = "⭐"
		}
	}

	return symbol, orientIcon
}

// formatTarotReading formats and displays tarot cards in a visual layout
func formatTarotReading(tarotData map[string]interface{}) {
	spreadName, _ := tarotData["spread_name"].(string)
	spreadType, _ := tarotData["spread_type"].(string)
	cards, ok := tarotData["cards"].([]interface{})
	if !ok {
		// Fallback to JSON if structure is unexpected
		jsonData, _ := json.MarshalIndent(tarotData, "", "  ")
		fmt.Println(string(jsonData))
		return
	}

	// Fetch card metadata for decorations
	cardMetadata, err := fetchTarotCardMetadata()
	if err != nil {
		// Continue without metadata if fetch fails
		cardMetadata = make(map[string]TarotCardMetadata)
	}

	// Mystical header
	fmt.Println()
	fmt.Println("╔═══════════════════════════════════════════════════════════════╗")
	fmt.Printf("║  🔮 %-60s ║\n", spreadName)
	fmt.Println("╚═══════════════════════════════════════════════════════════════╝")
	fmt.Println()

	if spreadType == "celtic" || len(cards) == 10 {
		displayCelticCross(cards, cardMetadata)
	} else {
		displayThreeCard(cards, cardMetadata)
	}
}

// displayThreeCard displays a 3-card spread horizontally with elegant styling
func displayThreeCard(cards []interface{}, metadata map[string]TarotCardMetadata) {
	if len(cards) != 3 {
		// Fallback
		displayCardList(cards, metadata)
		return
	}

	positions := make([]string, 3)
	names := make([]string, 3)
	meanings := make([]string, 3)
	orientations := make([]string, 3)
	symbols := make([]string, 3)
	orientIcons := make([]string, 3)
	positionLabels := []string{"◄ PAST", "● PRESENT", "► FUTURE"}

	for i, card := range cards {
		cardMap, ok := card.(map[string]interface{})
		if !ok {
			continue
		}
		// Extract position without number
		posStr := fmt.Sprintf("%v", cardMap["position"])
		posParts := strings.SplitN(posStr, ". ", 2)
		if len(posParts) > 1 {
			positions[i] = posParts[1]
		} else {
			positions[i] = posStr
		}
		names[i] = fmt.Sprintf("%v", cardMap["card_name"])
		meanings[i] = fmt.Sprintf("%v", cardMap["card_meaning"])

		// Get orientation
		orientation := "upright"
		if orient, ok := cardMap["orientation"].(string); ok {
			orientation = orient
		}
		orientations[i] = orientation

		// Get decorations
		symbols[i], orientIcons[i] = getCardDecoration(names[i], orientation, metadata)
	}

	// Card dimensions
	cardWidth := 38
	cardPadding := 2

	// Build each card's content
	allCards := make([][]string, 3)
	for i := 0; i < 3; i++ {
		var cardLines []string

		// Top border with decorative corners
		cardLines = append(cardLines, "┌"+strings.Repeat("─", cardWidth)+"┐")

		// Position label - centered and bold
		posLabel := positionLabels[i]
		posPadding := (cardWidth - len(posLabel)) / 2
		cardLines = append(cardLines, fmt.Sprintf("│%s%s%s│",
			strings.Repeat(" ", posPadding), posLabel, strings.Repeat(" ", cardWidth-posPadding-len(posLabel))))

		// Divider
		cardLines = append(cardLines, "├"+strings.Repeat("─", cardWidth)+"┤")

		// Card name - centered with symbol
		name := names[i]
		if len(name) > cardWidth-10 {
			name = truncate(name, cardWidth-10)
		}
		nameDisplay := fmt.Sprintf("%s %s %s", symbols[i], name, orientIcons[i])
		namePadding := (cardWidth - len(nameDisplay)) / 2
		if namePadding < 0 {
			namePadding = 0
		}
		cardLines = append(cardLines, fmt.Sprintf("│%s%s%s│",
			strings.Repeat(" ", namePadding), nameDisplay, strings.Repeat(" ", cardWidth-namePadding-len(nameDisplay))))

		// Orientation badge
		orientText := "▲ UPRIGHT"
		if orientations[i] == "reversed" {
			orientText = "↻ REVERSED"
		}
		orientPadding := (cardWidth - len(orientText)) / 2
		cardLines = append(cardLines, fmt.Sprintf("│%s%s%s│",
			strings.Repeat(" ", orientPadding), orientText, strings.Repeat(" ", cardWidth-orientPadding-len(orientText))))

		// Divider
		cardLines = append(cardLines, "├"+strings.Repeat("─", cardWidth)+"┤")

		// Meaning - wrapped and justified
		meaning := meanings[i]
		words := strings.Fields(meaning)
		currentLine := ""
		for _, word := range words {
			testLine := currentLine
			if testLine != "" {
				testLine += " "
			}
			testLine += word
			if len(testLine) <= cardWidth-cardPadding*2 {
				currentLine = testLine
			} else {
				if currentLine != "" {
					// Center the line
					linePadding := (cardWidth - len(currentLine)) / 2
					cardLines = append(cardLines, fmt.Sprintf("│%s%s%s│",
						strings.Repeat(" ", linePadding), currentLine, strings.Repeat(" ", cardWidth-linePadding-len(currentLine))))
				}
				currentLine = word
			}
		}
		if currentLine != "" {
			linePadding := (cardWidth - len(currentLine)) / 2
			cardLines = append(cardLines, fmt.Sprintf("│%s%s%s│",
				strings.Repeat(" ", linePadding), currentLine, strings.Repeat(" ", cardWidth-linePadding-len(currentLine))))
		}

		// Bottom border
		cardLines = append(cardLines, "└"+strings.Repeat("─", cardWidth)+"┘")

		allCards[i] = cardLines
	}

	// Find max height
	maxHeight := 0
	for i := 0; i < 3; i++ {
		if len(allCards[i]) > maxHeight {
			maxHeight = len(allCards[i])
		}
	}

	// Print cards side by side with spacing
	spacing := "   "
	fmt.Println()
	for row := 0; row < maxHeight; row++ {
		for i := 0; i < 3; i++ {
			if row < len(allCards[i]) {
				fmt.Print(allCards[i][row])
			} else {
				// Fill empty rows
				fmt.Print("│" + strings.Repeat(" ", cardWidth) + "│")
			}
			if i < 2 {
				fmt.Print(spacing)
			}
		}
		fmt.Println()
	}
	fmt.Println()
}

// displayCelticCross displays a Celtic Cross spread with creative visual layout
func displayCelticCross(cards []interface{}, metadata map[string]TarotCardMetadata) {
	if len(cards) != 10 {
		// Fallback
		displayCardList(cards, metadata)
		return
	}

	// Extract card data
	cardData := make([]struct {
		position    string
		name        string
		meaning     string
		orientation string
		symbol      string
		orientIcon  string
	}, 10)

	for i, card := range cards {
		cardMap, ok := card.(map[string]interface{})
		if !ok {
			continue
		}
		// Extract position without number
		posStr := fmt.Sprintf("%v", cardMap["position"])
		posParts := strings.SplitN(posStr, ". ", 2)
		if len(posParts) > 1 {
			cardData[i].position = posParts[1]
		} else {
			cardData[i].position = posStr
		}
		cardData[i].name = fmt.Sprintf("%v", cardMap["card_name"])
		cardData[i].meaning = fmt.Sprintf("%v", cardMap["card_meaning"])

		// Get orientation
		orientation := "upright"
		if orient, ok := cardMap["orientation"].(string); ok {
			orientation = orient
		}
		cardData[i].orientation = orientation

		// Get decorations
		cardData[i].symbol, cardData[i].orientIcon = getCardDecoration(cardData[i].name, orientation, metadata)
	}

	// Visual Celtic Cross layout diagram - compact overview
	fmt.Println()
	fmt.Println("                    ┌─────────────────────────┐")
	fmt.Printf("                    │ %s %-19s %s │\n", cardData[0].symbol, truncate(cardData[0].name, 17), cardData[0].orientIcon)
	fmt.Println("                    │ 1. Present Situation    │")
	fmt.Println("                    └─────────────────────────┘")
	fmt.Println("                    ┌─────────────────────────┐")
	fmt.Printf("                    │ %s %-19s %s │\n", cardData[1].symbol, truncate(cardData[1].name, 17), cardData[1].orientIcon)
	fmt.Println("                    │ 2. Challenge/Crossing   │")
	fmt.Println("                    └─────────────────────────┘")
	fmt.Println("  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐")
	fmt.Printf("  │ %s %-7s %s │  │ %s %-7s %s │  │ %s %-7s %s │  │ %s %-7s %s │\n",
		cardData[2].symbol, truncate(cardData[2].name, 5), cardData[2].orientIcon,
		cardData[3].symbol, truncate(cardData[3].name, 5), cardData[3].orientIcon,
		cardData[4].symbol, truncate(cardData[4].name, 5), cardData[4].orientIcon,
		cardData[5].symbol, truncate(cardData[5].name, 5), cardData[5].orientIcon)
	fmt.Println("  │ 3. Past     │  │ 4. Past     │  │ 5. Future   │  │ 6. Future   │")
	fmt.Println("  └─────────────┘  └─────────────┘  └─────────────┘  └─────────────┘")
	fmt.Println("                    ┌─────────────────────────┐")
	fmt.Printf("                    │ %s %-19s %s │\n", cardData[6].symbol, truncate(cardData[6].name, 17), cardData[6].orientIcon)
	fmt.Println("                    │ 7. Your Approach        │")
	fmt.Println("                    └─────────────────────────┘")
	fmt.Println("  ┌─────────────┐                    ┌─────────────┐")
	fmt.Printf("  │ %s %-7s %s │                    │ %s %-7s %s │\n",
		cardData[7].symbol, truncate(cardData[7].name, 5), cardData[7].orientIcon,
		cardData[8].symbol, truncate(cardData[8].name, 5), cardData[8].orientIcon)
	fmt.Println("  │ 8. External │                    │ 9. Hopes/   │")
	fmt.Println("  │             │                    │    Fears    │")
	fmt.Println("  └─────────────┘                    └─────────────┘")
	fmt.Println("                    ┌─────────────────────────┐")
	fmt.Printf("                    │ %s %-19s %s │\n", cardData[9].symbol, truncate(cardData[9].name, 17), cardData[9].orientIcon)
	fmt.Println("                    │ 10. Final Outcome       │")
	fmt.Println("                    └─────────────────────────┘")
	fmt.Println()
	fmt.Println("┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓")
	fmt.Println("┃                    ✦ Detailed Card Meanings ✦              ┃")
	fmt.Println("┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛")
	fmt.Println()

	// Display detailed card information with elegant formatting
	contentWidth := 63 // Width of content inside the box (excluding borders)
	for i, card := range cardData {
		// Top border
		fmt.Println("┌" + strings.Repeat("─", contentWidth) + "┐")

		// Position header
		posText := fmt.Sprintf("%d. %s", i+1, card.position)
		fmt.Printf("│ %-61s │\n", posText)

		// Divider
		fmt.Println("├" + strings.Repeat("─", contentWidth) + "┤")

		// Card name with symbol - centered
		nameDisplay := fmt.Sprintf("%s %s %s", card.symbol, card.name, card.orientIcon)
		namePadding := (contentWidth - len(nameDisplay)) / 2
		if namePadding < 0 {
			namePadding = 0
		}
		leftPad := namePadding
		rightPad := contentWidth - len(nameDisplay) - leftPad
		fmt.Printf("│%s%s%s│\n",
			strings.Repeat(" ", leftPad), nameDisplay, strings.Repeat(" ", rightPad))

		// Orientation badge - centered
		orientText := "▲ UPRIGHT"
		if card.orientation == "reversed" {
			orientText = "↻ REVERSED"
		}
		orientPadding := (contentWidth - len(orientText)) / 2
		leftPad = orientPadding
		rightPad = contentWidth - len(orientText) - leftPad
		fmt.Printf("│%s%s%s│\n",
			strings.Repeat(" ", leftPad), orientText, strings.Repeat(" ", rightPad))

		// Divider
		fmt.Println("├" + strings.Repeat("─", contentWidth) + "┤")

		// Wrap meaning with proper justification
		words := strings.Fields(card.meaning)
		currentLine := ""
		for _, word := range words {
			testLine := currentLine
			if testLine != "" {
				testLine += " "
			}
			testLine += word
			if len(testLine) <= contentWidth {
				currentLine = testLine
			} else {
				if currentLine != "" {
					// Center justify the meaning text
					linePadding := (contentWidth - len(currentLine)) / 2
					leftPad = linePadding
					rightPad = contentWidth - len(currentLine) - leftPad
					fmt.Printf("│%s%s%s│\n",
						strings.Repeat(" ", leftPad), currentLine, strings.Repeat(" ", rightPad))
				}
				currentLine = word
			}
		}
		if currentLine != "" {
			linePadding := (contentWidth - len(currentLine)) / 2
			leftPad = linePadding
			rightPad = contentWidth - len(currentLine) - leftPad
			fmt.Printf("│%s%s%s│\n",
				strings.Repeat(" ", leftPad), currentLine, strings.Repeat(" ", rightPad))
		}

		// Bottom border
		fmt.Println("└" + strings.Repeat("─", contentWidth) + "┘")
		if i < len(cardData)-1 {
			fmt.Println()
		}
	}
}

// truncate truncates a string to max length
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// formatTarotReadingAsString formats tarot reading as a string for AI prompts
func formatTarotReadingAsString(tarotData map[string]interface{}) string {
	var output strings.Builder
	spreadName, _ := tarotData["spread_name"].(string)
	spreadType, _ := tarotData["spread_type"].(string)
	cards, ok := tarotData["cards"].([]interface{})
	if !ok {
		// Fallback to JSON if structure is unexpected
		jsonData, _ := json.MarshalIndent(tarotData, "", "  ")
		return string(jsonData)
	}

	// Output clean, structured format optimized for AI parsing
	output.WriteString("TAROT READING\n")
	output.WriteString(fmt.Sprintf("Spread: %s (%s)\n", spreadName, spreadType))
	output.WriteString(fmt.Sprintf("Cards Drawn: %d\n\n", len(cards)))

	for i, card := range cards {
		cardMap, ok := card.(map[string]interface{})
		if !ok {
			continue
		}

		position := fmt.Sprintf("%v", cardMap["position"])
		// Remove leading number if present
		posParts := strings.SplitN(position, ". ", 2)
		if len(posParts) > 1 {
			position = posParts[1]
		}

		cardName := fmt.Sprintf("%v", cardMap["card_name"])
		meaning := fmt.Sprintf("%v", cardMap["card_meaning"])
		orientation := "upright"
		if orient, ok := cardMap["orientation"].(string); ok {
			orientation = orient
		}

		// Title case orientation
		orientationTitle := orientation
		if len(orientation) > 0 {
			orientationTitle = strings.ToUpper(orientation[:1]) + strings.ToLower(orientation[1:])
		}

		// Clean, structured output format
		output.WriteString(fmt.Sprintf("CARD %d\n", i+1))
		output.WriteString(fmt.Sprintf("Position: %s\n", position))
		output.WriteString(fmt.Sprintf("Card: %s\n", cardName))
		output.WriteString(fmt.Sprintf("Orientation: %s\n", orientationTitle))
		output.WriteString(fmt.Sprintf("Meaning: %s\n", meaning))
		output.WriteString("\n")
	}

	return output.String()
}

// outputParsedTarotReading outputs tarot reading in clean, structured format for AI interpretation
func outputParsedTarotReading(tarotData map[string]interface{}) {
	fmt.Print(formatTarotReadingAsString(tarotData))
}

// displayCardList is a fallback simple list display
func displayCardList(cards []interface{}, metadata map[string]TarotCardMetadata) {
	for i, card := range cards {
		cardMap, ok := card.(map[string]interface{})
		if !ok {
			continue
		}
		posStr := fmt.Sprintf("%v", cardMap["position"])
		posParts := strings.SplitN(posStr, ". ", 2)
		position := posStr
		if len(posParts) > 1 {
			position = posParts[1]
		}
		name := fmt.Sprintf("%v", cardMap["card_name"])
		meaning := fmt.Sprintf("%v", cardMap["card_meaning"])
		orientation := "upright"
		if orient, ok := cardMap["orientation"].(string); ok {
			orientation = orient
		}

		symbol, orientIcon := getCardDecoration(name, orientation, metadata)

		fmt.Printf("\n%d. %s\n", i+1, position)
		fmt.Printf("   %s %s %s\n", symbol, name, orientIcon)
		if orientation == "reversed" {
			fmt.Printf("   ↻ Reversed\n")
		} else {
			fmt.Printf("   ▲ Upright\n")
		}
		fmt.Printf("   Meaning: %s\n", meaning)
	}
}

// loadS3Config loads S3 configuration from ~/.celeste.cfg file
func loadS3Config() (*S3Config, error) {
	config := &S3Config{
		Endpoint:   "https://sfo3.digitaloceanspaces.com",
		Region:     "sfo3",
		BucketName: "whykusanagi",
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %v", err)
	}

	celesteConfigPath := filepath.Join(homeDir, ".celeste.cfg")
	if _, err := os.Stat(celesteConfigPath); err == nil {
		data, err := os.ReadFile(celesteConfigPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read ~/.celeste.cfg: %v", err)
		}

		lines := bytes.Split(data, []byte("\n"))
		for _, line := range lines {
			parts := bytes.SplitN(line, []byte("="), 2)
			if len(parts) != 2 {
				continue
			}
			key := string(bytes.TrimSpace(parts[0]))
			val := string(bytes.TrimSpace(parts[1]))

			switch key {
			case "access_key_id":
				config.AccessKeyID = val
			case "secret_access_key":
				config.SecretAccessKey = val
			case "endpoint":
				config.Endpoint = val
			case "region":
				config.Region = val
			case "bucket_name":
				config.BucketName = val
			}
		}
	}

	if config.AccessKeyID == "" {
		config.AccessKeyID = os.Getenv("DO_SPACES_ACCESS_KEY_ID")
	}
	if config.SecretAccessKey == "" {
		config.SecretAccessKey = os.Getenv("DO_SPACES_SECRET_ACCESS_KEY")
	}

	if config.AccessKeyID == "" || config.SecretAccessKey == "" {
		return nil, fmt.Errorf("missing DigitalOcean Spaces credentials. Set in ~/.celeste.cfg or environment variables (DO_SPACES_ACCESS_KEY_ID, DO_SPACES_SECRET_ACCESS_KEY)")
	}

	return config, nil
}

// createS3Session creates an AWS S3 session for DigitalOcean Spaces
func createS3Session(config *S3Config) (*s3.S3, error) {
	sess, err := session.NewSession(&aws.Config{
		Endpoint:    aws.String(config.Endpoint),
		Region:      aws.String(config.Region),
		Credentials: credentials.NewStaticCredentials(config.AccessKeyID, config.SecretAccessKey, ""),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create S3 session: %v", err)
	}

	return s3.New(sess), nil
}

// uploadConversationToS3 uploads a conversation entry to DigitalOcean Spaces
func uploadConversationToS3(entry *ConversationEntry) error {
	config, err := loadS3Config()
	if err != nil {
		return fmt.Errorf("failed to load S3 config: %v", err)
	}

	s3Client, err := createS3Session(config)
	if err != nil {
		return fmt.Errorf("failed to create S3 session: %v", err)
	}

	// Convert entry to JSON
	jsonData, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal conversation entry: %v", err)
	}

	// Create S3 key with timestamp
	key := fmt.Sprintf("celeste/conversations/%s.json", entry.ID)

	// Upload to S3
	_, err = s3Client.PutObject(&s3.PutObjectInput{
		Bucket:      aws.String(config.BucketName),
		Key:         aws.String(key),
		Body:        bytes.NewReader(jsonData),
		ContentType: aws.String("application/json"),
	})

	if err != nil {
		return fmt.Errorf("failed to upload to S3: %v", err)
	}

	fmt.Fprintf(os.Stderr, "✅ Conversation uploaded to S3: %s\n", key)
	return nil
}

// createConversationEntry creates a conversation entry from the current request
func createConversationEntry(format, platform, topic, tone, persona, prompt, response string, result map[string]interface{}) *ConversationEntry {
	// Get user ID from environment or default to kusanagi
	userID := os.Getenv("CELESTE_USER_ID")
	if userID == "" {
		userID = "kusanagi" // Default user ID
	}

	entry := &ConversationEntry{
		ID:          fmt.Sprintf("%d", time.Now().UnixNano()),
		Timestamp:   time.Now(),
		UserID:      userID,
		ContentType: format,
		Tone:        tone,
		Game:        topic, // Keep Game field for backward compatibility, but use topic
		Persona:     persona,
		Prompt:      prompt,
		Response:    response,
		TokensUsed:  make(map[string]interface{}),
		Metadata:    make(map[string]interface{}),
		Success:     true,
	}

	if usage, ok := result["usage"].(map[string]interface{}); ok {
		entry.TokensUsed = usage
	}

	entry.Metadata["command_line"] = strings.Join(os.Args, " ")
	entry.Metadata["api_endpoint"] = os.Getenv("CELESTE_API_ENDPOINT")

	// Add bot integration metadata
	entry.Metadata["platform"] = os.Getenv("CELESTE_PLATFORM") // discord, twitch, cli
	entry.Metadata["channel_id"] = os.Getenv("CELESTE_CHANNEL_ID")
	entry.Metadata["guild_id"] = os.Getenv("CELESTE_GUILD_ID")
	entry.Metadata["message_id"] = os.Getenv("CELESTE_MESSAGE_ID")
	entry.Metadata["pgp_signature"] = os.Getenv("CELESTE_PGP_SIGNATURE")
	entry.Metadata["override_enabled"] = os.Getenv("CELESTE_OVERRIDE_ENABLED") == "true"

	// Enhanced fields for OpenSearch RAG
	entry.Intent = determineIntent(format, platform)
	entry.Purpose = format
	entry.Platform = determinePlatformFromFlags(platform, format)
	entry.Sentiment = determineSentiment(tone)
	entry.Topics = extractTopics(topic, prompt, response)
	entry.Tags = generateTags(format, topic, tone, persona)
	entry.Context = fmt.Sprintf("Format: %s, Platform: %s, Topic: %s, Tone: %s, Persona: %s", format, platform, topic, tone, persona)

	return entry
}

// Helper functions for RAG
func determineIntent(format, platform string) string {
	// Use platform if available, otherwise infer from format
	if platform != "" {
		switch platform {
		case "twitter", "tiktok":
			return "social_media"
		case "youtube":
			return "content_creation"
		case "discord":
			return "community_management"
		}
	}

	// Fallback to format-based intent
	switch format {
	case "short":
		return "social_media"
	case "long":
		return "content_creation"
	default:
		return "general"
	}
}

func determinePlatformFromFlags(platform, format string) string {
	if platform != "" {
		return platform
	}

	// Infer from format if platform not specified
	switch format {
	case "short":
		return "twitter" // Default for short format
	case "long":
		return "youtube" // Default for long format
	default:
		return "general"
	}
}

func determineSentiment(tone string) string {
	switch strings.ToLower(tone) {
	case "lewd", "explicit", "suggestive":
		return "playful"
	case "teasing", "chaotic", "funny":
		return "positive"
	case "dramatic", "parody":
		return "neutral"
	case "cute", "sweet":
		return "positive"
	case "official":
		return "neutral"
	default:
		return "neutral"
	}
}

func extractTopics(topic, prompt, response string) []string {
	topics := []string{}
	if topic != "" {
		topics = append(topics, strings.ToLower(topic))
	}
	// Simple keyword extraction - could be enhanced
	keywords := []string{"celeste", "kusanagi", "vtuber", "stream", "game", "anime"}
	for _, keyword := range keywords {
		if strings.Contains(strings.ToLower(response), keyword) {
			topics = append(topics, keyword)
		}
	}
	return topics
}

func generateTags(format, topic, tone, persona string) []string {
	tags := []string{"celeste", "ai", "content", "format:" + strings.ToLower(format)}
	if topic != "" {
		tags = append(tags, "topic:"+strings.ToLower(topic))
	}
	if tone != "" {
		tags = append(tags, "tone:"+strings.ToLower(tone))
	}
	if persona != "" {
		tags = append(tags, "persona:"+strings.ToLower(persona))
	}
	return tags
}

// loadVeniceConfig loads Venice.ai configuration from ~/.celesteAI file
func loadVeniceConfig() (*VeniceConfig, error) {
	config := &VeniceConfig{
		BaseURL:  "https://api.venice.ai/api/v1",
		Model:    "venice-uncensored",
		Upscaler: "upscaler",
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %v", err)
	}

	celesteConfigPath := filepath.Join(homeDir, ".celesteAI")
	if _, err := os.Stat(celesteConfigPath); err == nil {
		data, err := os.ReadFile(celesteConfigPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read ~/.celesteAI: %v", err)
		}

		lines := bytes.Split(data, []byte("\n"))
		for _, line := range lines {
			parts := bytes.SplitN(line, []byte("="), 2)
			if len(parts) != 2 {
				continue
			}
			key := string(bytes.TrimSpace(parts[0]))
			val := string(bytes.TrimSpace(parts[1]))

			switch key {
			case "venice_api_key":
				config.APIKey = val
			case "venice_base_url":
				config.BaseURL = val
			case "venice_model":
				config.Model = val
			case "venice_upscaler":
				config.Upscaler = val
			}
		}
	}

	if config.APIKey == "" {
		config.APIKey = os.Getenv("VENICE_API_KEY")
	}

	if config.APIKey == "" {
		return nil, fmt.Errorf("missing Venice.ai API key. Set VENICE_API_KEY environment variable or venice_api_key in ~/.celesteAI")
	}

	return config, nil
}

// listVeniceModels lists available Venice.ai models
func listVeniceModels(config *VeniceConfig) ([]VeniceModel, error) {
	url := config.BaseURL + "/models"

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+config.APIKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("venice.ai request failed: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("Venice.ai API error: %s", string(body))
	}

	var response VeniceModelsResponse
	if err := json.Unmarshal(body, &response); err != nil {
		// If the response is not in the expected format, try parsing as a direct array
		var models []VeniceModel
		if err2 := json.Unmarshal(body, &models); err2 != nil {
			return nil, fmt.Errorf("failed to parse models response: %v (also tried direct array: %v)", err, err2)
		}
		return models, nil
	}

	return response.Models, nil
}

// makeVeniceEditRequest makes a request to Venice.ai for image editing/inpainting
func makeVeniceEditRequest(imagePath, prompt string, config *VeniceConfig) ([]byte, error) {
	url := config.BaseURL + "/image/edit"

	// Read the image file and convert to base64
	imageData, err := os.ReadFile(imagePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read image file: %v", err)
	}

	// Convert to base64
	imageBase64 := base64.StdEncoding.EncodeToString(imageData)

	// Create a proper JSON structure for editing
	requestData := map[string]interface{}{
		"image":  imageBase64,
		"prompt": prompt,
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(requestData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal JSON: %v", err)
	}

	payload := string(jsonData)

	req, err := http.NewRequest("POST", url, strings.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+config.APIKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Venice.ai edit request failed: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("Venice.ai edit API error: %s", string(body))
	}

	// The edit endpoint returns the image file directly
	return body, nil
}

// makeVeniceRequest makes a request to Venice.ai API with optional streaming
func makeVeniceRequest(prompt string, config *VeniceConfig, enableStream bool, noAnimation bool) (string, error) {
	clientConfig := openai.DefaultConfig(config.APIKey)
	clientConfig.BaseURL = config.BaseURL
	client := openai.NewClientWithConfig(clientConfig)

	ctx := context.Background()

	// Start corruption animation by default (during wait period)
	var animationDone chan bool
	var animationCtx context.Context
	var cancelAnimation context.CancelFunc
	if !noAnimation && shouldShowAnimation() {
		animationCtx, cancelAnimation = context.WithCancel(context.Background())
		animationDone = make(chan bool)
		startCorruptionAnimation(animationCtx, animationDone, os.Stderr)
		defer cancelAnimation()
	}

	// Use streaming if enabled
	if enableStream {
		stream, err := client.CreateChatCompletionStream(
			ctx,
			openai.ChatCompletionRequest{
				Model: config.Model,
				Messages: []openai.ChatCompletionMessage{
					{
						Role:    openai.ChatMessageRoleUser,
						Content: prompt,
					},
				},
				Stream: true,
			},
		)
		if err != nil {
			// Stop animation on error
			if animationDone != nil {
				cancelAnimation()
				<-animationDone
				fmt.Fprintf(os.Stderr, "\r\033[K")
			}
			return "", fmt.Errorf("Venice.ai API error: %v", err)
		}
		defer stream.Close()

		var fullResponse strings.Builder
		firstToken := true
		for {
			response, err := stream.Recv()
			if err != nil {
				if err == io.EOF {
					break
				}
				// Stop animation on error
				if animationDone != nil {
					cancelAnimation()
					<-animationDone
					fmt.Fprintf(os.Stderr, "\r\033[K")
				}
				return "", fmt.Errorf("Venice.ai streaming error: %v", err)
			}

			if len(response.Choices) > 0 {
				delta := response.Choices[0].Delta.Content
				if delta != "" {
					// Stop animation when first token arrives
					if firstToken && animationDone != nil {
						cancelAnimation()
						<-animationDone
						fmt.Fprintf(os.Stderr, "\r\033[K")
						firstToken = false
					}
					// Print token normally without corruption
					fmt.Print(delta)
					fullResponse.WriteString(delta)
				}
			}
		}
		fmt.Println() // New line after streaming
		return fullResponse.String(), nil
	}

	// Stop animation before non-streaming request
	if animationDone != nil {
		cancelAnimation()
		<-animationDone
		fmt.Fprintf(os.Stderr, "\r\033[K")
	}

	// Non-streaming fallback
	resp, err := client.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model: config.Model,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleUser,
					Content: prompt,
				},
			},
		},
	)

	if err != nil {
		return "", fmt.Errorf("Venice.ai API error: %v", err)
	}

	return resp.Choices[0].Message.Content, nil
}

// makeVeniceImageRequest makes a request to Venice.ai for image generation
func makeVeniceImageRequest(prompt string, config *VeniceConfig) (string, error) {
	url := config.BaseURL + "/image/generate"

	// Create a proper JSON structure
	requestData := map[string]interface{}{
		"cfg_scale":           7.5,
		"embed_exif_metadata": false,
		"format":              "webp",
		"height":              1024,
		"hide_watermark":      false,
		"model":               config.Model,
		"negative_prompt":     "blurry, low quality, distorted",
		"prompt":              prompt,
		"return_binary":       false,
		"variants":            1,
		"safe_mode":           false,
		"steps":               20,
		"width":               1024,
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(requestData)
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON: %v", err)
	}

	payload := string(jsonData)

	req, err := http.NewRequest("POST", url, strings.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+config.APIKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("Venice.ai request failed: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %v", err)
	}

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("Venice.ai API error: %s", string(body))
	}

	// Parse the response to extract image URL
	// The response should contain image URLs in the response
	return string(body), nil
}

// makeVeniceUpscaleRequest makes a request to Venice.ai for image upscaling
func makeVeniceUpscaleRequest(imagePath string, config *VeniceConfig, enhanceCreativity, replication float64, enhancePrompt string) ([]byte, error) {
	url := config.BaseURL + "/image/upscale"

	// Read the image file and convert to base64
	imageData, err := os.ReadFile(imagePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read image file: %v", err)
	}

	// Convert to base64
	imageBase64 := base64.StdEncoding.EncodeToString(imageData)

	// Create a proper JSON structure for upscaling
	requestData := map[string]interface{}{
		"enhance":           true,
		"enhanceCreativity": enhanceCreativity,
		"enhancePrompt":     enhancePrompt,
		"replication":       replication,
		"image":             imageBase64,
		"scale":             2,
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(requestData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal JSON: %v", err)
	}

	payload := string(jsonData)

	req, err := http.NewRequest("POST", url, strings.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+config.APIKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Venice.ai upscale request failed: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("Venice.ai upscale API error: %s", string(body))
	}

	// The upscale endpoint returns the image file directly
	return body, nil
}

// generateFilename generates a filename for saved images
func generateFilename(prefix string, isUpscaled bool) string {
	timestamp := time.Now().Format("2006-01-02_15-04-05")

	if isUpscaled {
		return fmt.Sprintf("%s_upscaled_%s.png", prefix, timestamp)
	}
	return fmt.Sprintf("%s_%s.png", prefix, timestamp)
}

// saveImageData saves image data to a file
func saveImageData(imageData []byte, filename string) error {
	// Create the file
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %v", filename, err)
	}
	defer file.Close()

	// Write the image data
	_, err = file.Write(imageData)
	if err != nil {
		return fmt.Errorf("failed to write image data: %v", err)
	}

	return nil
}

// extractImageFromResponse extracts image data from Venice.ai response
func extractImageFromResponse(response string) ([]byte, error) {
	// Try to parse as JSON first to extract image URL or data
	var responseData map[string]interface{}
	if err := json.Unmarshal([]byte(response), &responseData); err == nil {
		fmt.Fprintf(os.Stderr, "Debug: Parsed JSON structure: %+v\n", responseData)

		// Check for nested data structure (Venice.ai format)
		if data, ok := responseData["data"].(map[string]interface{}); ok {
			fmt.Fprintf(os.Stderr, "Debug: Found data object: %+v\n", data)

			// Check for image URL in data
			if imageURL, ok := data["image_url"].(string); ok {
				fmt.Fprintf(os.Stderr, "Debug: Found image URL: %s\n", imageURL)
				// Download image from URL
				resp, err := http.Get(imageURL)
				if err != nil {
					return nil, fmt.Errorf("failed to download image: %v", err)
				}
				defer resp.Body.Close()

				return io.ReadAll(resp.Body)
			}

			// Check for base64 image data in data
			if imageData, ok := data["image_data"].(string); ok {
				fmt.Fprintf(os.Stderr, "Debug: Found image_data field\n")
				return base64.StdEncoding.DecodeString(imageData)
			}

			// Check for any field that might contain image data
			for key, value := range data {
				if str, ok := value.(string); ok && len(str) > 100 {
					fmt.Fprintf(os.Stderr, "Debug: Found potential image data in field '%s' (length: %d)\n", key, len(str))
					// Try to decode as base64
					if decoded, err := base64.StdEncoding.DecodeString(str); err == nil {
						return decoded, nil
					}
				}
			}
		}

		// Check for direct image URL
		if imageURL, ok := responseData["image_url"].(string); ok {
			fmt.Fprintf(os.Stderr, "Debug: Found direct image URL: %s\n", imageURL)
			// Download image from URL
			resp, err := http.Get(imageURL)
			if err != nil {
				return nil, fmt.Errorf("failed to download image: %v", err)
			}
			defer resp.Body.Close()

			return io.ReadAll(resp.Body)
		}

		// Check for direct base64 image data
		if imageData, ok := responseData["image_data"].(string); ok {
			fmt.Fprintf(os.Stderr, "Debug: Found direct image_data field\n")
			return base64.StdEncoding.DecodeString(imageData)
		}

		// Check for other possible fields
		if data, ok := responseData["data"].(string); ok {
			fmt.Fprintf(os.Stderr, "Debug: Found data as string\n")
			return base64.StdEncoding.DecodeString(data)
		}
		if result, ok := responseData["result"].(string); ok {
			fmt.Fprintf(os.Stderr, "Debug: Found result field\n")
			return base64.StdEncoding.DecodeString(result)
		}

		// Check for image field
		if image, ok := responseData["image"].(string); ok {
			fmt.Fprintf(os.Stderr, "Debug: Found image field\n")
			return base64.StdEncoding.DecodeString(image)
		}

		// Check for output field
		if output, ok := responseData["output"].(string); ok {
			fmt.Fprintf(os.Stderr, "Debug: Found output field\n")
			return base64.StdEncoding.DecodeString(output)
		}

		// Check for files field (array of files)
		if files, ok := responseData["files"].([]interface{}); ok {
			fmt.Fprintf(os.Stderr, "Debug: Found files array with %d items\n", len(files))
			if len(files) > 0 {
				if file, ok := files[0].(map[string]interface{}); ok {
					if url, ok := file["url"].(string); ok {
						fmt.Fprintf(os.Stderr, "Debug: Found file URL: %s\n", url)
						resp, err := http.Get(url)
						if err != nil {
							return nil, fmt.Errorf("failed to download image: %v", err)
						}
						defer resp.Body.Close()
						return io.ReadAll(resp.Body)
					}
				}
			}
		}

		// Check for images field (array of images)
		if images, ok := responseData["images"].([]interface{}); ok {
			fmt.Fprintf(os.Stderr, "Debug: Found images array with %d items\n", len(images))
			if len(images) > 0 {
				if image, ok := images[0].(string); ok {
					fmt.Fprintf(os.Stderr, "Debug: Found image as string, attempting base64 decode\n")
					return base64.StdEncoding.DecodeString(image)
				}
			}
		}

		// Check for any field that might contain image data
		for key, value := range responseData {
			if str, ok := value.(string); ok && len(str) > 100 {
				fmt.Fprintf(os.Stderr, "Debug: Found potential image data in field '%s' (length: %d)\n", key, len(str))
				// Try to decode as base64
				if decoded, err := base64.StdEncoding.DecodeString(str); err == nil {
					return decoded, nil
				}
			}
		}
	}

	// If not JSON or no image data found, assume the response is base64 encoded image data
	fmt.Fprintf(os.Stderr, "Debug: Attempting to decode entire response as base64\n")
	return base64.StdEncoding.DecodeString(response)
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// verifyPGPSignature verifies a PGP signature for override commands
func verifyPGPSignature(message, signature string) bool {
	// This is a placeholder for PGP signature verification
	// In a real implementation, you would use a PGP library like golang.org/x/crypto/openpgp
	// For now, we'll implement a simple check for demonstration

	if signature == "" {
		return false
	}

	// Simple validation - in production, use proper PGP verification
	// This checks if the signature contains expected patterns for Kusanagi's key
	expectedPatterns := []string{
		"kusanagi", "abyss", "celeste", "override",
	}

	for _, pattern := range expectedPatterns {
		if strings.Contains(strings.ToLower(signature), pattern) {
			return true
		}
	}

	return false
}

// checkOverridePermissions checks if the user has override permissions
func checkOverridePermissions() bool {
	overrideEnabled := os.Getenv("CELESTE_OVERRIDE_ENABLED") == "true"
	pgpSignature := os.Getenv("CELESTE_PGP_SIGNATURE")

	if !overrideEnabled {
		return false
	}

	// Verify PGP signature if provided
	if pgpSignature != "" {
		message := strings.Join(os.Args, " ")
		return verifyPGPSignature(message, pgpSignature)
	}

	// If no PGP signature required, just check if override is enabled
	return true
}

// getImageDimensions gets the dimensions of an image file
func getImageDimensions(imagePath string) (int, int, error) {
	// This is a simple implementation that assumes the image is a PNG
	// In a production environment, you'd want to use a proper image library
	// For now, we'll use the file command to get dimensions
	cmd := fmt.Sprintf("file \"%s\"", imagePath)
	output, err := runCommand(cmd)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get image dimensions: %v", err)
	}

	// Parse the output to extract dimensions
	// Example output: "image.png: PNG image data, 800 x 800, 8-bit/color RGBA, non-interlaced"
	parts := strings.Split(output, ",")
	if len(parts) < 2 {
		return 0, 0, fmt.Errorf("could not parse image dimensions from: %s", output)
	}

	dimensionPart := strings.TrimSpace(parts[1])
	// Extract "800 x 800" part
	dimensionMatch := strings.Split(dimensionPart, " ")
	if len(dimensionMatch) < 3 {
		return 0, 0, fmt.Errorf("could not parse dimensions from: %s", dimensionPart)
	}

	width, err := strconv.Atoi(dimensionMatch[0])
	if err != nil {
		return 0, 0, fmt.Errorf("could not parse width: %v", err)
	}

	height, err := strconv.Atoi(dimensionMatch[2])
	if err != nil {
		return 0, 0, fmt.Errorf("could not parse height: %v", err)
	}

	return width, height, nil
}

// runCommand executes a shell command and returns the output
func runCommand(command string) (string, error) {
	cmd := exec.Command("sh", "-c", command)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func main() {
	// Command-line flags
	var format, platform, topic, tone, media, request string
	var debug, sync, nsfw bool
	var contextExtra string
	var persona string
	var tarotMode bool
	var spreadType string

	flag.StringVar(&format, "format", "short", "Content format: short (280 chars), long (5000 chars), or general (flexible)")
	flag.StringVar(&platform, "platform", "", "Platform: twitter, tiktok, youtube, or discord (optional)")
	flag.StringVar(&topic, "topic", "", "Topic or subject (game name, event, etc.)")
	flag.StringVar(&tone, "tone", "", "Tone or style for Celeste to use (lewd, teasing, etc.)")
	flag.StringVar(&media, "media", "", "Optional media reference (image/GIF URL)")
	flag.StringVar(&contextExtra, "context", "", "Additional background context for Celeste to include")
	flag.StringVar(&request, "request", "", "Direct instructions or specific requirements for the content")
	flag.StringVar(&persona, "persona", "celeste_stream", "Persona to use (celeste_stream, celeste_ad_read, celeste_moderation_warning)")
	flag.BoolVar(&tarotMode, "tarot", false, "Get a tarot reading from the DigitalOcean function")
	flag.StringVar(&spreadType, "spread", "three", "Type of tarot spread: 'celtic' or 'three' (default: three)")
	var parsedOutput bool
	flag.BoolVar(&parsedOutput, "parsed", false, "Output tarot reading in clean parsed format for AI interpretation")
	var divineMode bool
	flag.BoolVar(&divineMode, "divine", false, "Get tarot reading and automatically interpret with AI (Digital Ocean)")
	var divineNSFW bool
	flag.BoolVar(&divineNSFW, "divine-nsfw", false, "Get tarot reading and automatically interpret with AI (Venice.ai NSFW)")
	var noScaffold bool
	flag.BoolVar(&noScaffold, "no-scaffold", false, "Disable default scaffolding prompt additions")
	var scaffoldOverride string
	flag.StringVar(&scaffoldOverride, "scaffold", "", "Explicit scaffold override text")
	flag.BoolVar(&debug, "debug", false, "Enable debug output (shows full JSON response)")
	flag.BoolVar(&sync, "sync", false, "Upload conversation to DigitalOcean Spaces after completion")
	flag.BoolVar(&nsfw, "nsfw", false, "Enable NSFW mode using Venice.ai (uncensored content generation)")
	var imageMode bool
	flag.BoolVar(&imageMode, "image", false, "Generate image using lustify-sdxl model (requires --nsfw)")
	var upscaleMode bool
	flag.BoolVar(&upscaleMode, "upscale", false, "Upscale an existing image (requires --nsfw)")
	var imagePath string
	flag.StringVar(&imagePath, "image-path", "", "Path to image file for upscaling (requires --upscale)")
	var outputFile string
	flag.StringVar(&outputFile, "output", "", "Output filename for generated/upscaled images (optional)")
	var listModels bool
	flag.BoolVar(&listModels, "list-models", false, "List available Venice.ai models")
	var modelOverride string
	flag.StringVar(&modelOverride, "model", "", "Override Venice.ai model (e.g., lustify-sdxl, wai-Illustrious)")
	var enhanceCreativity float64
	flag.Float64Var(&enhanceCreativity, "enhance-creativity", 0.1, "Enhancement creativity level (0.0-1.0, lower = more faithful to original)")
	var replication float64
	flag.Float64Var(&replication, "replication", 0.8, "Replication level to preserve original details (0.0-1.0, higher = more faithful)")
	var enhancePrompt string
	flag.StringVar(&enhancePrompt, "enhance-prompt", "preserve original details, maintain authenticity", "Enhancement prompt for upscaling")
	var editMode bool
	flag.BoolVar(&editMode, "edit", false, "Edit/inpaint an existing image (requires --nsfw)")
	var editPrompt string
	flag.StringVar(&editPrompt, "edit-prompt", "", "Prompt for image editing (e.g., 'remove the signature', 'change the background')")
	var preserveSize bool
	flag.BoolVar(&preserveSize, "preserve-size", false, "Automatically upscale edited image back to original dimensions (requires --edit)")
	var upscaleFirst bool
	flag.BoolVar(&upscaleFirst, "upscale-first", false, "Upscale to 1024x1024 first, then inpaint (prevents distortion, 2 API calls)")
	var noAnimation bool
	flag.BoolVar(&noAnimation, "no-animation", false, "Disable corruption animation and visual feedback")
	var enableStream bool
	flag.BoolVar(&enableStream, "stream", true, "Enable streaming responses (default: true)")

	flag.Usage = func() {
		fmt.Println("Usage of CelesteCLI:")
		fmt.Println("  --format     Content format: short (280 chars), long (5000 chars), or general (flexible)")
		fmt.Println("               Default: short")
		fmt.Println("  --platform  Platform: twitter, tiktok, youtube, or discord (optional)")
		fmt.Println("  --topic     Topic or subject (game name, event, etc.)")
		fmt.Println("  --request   Direct instructions or specific requirements for the content")
		fmt.Println("  --tone      Style or tone for Celeste's response:")
		fmt.Println("               Examples: lewd, teasing, chaotic, cute, official, dramatic, parody, funny, sweet")
		fmt.Println("  --persona   Persona to use (celeste_stream, celeste_ad_read, celeste_moderation_warning)")
		fmt.Println("  --media     Optional image/GIF URL for Celeste to react to or include in context")
		fmt.Println("  --context   Additional background context for Celeste to include")
		fmt.Println("  --no-scaffold Disable default scaffolding prompt additions")
		fmt.Println("  --scaffold  Explicit scaffold override text")
		fmt.Println("  --tarot     Get a tarot reading from the DigitalOcean function")
		fmt.Println("  --spread    Type of tarot spread: 'celtic' or 'three' (default: three)")
		fmt.Println("  --parsed    Output tarot reading in clean parsed format for AI interpretation")
		fmt.Println("  --divine    Get tarot reading and automatically interpret with AI (Digital Ocean)")
		fmt.Println("  --divine-nsfw Get tarot reading and automatically interpret with AI (Venice.ai NSFW)")
		fmt.Println("  --sync      Upload conversation to DigitalOcean Spaces after completion")
		fmt.Println("  --nsfw      Enable NSFW mode using Venice.ai (uncensored content generation)")
		fmt.Println("               Works with new format system: --nsfw --format short --platform twitter --topic \"NIKKE\" --tone \"explicit\"")
		fmt.Println("  --image     Generate image using lustify-sdxl model (requires --nsfw)")
		fmt.Println("  --upscale   Upscale an existing image (requires --nsfw)")
		fmt.Println("  --edit      Edit/inpaint an existing image (requires --nsfw)")
		fmt.Println("  --image-path Path to image file for upscaling/editing (requires --upscale or --edit)")
		fmt.Println("  --edit-prompt Prompt for image editing (e.g., 'remove the signature', 'change the background')")
		fmt.Println("  --preserve-size Automatically upscale edited image back to original dimensions (requires --edit)")
		fmt.Println("  --upscale-first Upscale to 1024x1024 first, then inpaint (prevents distortion, 2 API calls)")
		fmt.Println("  --output    Output filename for generated/upscaled/edited images (optional)")
		fmt.Println("  --list-models List available Venice.ai models")
		fmt.Println("  --model     Override Venice.ai model (e.g., lustify-sdxl, wai-Illustrious)")
		fmt.Println("  --enhance-creativity Enhancement creativity level (0.0-1.0, lower = more faithful)")
		fmt.Println("  --replication Replication level to preserve original details (0.0-1.0, higher = more faithful)")
		fmt.Println("  --enhance-prompt Enhancement prompt for upscaling")
		fmt.Println("  --no-animation Disable corruption animation and visual feedback")
		fmt.Println("  --stream    Enable streaming responses (default: true)")
		fmt.Println("  --debug     Show raw JSON output from API")
		fmt.Println()
		fmt.Println("Configuration:")
		fmt.Println("  ~/.celesteAI                - Celeste configuration file")
		fmt.Println("    endpoint                    - CelesteAI API endpoint")
		fmt.Println("    api_key                     - CelesteAI API key")
		fmt.Println("    tarot_function_url          - Tarot function URL (optional)")
		fmt.Println("    tarot_auth_token            - Tarot Basic Auth token")
		fmt.Println("    venice_api_key              - Venice.ai API key (for NSFW mode)")
		fmt.Println("  ~/.celeste.cfg              - DigitalOcean Spaces configuration")
		fmt.Println("  Environment Variables       - Fallback if config files not found")
		fmt.Println("    CELESTE_API_ENDPOINT        - CelesteAI API endpoint")
		fmt.Println("    CELESTE_API_KEY             - CelesteAI API key")
		fmt.Println("    TAROT_FUNCTION_URL          - Tarot function URL")
		fmt.Println("    TAROT_AUTH_TOKEN            - Tarot Basic Auth token")
		fmt.Println("    DO_SPACES_ACCESS_KEY_ID     - DigitalOcean Spaces access key")
		fmt.Println("    DO_SPACES_SECRET_ACCESS_KEY - DigitalOcean Spaces secret key")
		fmt.Println("    CELESTE_USER_ID             - User ID for conversation tracking")
		fmt.Println("    CELESTE_PLATFORM            - Platform (discord, twitch, cli)")
		fmt.Println("    CELESTE_OVERRIDE_ENABLED    - Enable override mode (true/false)")
		fmt.Println("    CELESTE_PGP_SIGNATURE       - PGP signature for override commands")
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  # Normal mode (Digital Ocean)")
		fmt.Println("  ./celestecli --format short --platform twitter --topic \"NIKKE\" --tone \"lewd\"")
		fmt.Println("  ./celestecli --format short --platform twitter --topic \"NIKKE\" --tone \"lewd\" --request \"write about Viper character\"")
		fmt.Println("  ./celestecli --format long --platform youtube --topic \"Streaming\" --request \"include links to website, socials, products\"")
		fmt.Println()
		fmt.Println("  # Tarot reading")
		fmt.Println("  ./celestecli --tarot")
		fmt.Println("  ./celestecli --tarot --spread celtic")
		fmt.Println("  ./celestecli --tarot --parsed  # Clean output for AI interpretation")
		fmt.Println("  ./celestecli --divine  # Get reading and interpret with AI")
		fmt.Println("  ./celestecli --divine-nsfw  # Get reading and interpret with NSFW AI")
		fmt.Println()
		fmt.Println("  # NSFW mode (Venice.ai) - text generation")
		fmt.Println("  ./celestecli --nsfw --format short --platform twitter --topic \"NIKKE\" --tone \"explicit\" --request \"write about character interactions\"")
		fmt.Println()
		fmt.Println("  # NSFW mode (Venice.ai) - media (unchanged)")
		fmt.Println("  ./celestecli --nsfw --image --request \"generate NSFW image of Celeste\"")
		fmt.Println("  ./celestecli --nsfw --upscale --image-path \"image.png\"")
		fmt.Println("  ./celestecli --nsfw --edit --image-path \"image.png\" --edit-prompt \"remove signature\"")
		fmt.Println("  ./celestecli --nsfw --list-models")
		fmt.Println("  ./celestecli --nsfw --model \"wai-Illustrious\" --image --request \"Generate anime-style image\"")
		fmt.Println("  ./celestecli --nsfw --upscale --image-path \"input.png\" --enhance-creativity 0.05 --replication 0.9")
		fmt.Println("  ./celestecli --nsfw --edit --image-path \"image.png\" --edit-prompt \"remove watermark\" --preserve-size")
		fmt.Println("  ./celestecli --nsfw --edit --image-path \"small_image.png\" --edit-prompt \"remove signature\" --upscale-first")
	}

	flag.Parse()

	// Handle divine modes (tarot + AI interpretation)
	if divineMode || divineNSFW {
		// Get tarot reading first
		tarotConfig, err := loadTarotConfig()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading tarot configuration: %v\n", err)
			os.Exit(1)
		}

		fmt.Fprintf(os.Stderr, "🔮 Fetching tarot reading...\n")

		// Start corruption animation if enabled
		var animationDone chan bool
		var cancelAnimation context.CancelFunc
		if !noAnimation && shouldShowAnimation() {
			animationCtx, cancel := context.WithCancel(context.Background())
			cancelAnimation = cancel
			animationDone = make(chan bool)
			startCorruptionAnimation(animationCtx, animationDone, os.Stderr)
		}

		// Make request in a goroutine so we can cancel animation on timeout
		var tarotData map[string]interface{}
		var requestErr error
		done := make(chan bool)

		go func() {
			tarotData, requestErr = makeTarotRequest(tarotConfig, spreadType)
			done <- true
		}()

		// Wait for request with timeout
		select {
		case <-done:
			// Request completed
		case <-time.After(11 * time.Second):
			// Timeout - cancel animation and return error
			if cancelAnimation != nil {
				cancelAnimation()
				<-animationDone
				fmt.Fprintf(os.Stderr, "\r\033[K") // Clear animation line
			}
			fmt.Fprintf(os.Stderr, "\n❌ Request timed out after 10 seconds\n")
			os.Exit(1)
		}

		// Stop animation
		if cancelAnimation != nil {
			cancelAnimation()
			<-animationDone
			fmt.Fprintf(os.Stderr, "\r\033[K") // Clear animation line
		}

		if requestErr != nil {
			fmt.Fprintf(os.Stderr, "Tarot request failed: %v\n", requestErr)
			os.Exit(1)
		}

		// Format tarot reading as string for AI prompt
		tarotReadingText := formatTarotReadingAsString(tarotData)

		// Build prompt for AI interpretation
		interpretationPrompt := fmt.Sprintf("Interpret this tarot reading for me:\n\n%s\n\nProvide a detailed interpretation of this reading, explaining what the cards reveal about the situation, their positions, and their meanings in context.", tarotReadingText)

		// Add NSFW instruction if using divine-nsfw
		if divineNSFW {
			interpretationPrompt += "\n\nProvide an explicit, uncensored, and detailed interpretation. Be direct and unfiltered in your analysis."
		}

		// Route to appropriate AI endpoint
		if divineNSFW {
			// Use Venice.ai for NSFW interpretation
			veniceConfig, err := loadVeniceConfig()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Venice.ai configuration error: %v\n", err)
				os.Exit(1)
			}

			fmt.Fprintf(os.Stderr, "✨ Interpreting tarot reading with NSFW AI...\n")
			_, err = makeVeniceRequest(interpretationPrompt, veniceConfig, enableStream, noAnimation)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Venice.ai interpretation failed: %v\n", err)
				os.Exit(1)
			}
			// Response is already printed by makeVeniceRequest
		} else {
			// Use Digital Ocean for standard interpretation
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
				fmt.Println("Missing CELESTE_API_ENDPOINT or CELESTE_API_KEY (env or ~/.celesteAI config file).")
				os.Exit(1)
			}

			fmt.Fprintf(os.Stderr, "✨ Interpreting tarot reading with AI...\n")

			// Build the request payload
			extraBody := make(map[string]interface{})
			messages := []Message{{Role: "user", Content: interpretationPrompt}}
			chatReq := ChatRequest{
				Model:     "celeste-ai",
				Messages:  messages,
				ExtraBody: extraBody,
			}

			body, err := json.Marshal(chatReq)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to encode request: %v\n", err)
				os.Exit(1)
			}

			// Declare variables for response handling
			var responseBody []byte
			var start time.Time
			var wasStreamed bool

			// Start corruption animation by default (during wait period)
			var aiAnimationDone chan bool
			var aiAnimationCtx context.Context
			var aiCancelAnimation context.CancelFunc
			if !noAnimation && shouldShowAnimation() {
				aiAnimationCtx, aiCancelAnimation = context.WithCancel(context.Background())
				aiAnimationDone = make(chan bool)
				startCorruptionAnimation(aiAnimationCtx, aiAnimationDone, os.Stderr)
				defer aiCancelAnimation()
			}

			// Try streaming if enabled
			if enableStream {
				// Add stream parameter to request
				var streamReq map[string]interface{}
				if err := json.Unmarshal(body, &streamReq); err == nil {
					streamReq["stream"] = true
					body, _ = json.Marshal(streamReq)
				}

				req, err := http.NewRequest("POST", endpoint+"chat/completions", bytes.NewBuffer(body))
				if err != nil {
					fmt.Fprintf(os.Stderr, "Failed to create request: %v\n", err)
					os.Exit(1)
				}
				req.Header.Set("Authorization", "Bearer "+apiKey)
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("Accept", "text/event-stream")

				start = time.Now()
				client := &http.Client{}
				resp, err := client.Do(req)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Request failed: %v\n", err)
					os.Exit(1)
				}
				defer resp.Body.Close()

				// Check if streaming is supported
				if resp.Header.Get("Content-Type") == "text/event-stream" || strings.Contains(resp.Header.Get("Content-Type"), "event-stream") {
					// Handle SSE streaming
					scanner := bufio.NewScanner(resp.Body)
					var fullResponse strings.Builder
					firstToken := true

					for scanner.Scan() {
						line := scanner.Text()
						if strings.HasPrefix(line, "data: ") {
							data := strings.TrimPrefix(line, "data: ")
							if data == "[DONE]" {
								break
							}

							var chunk map[string]interface{}
							if err := json.Unmarshal([]byte(data), &chunk); err == nil {
								if choices, ok := chunk["choices"].([]interface{}); ok && len(choices) > 0 {
									if choice, ok := choices[0].(map[string]interface{}); ok {
										if delta, ok := choice["delta"].(map[string]interface{}); ok {
											if content, ok := delta["content"].(string); ok && content != "" {
												// Stop animation when first token arrives
												if firstToken && aiAnimationDone != nil {
													aiCancelAnimation()
													<-aiAnimationDone
													fmt.Fprintf(os.Stderr, "\r\033[K")
													firstToken = false
												}
												// Print token normally without corruption
												fmt.Print(content)
												fullResponse.WriteString(content)
											}
										}
									}
								}
							}
						}
					}

					if err := scanner.Err(); err != nil {
						// Stop animation on error
						if aiAnimationDone != nil {
							aiCancelAnimation()
							<-aiAnimationDone
							fmt.Fprintf(os.Stderr, "\r\033[K")
						}
						fmt.Fprintf(os.Stderr, "\nStreaming error: %v\n", err)
						os.Exit(1)
					}

					fmt.Println() // New line after streaming
					elapsed := time.Since(start)
					fmt.Fprintf(os.Stderr, "✅ Interpretation completed in %s\n", elapsed)

					// Parse the full response for conversation entry
					result := map[string]interface{}{
						"choices": []interface{}{
							map[string]interface{}{
								"message": map[string]interface{}{
									"content": fullResponse.String(),
								},
							},
						},
					}
					responseBody, _ = json.Marshal(result)
					wasStreamed = true
				} else {
					// Streaming not supported, fall back to regular request
					if aiAnimationDone != nil {
						aiCancelAnimation()
						<-aiAnimationDone
						fmt.Fprintf(os.Stderr, "\r\033[K")
					}
					responseBody, err = io.ReadAll(resp.Body)
					if err != nil {
						fmt.Fprintf(os.Stderr, "Failed to read response: %v\n", err)
						os.Exit(1)
					}
					elapsed := time.Since(start)
					fmt.Fprintf(os.Stderr, "✅ Response received in %s\n", elapsed)
				}
			} else {
				// Non-streaming request
				fmt.Fprintln(os.Stderr, "⏳ Sending request to CelesteAI...")
				start = time.Now()
				req, err := http.NewRequest("POST", endpoint+"chat/completions", bytes.NewBuffer(body))
				if err != nil {
					fmt.Fprintf(os.Stderr, "Failed to create request: %v\n", err)
					os.Exit(1)
				}
				req.Header.Set("Authorization", "Bearer "+apiKey)
				req.Header.Set("Content-Type", "application/json")

				client := &http.Client{}
				resp, err := client.Do(req)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Request failed: %v\n", err)
					os.Exit(1)
				}
				defer resp.Body.Close()

				responseBody, err = io.ReadAll(resp.Body)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Failed to read response: %v\n", err)
					os.Exit(1)
				}

				elapsed := time.Since(start)
				fmt.Fprintf(os.Stderr, "✅ Response received in %s\n", elapsed)
			}

			// Stop animation if it was running
			if aiAnimationDone != nil {
				<-aiAnimationDone
				fmt.Fprintf(os.Stderr, "\r\033[K") // Clear animation line
			}

			if debug {
				fmt.Println(string(responseBody))
			} else {
				var result map[string]interface{}
				if err := json.Unmarshal(responseBody, &result); err != nil {
					fmt.Fprintf(os.Stderr, "Failed to parse response: %v\n", err)
					os.Exit(1)
				}

				if choices, ok := result["choices"].([]interface{}); ok && len(choices) > 0 {
					if choice, ok := choices[0].(map[string]interface{}); ok {
						if message, ok := choice["message"].(map[string]interface{}); ok {
							if content, ok := message["content"].(string); ok {
								// Only print if we didn't already stream it
								if !wasStreamed {
									fmt.Println(content)
								}
							} else {
								fmt.Fprintf(os.Stderr, "No content in response\n")
								os.Exit(1)
							}
						} else {
							fmt.Fprintf(os.Stderr, "No message in response\n")
							os.Exit(1)
						}
					} else {
						fmt.Fprintf(os.Stderr, "No choice in response\n")
						os.Exit(1)
					}
				} else {
					fmt.Fprintf(os.Stderr, "No choices in response\n")
					os.Exit(1)
				}
			}
		}
		return
	}

	// Handle tarot mode (separate from AI)
	if tarotMode {
		tarotConfig, err := loadTarotConfig()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading tarot configuration: %v\n", err)
			os.Exit(1)
		}

		fmt.Fprintf(os.Stderr, "🔮 Fetching tarot reading...\n")
		startTime := time.Now()

		// Start corruption animation if enabled
		var animationDone chan bool
		var cancelAnimation context.CancelFunc
		if !noAnimation && shouldShowAnimation() {
			animationCtx, cancel := context.WithCancel(context.Background())
			cancelAnimation = cancel
			animationDone = make(chan bool)
			startCorruptionAnimation(animationCtx, animationDone, os.Stderr)
		}

		// Make request in a goroutine so we can cancel animation on timeout
		var tarotData map[string]interface{}
		var requestErr error
		done := make(chan bool)

		go func() {
			tarotData, requestErr = makeTarotRequest(tarotConfig, spreadType)
			done <- true
		}()

		// Wait for request with timeout
		select {
		case <-done:
			// Request completed
		case <-time.After(11 * time.Second):
			// Timeout - cancel animation and return error
			if cancelAnimation != nil {
				cancelAnimation()
				<-animationDone
				fmt.Fprintf(os.Stderr, "\r\033[K") // Clear animation line
			}
			fmt.Fprintf(os.Stderr, "\n❌ Request timed out after 10 seconds\n")
			os.Exit(1)
		}

		// Stop animation
		if cancelAnimation != nil {
			cancelAnimation()
			<-animationDone
			fmt.Fprintf(os.Stderr, "\r\033[K") // Clear animation line
		}

		if requestErr != nil {
			fmt.Fprintf(os.Stderr, "Tarot request failed: %v\n", requestErr)
			os.Exit(1)
		}

		duration := time.Since(startTime)

		if parsedOutput {
			// Print parsed output for AI consumption
			outputParsedTarotReading(tarotData)
		} else if debug {
			// Print raw JSON
			jsonData, _ := json.MarshalIndent(tarotData, "", "  ")
			fmt.Println(string(jsonData))
		} else {
			// Print visual formatted output
			formatTarotReading(tarotData)
		}

		if !parsedOutput {
			fmt.Fprintf(os.Stderr, "\n✨ Tarot reading completed in %v\n", duration)
		}
		return
	}

	// Check for override permissions
	hasOverride := checkOverridePermissions()
	if hasOverride {
		fmt.Fprintln(os.Stderr, "🔓 Override mode enabled - Abyssal laws may be bypassed")
	}

	// Load personality configuration
	personalityConfig, err := loadPersonalityConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to load personality.yml: %v\n", err)
		personalityConfig = &PersonalityConfig{}
	}

	personality := getPersonalityPrompt(personalityConfig, persona)

	// Load scaffolding configuration
	scaffoldingConfig, err := loadScaffoldingConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to load scaffolding.json: %v\n", err)
		scaffoldingConfig = getDefaultScaffoldingConfig()
	}

	// Build scaffold text with overrides
	defaultScaffold := personality + getScaffoldPrompt(format, platform, topic, request, tone, contextExtra, scaffoldingConfig)
	scaffoldText := defaultScaffold
	if noScaffold {
		scaffoldText = ""
	}
	if scaffoldOverride != "" {
		scaffoldText = scaffoldOverride
	}
	scaffoldText = strings.TrimSpace(scaffoldText)

	var userPromptBuilder strings.Builder
	if contextExtra != "" {
		userPromptBuilder.WriteString(fmt.Sprintf("Context: %s", contextExtra))
	}
	if media != "" {
		if userPromptBuilder.Len() > 0 {
			userPromptBuilder.WriteString(" ")
		}
		userPromptBuilder.WriteString(fmt.Sprintf("React to this media: %s", media))
	}
	userPrompt := strings.TrimSpace(userPromptBuilder.String())

	promptParts := make([]string, 0, 2)
	if scaffoldText != "" {
		promptParts = append(promptParts, scaffoldText)
	}
	if userPrompt != "" {
		promptParts = append(promptParts, userPrompt)
	}

	prompt := strings.TrimSpace(strings.Join(promptParts, "\n\n"))
	if prompt == "" {
		fmt.Fprintln(os.Stderr, "Warning: prompt is empty after applying scaffolding options; provide --context or scaffolding text")
	}

	// Handle NSFW mode with Venice.ai
	if nsfw {
		veniceConfig, err := loadVeniceConfig()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Venice.ai configuration error: %v\n", err)
			os.Exit(1)
		}

		// Handle model override
		if modelOverride != "" {
			veniceConfig.Model = modelOverride
		}

		// Handle model listing
		if listModels {
			fmt.Fprintln(os.Stderr, "📋 Fetching available Venice.ai models...")
			startTime := time.Now()
			models, err := listVeniceModels(veniceConfig)
			duration := time.Since(startTime)

			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to list models: %v\n", err)
				os.Exit(1)
			}

			fmt.Printf("Available Venice.ai models (fetched in %v):\n", duration)
			for _, model := range models {
				fmt.Printf("  • %s (%s) - %s\n", model.ID, model.Type, model.Description)
			}
			return
		}

		if upscaleMode {
			if imagePath == "" {
				fmt.Fprintf(os.Stderr, "Error: --image-path is required for upscaling\n")
				os.Exit(1)
			}

			fmt.Fprintln(os.Stderr, "🔍 NSFW Upscale Mode: Using Venice.ai upscaler")

			// Start corruption animation if enabled
			var animationDone chan bool
			if !noAnimation && shouldShowAnimation() {
				animationCtx, cancel := context.WithCancel(context.Background())
				animationDone = make(chan bool)
				startCorruptionAnimation(animationCtx, animationDone, os.Stderr)
				defer cancel()
			}

			startTime := time.Now()
			imageData, err := makeVeniceUpscaleRequest(imagePath, veniceConfig, enhanceCreativity, replication, enhancePrompt)
			duration := time.Since(startTime)

			// Stop animation
			if animationDone != nil {
				<-animationDone
				fmt.Fprintf(os.Stderr, "\r\033[K") // Clear animation line
			}

			if err != nil {
				fmt.Fprintf(os.Stderr, "Venice.ai upscaling failed: %v\n", err)
				os.Exit(1)
			}

			// Generate filename
			filename := outputFile
			if filename == "" {
				baseName := strings.TrimSuffix(filepath.Base(imagePath), filepath.Ext(imagePath))
				filename = generateFilename(baseName, true)
			}

			// Save image
			if err := saveImageData(imageData, filename); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to save image: %v\n", err)
				os.Exit(1)
			}

			fmt.Printf("✅ Image upscaled and saved as '%s' (took %v)\n", filename, duration)

		} else if imageMode {
			fmt.Fprintln(os.Stderr, "🎨 NSFW Image Mode: Using lustify-sdxl for image generation")
			// Switch to image generation model
			veniceConfig.Model = "lustify-sdxl"

			// Start corruption animation if enabled
			var animationDone chan bool
			if !noAnimation && shouldShowAnimation() {
				animationCtx, cancel := context.WithCancel(context.Background())
				animationDone = make(chan bool)
				startCorruptionAnimation(animationCtx, animationDone, os.Stderr)
				defer cancel()
			}

			startTime := time.Now()
			response, err := makeVeniceImageRequest(prompt, veniceConfig)
			duration := time.Since(startTime)

			// Stop animation
			if animationDone != nil {
				<-animationDone
				fmt.Fprintf(os.Stderr, "\r\033[K") // Clear animation line
			}

			if err != nil {
				fmt.Fprintf(os.Stderr, "Venice.ai image generation failed: %v\n", err)
				os.Exit(1)
			}

			// Extract and save image data
			imageData, err := extractImageFromResponse(response)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to extract image data: %v\n", err)
				os.Exit(1)
			}

			// Generate filename
			filename := outputFile
			if filename == "" {
				filename = generateFilename("nsfw_image", false)
			}

			// Save image
			if err := saveImageData(imageData, filename); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to save image: %v\n", err)
				os.Exit(1)
			}

			fmt.Printf("✅ Image generated and saved as '%s' (took %v)\n", filename, duration)

		} else if editMode {
			if imagePath == "" {
				fmt.Fprintf(os.Stderr, "Error: --image-path is required for editing\n")
				os.Exit(1)
			}
			if editPrompt == "" {
				fmt.Fprintf(os.Stderr, "Error: --edit-prompt is required for editing\n")
				os.Exit(1)
			}

			// Start corruption animation if enabled
			var animationDone chan bool
			if !noAnimation && shouldShowAnimation() {
				animationCtx, cancel := context.WithCancel(context.Background())
				animationDone = make(chan bool)
				startCorruptionAnimation(animationCtx, animationDone, os.Stderr)
				defer cancel()
			}

			// Declare variables for the edit workflow
			var imageData []byte
			var duration time.Duration
			var err error

			// Handle upscale-first workflow
			if upscaleFirst {
				fmt.Fprintln(os.Stderr, "🔄 Upscale-First Mode: Upscaling image first to prevent distortion")

				// Get original dimensions
				originalWidth, originalHeight, err := getImageDimensions(imagePath)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Warning: Could not get original image dimensions: %v\n", err)
					originalWidth, originalHeight = 1024, 1024 // Default fallback
				}

				// Check if image is smaller than Venice.ai's output size
				if originalWidth < 1024 || originalHeight < 1024 {
					fmt.Fprintf(os.Stderr, "📏 Original: %dx%d - Upscaling to 1024x1024 for optimal inpainting\n", originalWidth, originalHeight)

					// Step 1: Upscale the original image to 1024x1024 (Venice.ai's inpainting size)
					fmt.Fprintln(os.Stderr, "🔍 Step 1: Upscaling to 1024x1024...")
					upscaledData, err := makeVeniceUpscaleRequest(imagePath, veniceConfig, 0.05, 0.9, "preserve original details exactly")
					if err != nil {
						fmt.Fprintf(os.Stderr, "Failed to upscale original image: %v\n", err)
						os.Exit(1)
					}

					// Save upscaled image temporarily
					tempUpscaledFile := "temp_upscaled_" + filepath.Base(imagePath)
					if err := saveImageData(upscaledData, tempUpscaledFile); err != nil {
						fmt.Fprintf(os.Stderr, "Failed to save upscaled image: %v\n", err)
						os.Exit(1)
					}
					defer os.Remove(tempUpscaledFile) // Clean up temp file

					// Step 2: Edit the upscaled image (should stay at 1024x1024)
					fmt.Fprintln(os.Stderr, "🎨 Step 2: Inpainting at 1024x1024 (no resizing)...")
					startTime := time.Now()
					imageData, err = makeVeniceEditRequest(tempUpscaledFile, editPrompt, veniceConfig)
					duration = time.Since(startTime)

					if err != nil {
						fmt.Fprintf(os.Stderr, "Venice.ai editing failed: %v\n", err)
						os.Exit(1)
					}

					fmt.Fprintf(os.Stderr, "✅ Optimized workflow completed (took %v) - 2 API calls instead of 3\n", duration)
				} else {
					fmt.Fprintf(os.Stderr, "ℹ️  Original image (%dx%d) is already large enough, using standard edit workflow\n", originalWidth, originalHeight)

					// Use standard edit workflow for large images
					fmt.Fprintln(os.Stderr, "🎨 NSFW Edit Mode: Using Venice.ai for image editing")
					fmt.Fprintln(os.Stderr, "⚠️  Warning: Venice.ai edit may resize your image to 1024x1024, potentially causing pixelation")
					startTime := time.Now()
					imageData, err = makeVeniceEditRequest(imagePath, editPrompt, veniceConfig)
					duration = time.Since(startTime)

					if err != nil {
						fmt.Fprintf(os.Stderr, "Venice.ai editing failed: %v\n", err)
						os.Exit(1)
					}

					// Handle preserve-size for large images
					if preserveSize {
						fmt.Fprintln(os.Stderr, "🔄 Preserving original size: upscaling edited image back to original dimensions")

						// Calculate scale factor needed
						scaleFactor := float64(originalWidth) / 1024.0
						if scaleFactor > 1.0 {
							fmt.Fprintf(os.Stderr, "📏 Original: %dx%d, Venice.ai output: 1024x1024, Scale factor: %.2f\n",
								originalWidth, originalHeight, scaleFactor)

							// Save the edited image temporarily
							tempFile := "temp_preserve_" + filepath.Base(imagePath)
							if err := saveImageData(imageData, tempFile); err != nil {
								fmt.Fprintf(os.Stderr, "Warning: Could not save temp file for upscaling: %v\n", err)
							} else {
								// Upscale back to original dimensions
								upscaledData, err := makeVeniceUpscaleRequest(tempFile, veniceConfig, 0.05, 0.9, "preserve original details exactly")
								if err != nil {
									fmt.Fprintf(os.Stderr, "Warning: Could not upscale back to original size: %v\n", err)
								} else {
									// Replace the image data with upscaled version
									imageData = upscaledData
									fmt.Fprintf(os.Stderr, "✅ Upscaled back to original dimensions\n")
								}

								// Clean up temp file
								os.Remove(tempFile)
							}
						} else {
							fmt.Fprintf(os.Stderr, "ℹ️  Original image is smaller than Venice.ai output, no upscaling needed\n")
						}
					}
				}
			} else {
				// Standard edit workflow
				fmt.Fprintln(os.Stderr, "🎨 NSFW Edit Mode: Using Venice.ai for image editing")
				fmt.Fprintln(os.Stderr, "⚠️  Warning: Venice.ai edit may resize your image to 1024x1024, potentially causing pixelation")
				startTime := time.Now()
				imageData, err = makeVeniceEditRequest(imagePath, editPrompt, veniceConfig)
				duration = time.Since(startTime)

				if err != nil {
					fmt.Fprintf(os.Stderr, "Venice.ai editing failed: %v\n", err)
					os.Exit(1)
				}
			}

			// Stop animation
			if animationDone != nil {
				<-animationDone
				fmt.Fprintf(os.Stderr, "\r\033[K") // Clear animation line
			}

			// Generate filename
			filename := outputFile
			if filename == "" {
				baseName := strings.TrimSuffix(filepath.Base(imagePath), filepath.Ext(imagePath))
				filename = generateFilename(baseName+"_edited", false)
			}

			// Save image
			if err := saveImageData(imageData, filename); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to save image: %v\n", err)
				os.Exit(1)
			}

			fmt.Printf("✅ Image edited and saved as '%s' (took %v)\n", filename, duration)

		} else {
			fmt.Fprintln(os.Stderr, "🔥 NSFW Mode: Using Venice.ai (uncensored text)")
			startTime := time.Now()
			response, err := makeVeniceRequest(prompt, veniceConfig, enableStream, noAnimation)
			duration := time.Since(startTime)

			if err != nil {
				fmt.Fprintf(os.Stderr, "Venice.ai request failed: %v\n", err)
				os.Exit(1)
			}

			// Only print duration if not streaming (streaming already printed the response)
			if !enableStream || noAnimation || !shouldShowAnimation() {
				fmt.Printf("Response (took %v):\n%s\n", duration, response)
			} else {
				fmt.Fprintf(os.Stderr, "\n✅ Response completed in %v\n", duration)
			}
		}
		return
	}

	// Get API credentials from env or config file
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
		fmt.Println("Missing CELESTE_API_ENDPOINT or CELESTE_API_KEY (env or ~/.celesteAI config file).")
		os.Exit(1)
	}

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
		fmt.Fprintf(os.Stderr, "Failed to encode request: %v\n", err)
		os.Exit(1)
	}

	// Declare variables for response handling
	var responseBody []byte
	var start time.Time
	var wasStreamed bool // Track if we used streaming to avoid duplicate output

	// Start corruption animation by default (during wait period)
	var animationDone chan bool
	var animationCtx context.Context
	var cancelAnimation context.CancelFunc
	if !noAnimation && shouldShowAnimation() {
		animationCtx, cancelAnimation = context.WithCancel(context.Background())
		animationDone = make(chan bool)
		startCorruptionAnimation(animationCtx, animationDone, os.Stderr)
		defer cancelAnimation()
	}

	// Try streaming if enabled
	if enableStream {
		// Add stream parameter to request
		var streamReq map[string]interface{}
		if err := json.Unmarshal(body, &streamReq); err == nil {
			streamReq["stream"] = true
			body, _ = json.Marshal(streamReq)
		}

		req, err := http.NewRequest("POST", endpoint+"chat/completions", bytes.NewBuffer(body))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create request: %v\n", err)
			os.Exit(1)
		}
		req.Header.Set("Authorization", "Bearer "+apiKey)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "text/event-stream")

		start = time.Now()
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Request failed: %v\n", err)
			os.Exit(1)
		}
		defer resp.Body.Close()

		// Check if streaming is supported (Content-Type should be text/event-stream)
		if resp.Header.Get("Content-Type") == "text/event-stream" || strings.Contains(resp.Header.Get("Content-Type"), "event-stream") {
			// Handle SSE streaming
			scanner := bufio.NewScanner(resp.Body)
			var fullResponse strings.Builder
			firstToken := true

			for scanner.Scan() {
				line := scanner.Text()
				if strings.HasPrefix(line, "data: ") {
					data := strings.TrimPrefix(line, "data: ")
					if data == "[DONE]" {
						break
					}

					var chunk map[string]interface{}
					if err := json.Unmarshal([]byte(data), &chunk); err == nil {
						if choices, ok := chunk["choices"].([]interface{}); ok && len(choices) > 0 {
							if choice, ok := choices[0].(map[string]interface{}); ok {
								if delta, ok := choice["delta"].(map[string]interface{}); ok {
									if content, ok := delta["content"].(string); ok && content != "" {
										// Stop animation when first token arrives
										if firstToken && animationDone != nil {
											cancelAnimation()
											<-animationDone
											fmt.Fprintf(os.Stderr, "\r\033[K")
											firstToken = false
										}
										// Print token normally without corruption
										fmt.Print(content)
										fullResponse.WriteString(content)
									}
								}
							}
						}
					}
				}
			}

			if err := scanner.Err(); err != nil {
				// Stop animation on error
				if animationDone != nil {
					cancelAnimation()
					<-animationDone
					fmt.Fprintf(os.Stderr, "\r\033[K")
				}
				fmt.Fprintf(os.Stderr, "\nStreaming error: %v\n", err)
				os.Exit(1)
			}

			fmt.Println() // New line after streaming
			elapsed := time.Since(start)
			fmt.Fprintf(os.Stderr, "✅ Response completed in %s\n", elapsed)

			// Parse the full response for conversation entry - properly JSON encode
			result := map[string]interface{}{
				"choices": []interface{}{
					map[string]interface{}{
						"message": map[string]interface{}{
							"content": fullResponse.String(),
						},
					},
				},
			}
			responseBody, _ = json.Marshal(result)
			wasStreamed = true // Mark that we already printed the response
		} else {
			// Streaming not supported, fall back to regular request
			// Stop animation before reading response
			if animationDone != nil {
				cancelAnimation()
				<-animationDone
				fmt.Fprintf(os.Stderr, "\r\033[K")
			}
			responseBody, err = io.ReadAll(resp.Body)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to read response: %v\n", err)
				os.Exit(1)
			}
			elapsed := time.Since(start)
			fmt.Fprintf(os.Stderr, "✅ Response received in %s\n", elapsed)
		}
	} else {
		// Non-streaming request - animation will stop when response arrives
		fmt.Fprintln(os.Stderr, "⏳ Sending request to CelesteAI...")
		start = time.Now()
		req, err := http.NewRequest("POST", endpoint+"chat/completions", bytes.NewBuffer(body))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create request: %v\n", err)
			os.Exit(1)
		}
		req.Header.Set("Authorization", "Bearer "+apiKey)
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Request failed: %v\n", err)
			os.Exit(1)
		}
		defer resp.Body.Close()

		responseBody, err = io.ReadAll(resp.Body)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to read response: %v\n", err)
			os.Exit(1)
		}

		elapsed := time.Since(start)
		fmt.Fprintf(os.Stderr, "✅ Response received in %s\n", elapsed)
	}

	// Stop animation if it was running
	if animationDone != nil {
		<-animationDone
		fmt.Fprintf(os.Stderr, "\r\033[K") // Clear animation line
	}

	if debug {
		fmt.Println(string(responseBody))
	} else {
		var result map[string]interface{}
		if err := json.Unmarshal(responseBody, &result); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to parse response: %v\n", err)
			os.Exit(1)
		}

		if choices, ok := result["choices"].([]interface{}); ok && len(choices) > 0 {
			if choice, ok := choices[0].(map[string]interface{}); ok {
				if message, ok := choice["message"].(map[string]interface{}); ok {
					if content, ok := message["content"].(string); ok {
						// Only print if we didn't already stream it
						if !wasStreamed {
							fmt.Println(content)
						}

						// Upload conversation to S3 if sync flag is set
						if sync {
							entry := createConversationEntry(format, platform, topic, tone, persona, prompt, content, result)
							if err := uploadConversationToS3(entry); err != nil {
								fmt.Fprintf(os.Stderr, "Warning: Failed to upload conversation to S3: %v\n", err)
							}
						}
					} else {
						fmt.Fprintf(os.Stderr, "No content in response\n")
						os.Exit(1)
					}
				} else {
					fmt.Fprintf(os.Stderr, "No message in response\n")
					os.Exit(1)
				}
			} else {
				fmt.Fprintf(os.Stderr, "No choice in response\n")
				os.Exit(1)
			}
		} else {
			fmt.Fprintf(os.Stderr, "No choices in response\n")
			os.Exit(1)
		}
	}
}
