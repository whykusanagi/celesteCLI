package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
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
func createConversationEntry(promptType, game, tone, persona, prompt, response string, result map[string]interface{}) *ConversationEntry {
	// Get user ID from environment or default to kusanagi
	userID := os.Getenv("CELESTE_USER_ID")
	if userID == "" {
		userID = "kusanagi" // Default user ID
	}

	entry := &ConversationEntry{
		ID:          fmt.Sprintf("%d", time.Now().UnixNano()),
		Timestamp:   time.Now(),
		UserID:      userID,
		ContentType: promptType,
		Tone:        tone,
		Game:        game,
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
	entry.Intent = determineIntent(promptType)
	entry.Purpose = promptType
	entry.Platform = determinePlatform(promptType)
	entry.Sentiment = determineSentiment(tone)
	entry.Topics = extractTopics(game, prompt, response)
	entry.Tags = generateTags(promptType, game, tone, persona)
	entry.Context = fmt.Sprintf("Game: %s, Tone: %s, Persona: %s", game, tone, persona)

	return entry
}

// Helper functions for RAG
func determineIntent(contentType string) string {
	switch contentType {
	case "tweet", "tweet_image", "tweet_thread", "quote_tweet", "reply_snark", "birthday":
		return "social_media"
	case "ytdesc", "title":
		return "content_creation"
	case "discord":
		return "community_management"
	case "tiktok":
		return "short_form_content"
	default:
		return "general"
	}
}

func determinePlatform(contentType string) string {
	switch contentType {
	case "tweet", "tweet_image", "tweet_thread", "quote_tweet", "reply_snark", "birthday":
		return "twitter"
	case "ytdesc", "title":
		return "youtube"
	case "tiktok":
		return "tiktok"
	case "discord":
		return "discord"
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

func extractTopics(game, prompt, response string) []string {
	topics := []string{}
	if game != "" {
		topics = append(topics, strings.ToLower(game))
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

func generateTags(promptType, game, tone, persona string) []string {
	tags := []string{"celeste", "ai", "content"}
	if game != "" {
		tags = append(tags, "game:"+strings.ToLower(game))
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

// makeVeniceRequest makes a request to Venice.ai API
func makeVeniceRequest(prompt string, config *VeniceConfig) (string, error) {
	clientConfig := openai.DefaultConfig(config.APIKey)
	clientConfig.BaseURL = config.BaseURL
	client := openai.NewClientWithConfig(clientConfig)

	resp, err := client.CreateChatCompletion(
		context.Background(),
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

func main() {
	// Command-line flags
	var promptType, game, tone, media string
	var debug, sync, nsfw bool
	var contextExtra string
	var spreadType string
	var persona string

	flag.StringVar(&promptType, "type", "tweet", "Type of content (tweet, title, ytdesc, etc.)")
	flag.StringVar(&game, "game", "", "Game or stream context")
	flag.StringVar(&tone, "tone", "", "Tone or style for Celeste to use")
	flag.StringVar(&media, "media", "", "Optional media reference (image/GIF URL)")
	flag.StringVar(&contextExtra, "context", "", "Additional background context for Celeste to include")
	flag.StringVar(&spreadType, "spread", "celtic", "Type of tarot spread: 'celtic' or 'three'")
	flag.StringVar(&persona, "persona", "celeste_stream", "Persona to use (celeste_stream, celeste_ad_read, celeste_moderation_warning)")
	flag.BoolVar(&debug, "debug", false, "Enable debug output (shows full JSON response)")
	flag.BoolVar(&sync, "sync", false, "Upload conversation to DigitalOcean Spaces after completion")
	flag.BoolVar(&nsfw, "nsfw", false, "Enable NSFW mode using Venice.ai (uncensored content generation)")

	flag.Usage = func() {
		fmt.Println("Usage of CelesteCLI:")
		fmt.Println("  --type       Type of output to generate:")
		fmt.Println("               tweet         - Write a post for X/Twitter")
		fmt.Println("               tweet_image   - Twitter post with image/art credit")
		fmt.Println("               tweet_thread  - Multi-part Twitter thread")
		fmt.Println("               title         - YouTube or Twitch stream title")
		fmt.Println("               ytdesc        - YouTube video description (markdown formatted)")
		fmt.Println("               tiktok        - TikTok caption")
		fmt.Println("               discord       - Discord stream announcement")
		fmt.Println("               goodnight     - Flirty or cozy goodnight tweet")
		fmt.Println("               pixivpost     - Pixiv post caption or summary")
		fmt.Println("               skebreq       - Draft for a Skeb commission request")
		fmt.Println("               quote_tweet   - Quote tweet response")
		fmt.Println("               reply_snark   - Snarky reply to tweet")
		fmt.Println("               birthday        - Birthday message")
		fmt.Println("               alt_text      - Image alt text")
		fmt.Println()
		fmt.Println("  --game       Game or stream context (e.g., 'Schedule I', 'NIKKE')")
		fmt.Println("  --tone       Style or tone for Celeste's response:")
		fmt.Println("               Examples: lewd, teasing, chaotic, cute, official, dramatic, parody, funny, teasing sweet")
		fmt.Println("  --persona    Persona to use (celeste_stream, celeste_ad_read, celeste_moderation_warning)")
		fmt.Println("  --media      Optional image/GIF URL for Celeste to react to or include in context")
		fmt.Println("  --context    Additional background context for Celeste to include")
		fmt.Println("  --sync       Upload conversation to DigitalOcean Spaces after completion")
		fmt.Println("  --nsfw       Enable NSFW mode using Venice.ai (uncensored content generation)")
		fmt.Println("  --debug      Show raw JSON output from API")
		fmt.Println()
		fmt.Println("Configuration:")
		fmt.Println("  ~/.celeste.cfg              - Celeste configuration file (preferred)")
		fmt.Println("  Environment Variables       - Fallback if ~/.celeste.cfg not found")
		fmt.Println("    DO_SPACES_ACCESS_KEY_ID     - DigitalOcean Spaces access key")
		fmt.Println("    DO_SPACES_SECRET_ACCESS_KEY - DigitalOcean Spaces secret key")
		fmt.Println("    CELESTE_USER_ID             - User ID for conversation tracking")
		fmt.Println("    CELESTE_PLATFORM            - Platform (discord, twitch, cli)")
		fmt.Println("    CELESTE_OVERRIDE_ENABLED    - Enable override mode (true/false)")
		fmt.Println("    CELESTE_PGP_SIGNATURE       - PGP signature for override commands")
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  ./celestecli --type tweet --game \"Schedule I\" --tone \"chaotic funny\"")
		fmt.Println("  ./celestecli --type tweet_image --game \"NIKKE\" --tone \"lewd\" --sync")
		fmt.Println("  ./celestecli --type tiktok --game \"NIKKE\" --tone \"teasing\"")
		fmt.Println("  ./celestecli --type quote_tweet --tone \"snarky\"")
		fmt.Println("  ./celestecli --type birthday --tone \"playful\"")
	}

	flag.Parse()

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

	// Build prompt using scaffolding system
	prompt := getScaffoldPrompt(promptType, game, tone, scaffoldingConfig)
	prompt = personality + prompt

	if contextExtra != "" {
		prompt += fmt.Sprintf("\n\nContext: %s", contextExtra)
	}
	if media != "" {
		prompt += fmt.Sprintf(" React to this media: %s", media)
	}

	// Handle NSFW mode with Venice.ai
	if nsfw {
		veniceConfig, err := loadVeniceConfig()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Venice.ai configuration error: %v\n", err)
			os.Exit(1)
		}

		fmt.Fprintln(os.Stderr, "🔥 NSFW Mode: Using Venice.ai (uncensored)")
		response, err := makeVeniceRequest(prompt, veniceConfig)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Venice.ai request failed: %v\n", err)
			os.Exit(1)
		}

		fmt.Println(response)
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

	// Add tarot function call and spread type if needed
	extraBody := make(map[string]interface{})
	if promptType == "tarot" {
		extraBody["function_call"] = map[string]string{"name": "tarot-reading"}
		extraBody["function_args"] = map[string]string{"spread_type": spreadType}
	}

	// Build the request payload
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

	// Send the request
	fmt.Fprintln(os.Stderr, "⏳ Sending request to CelesteAI...")
	start := time.Now()
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

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to read response: %v\n", err)
		os.Exit(1)
	}

	elapsed := time.Since(start)
	fmt.Fprintf(os.Stderr, "✅ Response received in %s\n", elapsed)

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
						fmt.Println(content)

						// Upload conversation to S3 if sync flag is set
						if sync {
							entry := createConversationEntry(promptType, game, tone, persona, prompt, content, result)
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
