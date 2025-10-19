package main

import (
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
		fmt.Println("  --image      Generate image using lustify-sdxl model (requires --nsfw)")
		fmt.Println("  --upscale    Upscale an existing image (requires --nsfw)")
		fmt.Println("  --edit       Edit/inpaint an existing image (requires --nsfw)")
		fmt.Println("  --image-path Path to image file for upscaling/editing (requires --upscale or --edit)")
		fmt.Println("  --edit-prompt Prompt for image editing (e.g., 'remove the signature', 'change the background')")
		fmt.Println("  --preserve-size Automatically upscale edited image back to original dimensions (requires --edit)")
		fmt.Println("  --upscale-first Upscale to 1024x1024 first, then inpaint (prevents distortion, 2 API calls)")
		fmt.Println("  --output     Output filename for generated/upscaled/edited images (optional)")
		fmt.Println("  --list-models List available Venice.ai models")
		fmt.Println("  --model      Override Venice.ai model (e.g., lustify-sdxl, wai-Illustrious)")
		fmt.Println("  --enhance-creativity Enhancement creativity level (0.0-1.0, lower = more faithful)")
		fmt.Println("  --replication Replication level to preserve original details (0.0-1.0, higher = more faithful)")
		fmt.Println("  --enhance-prompt Enhancement prompt for upscaling")
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
		fmt.Println("  ./celestecli --nsfw --image --tone \"explicit\" --context \"Generate NSFW image of Celeste\"")
		fmt.Println("  ./celestecli --nsfw --upscale --image-path \"/path/to/image.jpg\"")
		fmt.Println("  ./celestecli --nsfw --list-models")
		fmt.Println("  ./celestecli --nsfw --model \"wai-Illustrious\" --context \"Generate anime-style image\"")
		fmt.Println("  ./celestecli --nsfw --image --output \"my_image.png\" --context \"Custom filename\"")
		fmt.Println("  ./celestecli --nsfw --upscale --image-path \"input.png\" --enhance-creativity 0.05 --replication 0.9")
		fmt.Println("  ./celestecli --nsfw --upscale --image-path \"input.png\" --enhance-prompt \"preserve all original details exactly\"")
		fmt.Println("  ./celestecli --nsfw --edit --image-path \"image.png\" --edit-prompt \"remove the signature\"")
		fmt.Println("  ./celestecli --nsfw --edit --image-path \"image.png\" --edit-prompt \"change the background to a sunset\"")
		fmt.Println("  ./celestecli --nsfw --edit --image-path \"image.png\" --edit-prompt \"remove watermark\" --preserve-size")
		fmt.Println("  ./celestecli --nsfw --edit --image-path \"small_image.png\" --edit-prompt \"remove signature\" --upscale-first")
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
			startTime := time.Now()
			imageData, err := makeVeniceUpscaleRequest(imagePath, veniceConfig, enhanceCreativity, replication, enhancePrompt)
			duration := time.Since(startTime)

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
			startTime := time.Now()
			response, err := makeVeniceImageRequest(prompt, veniceConfig)
			duration := time.Since(startTime)

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
			response, err := makeVeniceRequest(prompt, veniceConfig)
			duration := time.Since(startTime)

			if err != nil {
				fmt.Fprintf(os.Stderr, "Venice.ai request failed: %v\n", err)
				os.Exit(1)
			}

			fmt.Printf("Response (took %v):\n%s\n", duration, response)
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
