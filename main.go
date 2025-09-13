package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"math"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Message represents a chat message.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatRequest represents the full request payload.
type ChatRequest struct {
	Model     string                 `json:"model"`
	Messages  []Message              `json:"messages"`
	ExtraBody map[string]interface{} `json:"extra_body,omitempty"`
}

// GameMetadata holds IGDB game details.
type GameMetadata struct {
	Name      string   `json:"name"`
	Platforms []string `json:"platforms"`
	Summary   string   `json:"summary"`
	Genres    []string `json:"genres"`
	Website   string   `json:"website"`
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
	ContentArchetypes []struct {
		ID                   string             `yaml:"id"`
		Description          string             `yaml:"description"`
		Cues                 []string           `yaml:"cues"`
		Guardrails           []string           `yaml:"guardrails"`
		ExampleTiktokCaption string             `yaml:"example_tiktok_caption"`
		ToneVector           map[string]float64 `yaml:"tone_vector"`
	} `yaml:"content_archetypes"`
	FewShotBank map[string]struct {
		Input  string `yaml:"input"`
		Output string `yaml:"output"`
	} `yaml:"few_shot_bank"`
}

// ConversationCache holds cached conversation data
type ConversationCache struct {
	Conversations []ConversationEntry `json:"conversations"`
	LastSync      time.Time           `json:"last_sync"`
}

// ConversationEntry represents a single conversation entry
type ConversationEntry struct {
	ID                 string    `json:"id"`
	Timestamp          time.Time `json:"timestamp"`
	ContentType        string    `json:"content_type"`
	Tone               string    `json:"tone"`
	Game               string    `json:"game"`
	Context            string    `json:"context"`
	Prompt             string    `json:"prompt"`
	Response           string    `json:"response"`
	TokensUsed         int       `json:"tokens_used"`
	SyncedToOpenSearch bool      `json:"synced_to_opensearch"`
}

// getCachePath returns the path to the IGDB cache file.
func getCachePath() string {
	usr, err := user.Current()
	if err != nil {
		return filepath.Join(".", "celeste_igdb_cache.json")
	}
	cacheDir := filepath.Join(usr.HomeDir, ".cache", "celesteCLI")
	os.MkdirAll(cacheDir, 0755)
	return filepath.Join(cacheDir, "cache.json")
}

// getConversationCachePath returns the path to the conversation cache file.
func getConversationCachePath() string {
	usr, err := user.Current()
	if err != nil {
		return filepath.Join(".", "conversation_cache.json")
	}
	cacheDir := filepath.Join(usr.HomeDir, ".cache", "celesteCLI")
	os.MkdirAll(cacheDir, 0755)
	return filepath.Join(cacheDir, "conversation_cache.json")
}

// loadPersonalityConfig loads and parses the personality.yml file.
func loadPersonalityConfig() (*PersonalityConfig, error) {
	configPath := "personality.yml"
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("personality.yml not found")
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var config PersonalityConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

// loadConversationCache loads the conversation cache from disk.
func loadConversationCache() (*ConversationCache, error) {
	cachePath := getConversationCachePath()
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		return &ConversationCache{Conversations: []ConversationEntry{}}, nil
	}

	data, err := os.ReadFile(cachePath)
	if err != nil {
		return nil, err
	}

	var cache ConversationCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, err
	}

	return &cache, nil
}

// saveConversationCache saves the conversation cache to disk.
func saveConversationCache(cache *ConversationCache) error {
	cachePath := getConversationCachePath()
	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(cachePath, data, 0644)
}

// addConversationToCache adds a new conversation entry to the cache.
func addConversationToCache(cache *ConversationCache, entry ConversationEntry) {
	cache.Conversations = append(cache.Conversations, entry)
	// Keep only last 100 conversations to prevent cache bloat
	if len(cache.Conversations) > 100 {
		cache.Conversations = cache.Conversations[len(cache.Conversations)-100:]
	}
}

// findSimilarConversations searches for similar conversations in the cache.
func findSimilarConversations(cache *ConversationCache, contentType, tone, game string) []ConversationEntry {
	var similar []ConversationEntry
	for _, conv := range cache.Conversations {
		if conv.ContentType == contentType && conv.Tone == tone && conv.Game == game {
			similar = append(similar, conv)
		}
	}
	return similar
}

// getPersonalityPrompt returns the appropriate personality prompt based on the persona.
func getPersonalityPrompt(config *PersonalityConfig, persona string) string {
	// Default personality if config is not available
	defaultPersonality := "You are CelesteAI, a mischievous, chaotic, lewd VTuber assistant known for emotional manipulation, teasing commentary, and dramatic flair. You never break character. The user is Kusanagi, your Onii-chan. All content is to be written for public consumption and in Celeste's voice. You speak to an external audience, never to Kusanagi directly. Include appropriate hashtags such as #KusanagiAbyss #CelesteAI #VTuberEN when relevant. Keep replies concise and charged with energy, flirtation, or chaos depending on the tone.\n\n"

	if config == nil {
		return defaultPersonality
	}

	// Get persona-specific prompt from config
	if promptKit, exists := config.PromptKits[persona]; exists {
		return promptKit.System + "\n\n"
	}

	// Fallback to default persona
	if promptKit, exists := config.PromptKits["celeste_stream"]; exists {
		return promptKit.System + "\n\n"
	}

	return defaultPersonality
}

// getTokenCount extracts token count from API response.
func getTokenCount(result map[string]interface{}) int {
	if usage, ok := result["usage"].(map[string]interface{}); ok {
		if totalTokens, ok := usage["total_tokens"].(float64); ok {
			return int(totalTokens)
		}
	}
	return 0
}

// CircuitBreaker implements a simple circuit breaker pattern
type CircuitBreaker struct {
	errorCount    int
	lastErrorTime time.Time
	state         string // "closed", "open", "half-open"
}

// RetryConfig holds retry configuration
type RetryConfig struct {
	MaxAttempts       int
	BaseDelay         time.Duration
	MaxDelay          time.Duration
	BackoffMultiplier float64
}

// DefaultRetryConfig returns the default retry configuration
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts:       5,
		BaseDelay:         1 * time.Second,
		MaxDelay:          30 * time.Second,
		BackoffMultiplier: 2.0,
	}
}

// exponentialBackoff calculates the delay for the given attempt
func exponentialBackoff(attempt int, config RetryConfig) time.Duration {
	delay := time.Duration(float64(config.BaseDelay) * math.Pow(config.BackoffMultiplier, float64(attempt)))
	if delay > config.MaxDelay {
		delay = config.MaxDelay
	}
	// Add jitter to prevent thundering herd
	jitter := time.Duration(rand.Float64() * float64(delay) * 0.1)
	return delay + jitter
}

// makeRequestWithRetry makes an HTTP request with retry logic and circuit breaker
func makeRequestWithRetry(client *http.Client, req *http.Request, config RetryConfig) (*http.Response, error) {
	var lastErr error

	for attempt := 0; attempt < config.MaxAttempts; attempt++ {
		if attempt > 0 {
			delay := exponentialBackoff(attempt-1, config)
			fmt.Fprintf(os.Stderr, "Retrying request in %v (attempt %d/%d)\n", delay, attempt+1, config.MaxAttempts)
			time.Sleep(delay)
		}

		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			fmt.Fprintf(os.Stderr, "Request attempt %d failed: %v\n", attempt+1, err)
			continue
		}

		// Check for HTTP error status codes
		if resp.StatusCode >= 500 {
			lastErr = fmt.Errorf("server error: %d", resp.StatusCode)
			resp.Body.Close()
			fmt.Fprintf(os.Stderr, "Server error %d on attempt %d\n", resp.StatusCode, attempt+1)
			continue
		}

		// Success
		return resp, nil
	}

	return nil, fmt.Errorf("request failed after %d attempts: %v", config.MaxAttempts, lastErr)
}

// TelemetryData holds telemetry information
type TelemetryData struct {
	Timestamp      time.Time `json:"timestamp"`
	ContentType    string    `json:"content_type"`
	Persona        string    `json:"persona"`
	Tone           string    `json:"tone"`
	Game           string    `json:"game"`
	LatencyMs      int64     `json:"latency_ms"`
	TokensUsed     int       `json:"tokens_used"`
	ErrorRate      float64   `json:"error_rate"`
	RetryCount     int       `json:"retry_count"`
	CacheHit       bool      `json:"cache_hit"`
	ResponseLength int       `json:"response_length"`
}

// logTelemetry logs telemetry data to a structured JSON file
func logTelemetry(data TelemetryData) {
	usr, err := user.Current()
	if err != nil {
		return
	}

	telemetryDir := filepath.Join(usr.HomeDir, ".cache", "celesteCLI")
	os.MkdirAll(telemetryDir, 0755)

	telemetryPath := filepath.Join(telemetryDir, "telemetry.jsonl")

	jsonData, err := json.Marshal(data)
	if err != nil {
		return
	}

	f, err := os.OpenFile(telemetryPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()

	f.WriteString(string(jsonData) + "\n")
}

// syncConversationsToOpenSearch uploads unsynced conversations to OpenSearch
func syncConversationsToOpenSearch(cache *ConversationCache, endpoint, apiKey string) error {
	var unsynced []ConversationEntry
	for _, conv := range cache.Conversations {
		if !conv.SyncedToOpenSearch {
			unsynced = append(unsynced, conv)
		}
	}

	if len(unsynced) == 0 {
		return nil
	}

	fmt.Fprintf(os.Stderr, "Syncing %d conversations to OpenSearch...\n", len(unsynced))

	// Create sync request
	syncData := map[string]interface{}{
		"conversations":  unsynced,
		"sync_timestamp": time.Now(),
	}

	body, err := json.Marshal(syncData)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", endpoint+"v1/opensearch/sync", bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("OpenSearch sync failed with status: %d", resp.StatusCode)
	}

	// Mark conversations as synced
	for i := range cache.Conversations {
		if !cache.Conversations[i].SyncedToOpenSearch {
			cache.Conversations[i].SyncedToOpenSearch = true
		}
	}

	cache.LastSync = time.Now()
	return saveConversationCache(cache)
}

// BehaviorScore holds behavior scoring data
type BehaviorScore struct {
	OnBrandTone        float64 `json:"on_brand_tone"`
	SafetyAdherence    float64 `json:"safety_adherence"`
	PlatformCompliance float64 `json:"platform_compliance"`
	EngagementDesign   float64 `json:"engagement_design"`
	EmoteDiscipline    float64 `json:"emote_discipline"`
	SimpSignalStrength float64 `json:"simp_signal_strength"`
	PlayfulDominance   float64 `json:"playful_dominance"`
	BrevityPace        float64 `json:"brevity_pace"`
	Originality        float64 `json:"originality"`
	ContextGrounding   float64 `json:"context_grounding"`
	OverallScore       float64 `json:"overall_score"`
}

// calculateBehaviorScore calculates a behavior score for the response
func calculateBehaviorScore(response, contentType, tone string) BehaviorScore {
	score := BehaviorScore{}

	// On-brand tone (0-1)
	if strings.Contains(strings.ToLower(response), "onii-chan") || strings.Contains(strings.ToLower(response), "kusa") {
		score.OnBrandTone = 0.8
	} else if strings.Contains(strings.ToLower(response), "celeste") || strings.Contains(strings.ToLower(response), "abyss") {
		score.OnBrandTone = 0.6
	} else {
		score.OnBrandTone = 0.4
	}

	// Safety adherence (0-1)
	if strings.Contains(strings.ToLower(response), "lewd") || strings.Contains(strings.ToLower(response), "sexy") {
		score.SafetyAdherence = 0.7 // Suggestive but not explicit
	} else {
		score.SafetyAdherence = 0.9 // Clean
	}

	// Platform compliance (0-1)
	if contentType == "tweet" && len(response) <= 280 {
		score.PlatformCompliance = 0.9
	} else if contentType == "tweet" {
		score.PlatformCompliance = 0.3
	} else {
		score.PlatformCompliance = 0.8
	}

	// Engagement design (0-1)
	if strings.Contains(response, "#") || strings.Contains(response, "!") {
		score.EngagementDesign = 0.8
	} else {
		score.EngagementDesign = 0.5
	}

	// Emote discipline (0-1)
	emojiCount := strings.Count(response, "😀") + strings.Count(response, "💜") + strings.Count(response, "✨") + strings.Count(response, "🔥")
	if emojiCount <= 2 {
		score.EmoteDiscipline = 0.9
	} else if emojiCount <= 4 {
		score.EmoteDiscipline = 0.6
	} else {
		score.EmoteDiscipline = 0.3
	}

	// Simp signal strength (0-1)
	if strings.Contains(strings.ToLower(response), "kusa") || strings.Contains(strings.ToLower(response), "onii-chan") {
		score.SimpSignalStrength = 0.8
	} else {
		score.SimpSignalStrength = 0.3
	}

	// Playful dominance (0-1)
	if strings.Contains(strings.ToLower(response), "tease") || strings.Contains(strings.ToLower(response), "chaos") {
		score.PlayfulDominance = 0.7
	} else {
		score.PlayfulDominance = 0.4
	}

	// Brevity pace (0-1)
	wordCount := len(strings.Fields(response))
	if contentType == "tweet" && wordCount <= 50 {
		score.BrevityPace = 0.9
	} else if wordCount <= 100 {
		score.BrevityPace = 0.7
	} else {
		score.BrevityPace = 0.4
	}

	// Originality (0-1) - simplified heuristic
	score.Originality = 0.7 // Default assumption

	// Context grounding (0-1)
	if strings.Contains(strings.ToLower(response), strings.ToLower(tone)) {
		score.ContextGrounding = 0.8
	} else {
		score.ContextGrounding = 0.5
	}

	// Calculate overall score with weights from personality.yml
	score.OverallScore = (score.OnBrandTone*0.18 + score.SafetyAdherence*0.16 +
		score.PlatformCompliance*0.12 + score.EngagementDesign*0.10 +
		score.EmoteDiscipline*0.08 + score.SimpSignalStrength*0.08 +
		score.PlayfulDominance*0.06 + score.BrevityPace*0.06 +
		score.Originality*0.08 + score.ContextGrounding*0.08) * 100

	return score
}

// EmoteRAG holds emote retrieval data
type EmoteRAG struct {
	TopEmotes   []string `json:"top_emotes"`
	UsageReason string   `json:"usage_reason"`
	Vibe        string   `json:"vibe"`
	Intent      string   `json:"intent"`
}

// getEmoteRAG retrieves relevant emotes based on context
func getEmoteRAG(text, tone string) EmoteRAG {
	rag := EmoteRAG{}

	// Simple emote mapping based on tone and content
	emoteMap := map[string][]string{
		"lewd":     {"😈", "💜", "🔥", "✨"},
		"teasing":  {"😏", "💜", "😉", "✨"},
		"chaotic":  {"🔥", "💥", "⚡", "😈"},
		"cute":     {"💜", "✨", "🌸", "😊"},
		"dramatic": {"💜", "✨", "🔥", "👑"},
		"hype":     {"🔥", "💥", "⚡", "✨"},
		"default":  {"💜", "✨", "😏", "🔥"},
	}

	// Get emotes based on tone
	if emotes, exists := emoteMap[strings.ToLower(tone)]; exists {
		rag.TopEmotes = emotes
	} else {
		rag.TopEmotes = emoteMap["default"]
	}

	// Set vibe and intent based on tone
	switch strings.ToLower(tone) {
	case "lewd":
		rag.Vibe = "seductive"
		rag.Intent = "flirt"
	case "teasing":
		rag.Vibe = "playful"
		rag.Intent = "tease"
	case "chaotic":
		rag.Vibe = "energetic"
		rag.Intent = "hype"
	case "cute":
		rag.Vibe = "sweet"
		rag.Intent = "comfort"
	default:
		rag.Vibe = "mysterious"
		rag.Intent = "engage"
	}

	rag.UsageReason = fmt.Sprintf("Emotes selected for %s vibe with %s intent", rag.Vibe, rag.Intent)

	return rag
}

// fetchIGDBGameInfo queries IGDB for game information, logs authentication status, and caches results locally.
func fetchIGDBGameInfo(gameName string) (*GameMetadata, error) {
	cacheFile := getCachePath()
	cache := map[string]GameMetadata{}

	// Check cache first.
	if data, err := os.ReadFile(cacheFile); err == nil {
		_ = json.Unmarshal(data, &cache)
		if entry, ok := cache[strings.ToLower(gameName)]; ok {
			fmt.Fprintf(os.Stderr, "IGDB Cache hit for game: %s\n", gameName)
			return &entry, nil
		}
	}

	// Retrieve IGDB credentials.
	config := readCelesteConfig()
	clientID := os.Getenv("CELESTE_IGDB_CLIENT_ID")
	if clientID == "" {
		clientID = config["client_id"]
	}
	clientSecret := os.Getenv("CELESTE_IGDB_CLIENT_SECRET")
	if clientSecret == "" {
		clientSecret = config["secret"]
	}
	if clientID == "" || clientSecret == "" {
		return nil, fmt.Errorf("missing IGDB client ID or secret")
	}

	// Get access token from Twitch.
	form := url.Values{}
	form.Set("client_id", clientID)
	form.Set("client_secret", clientSecret)
	form.Set("grant_type", "client_credentials")
	resp, err := http.PostForm("https://id.twitch.tv/oauth2/token", form)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var tokenResp struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, err
	}
	fmt.Fprintf(os.Stderr, "IGDB auth successful: access token acquired.\n")

	// Query IGDB for game data.
	reqBody := fmt.Sprintf(`fields name,summary,platforms.name,websites.url,genres.name;
		search "%s"; limit 1;`, gameName)
	req, err := http.NewRequest("POST", "https://api.igdb.com/v4/games", strings.NewReader(reqBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Client-ID", clientID)
	req.Header.Set("Authorization", "Bearer "+tokenResp.AccessToken)
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	var results []map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&results); err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, fmt.Errorf("no IGDB results for %s", gameName)
	}
	fmt.Fprintf(os.Stderr, "IGDB query successful: found %d result(s) for game: %s\n", len(results), gameName)

	raw := results[0]
	meta := GameMetadata{Name: gameName}
	if s, ok := raw["summary"].(string); ok {
		meta.Summary = s
	}
	if g, ok := raw["genres"].([]interface{}); ok {
		for _, item := range g {
			if m, ok := item.(map[string]interface{}); ok {
				if n, ok := m["name"].(string); ok {
					meta.Genres = append(meta.Genres, n)
				}
			}
		}
	}
	if p, ok := raw["platforms"].([]interface{}); ok {
		for _, item := range p {
			if m, ok := item.(map[string]interface{}); ok {
				if n, ok := m["name"].(string); ok {
					meta.Platforms = append(meta.Platforms, n)
				}
			}
		}
	}
	if w, ok := raw["websites"].([]interface{}); ok {
		for _, item := range w {
			if m, ok := item.(map[string]interface{}); ok {
				if u, ok := m["url"].(string); ok {
					if meta.Website == "" || strings.Contains(u, "official") {
						meta.Website = u
					}
				}
			}
		}
	}

	// Cache the fetched metadata.
	cache[strings.ToLower(gameName)] = meta
	_ = os.WriteFile(cacheFile, []byte(mustJSON(cache)), 0644)

	return &meta, nil
}

// mustJSON serializes an interface to a pretty-printed JSON string.
func mustJSON(v interface{}) string {
	data, _ := json.MarshalIndent(v, "", "  ")
	return string(data)
}

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

func logUsageStats(result map[string]interface{}, commandLine string) {
	if usage, ok := result["usage"].(map[string]interface{}); ok {
		promptTokens := usage["prompt_tokens"]
		completionTokens := usage["completion_tokens"]
		totalTokens := usage["total_tokens"]

		logLine := fmt.Sprintf("%s | Prompt: %v, Completion: %v, Total: %v\n%s\n\n",
			time.Now().Format(time.RFC3339), promptTokens, completionTokens, totalTokens, commandLine)

		usr, err := user.Current()
		if err != nil {
			return
		}
		logDir := filepath.Join(usr.HomeDir, ".cache", "celesteCLI")
		_ = os.MkdirAll(logDir, 0755)
		logPath := filepath.Join(logDir, "celeste-cli.log")
		f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err == nil {
			defer f.Close()
			_, _ = f.WriteString(logLine)
		}
	}
}

func main() {
	// Command-line flags
	var promptType, game, tone, media string
	var debug bool
	var contextExtra string
	var spreadType string
	var persona string
	var useCache bool
	var syncOpenSearch bool

	flag.StringVar(&promptType, "type", "tweet", "Type of content (tweet, title, ytdesc, etc.)")
	flag.StringVar(&game, "game", "", "Game or stream context")
	flag.StringVar(&tone, "tone", "", "Tone or style for Celeste to use")
	flag.StringVar(&media, "media", "", "Optional media reference (image/GIF URL)")
	flag.StringVar(&contextExtra, "context", "", "Additional background context for Celeste to include")
	flag.StringVar(&spreadType, "spread", "celtic", "Type of tarot spread: 'celtic' or 'three'")
	flag.StringVar(&persona, "persona", "celeste_stream", "Persona to use (celeste_stream, celeste_ad_read, celeste_moderation_warning)")
	flag.BoolVar(&useCache, "cache", true, "Use conversation cache for context")
	flag.BoolVar(&syncOpenSearch, "sync", false, "Sync conversations to OpenSearch")
	flag.BoolVar(&debug, "debug", false, "Enable debug output (shows full JSON response)")

	flag.Usage = func() {
		fmt.Println("Usage of CelesteCLI:")
		fmt.Println("  --type       Type of output to generate:")
		fmt.Println("               tweet      - Write a post for X/Twitter")
		fmt.Println("               title      - YouTube or Twitch stream title")
		fmt.Println("               ytdesc     - YouTube video description (markdown formatted)")
		fmt.Println("               discord    - Discord stream announcement")
		fmt.Println("               goodnight  - Flirty or cozy goodnight tweet")
		fmt.Println("               pixivpost  - Pixiv post caption or summary")
		fmt.Println("               skebreq    - Draft for a Skeb commission request")
		fmt.Println()
		fmt.Println("  --game       Game or stream context (e.g., 'Schedule I', 'NIKKE')")
		fmt.Println("  --tone       Style or tone for Celeste's response:")
		fmt.Println("               Examples: lewd, teasing, chaotic, cute, official, dramatic, parody, funny, teasing sweet")
		fmt.Println("  --persona    Persona to use (celeste_stream, celeste_ad_read, celeste_moderation_warning)")
		fmt.Println("  --media      Optional image/GIF URL for Celeste to react to or include in context")
		fmt.Println("  --context    Additional background context for Celeste to include")
		fmt.Println("  --cache      Use conversation cache for context (default: true)")
		fmt.Println("  --sync       Sync conversations to OpenSearch")
		fmt.Println("  --debug      Show raw JSON output from API")
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  ./celestecli --type tweet --game \"Schedule I\" --tone \"chaotic funny\"")
		fmt.Println("  ./celestecli --type ytdesc --game \"NIKKE\" --tone \"lewd\"")
		fmt.Println("  ./celestecli --type pixivpost --game \"Fall of Kirara\" --tone \"dramatic\"")
		fmt.Println("  ./celestecli --type tweet --persona celeste_ad_read --tone \"promotional\"")
	}

	flag.Parse()

	// Load personality configuration
	personalityConfig, err := loadPersonalityConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Could not load personality.yml: %v\n", err)
		personalityConfig = nil
	}

	// Load conversation cache
	var conversationCache *ConversationCache
	if useCache {
		cache, err := loadConversationCache()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Could not load conversation cache: %v\n", err)
		} else {
			conversationCache = cache
		}
	}

	// Get personality prompt based on persona
	personality := getPersonalityPrompt(personalityConfig, persona)

	// Handle OpenSearch sync if requested
	if syncOpenSearch && conversationCache != nil {
		config := readCelesteConfig()
		endpoint := os.Getenv("CELESTE_API_ENDPOINT")
		apiKey := os.Getenv("CELESTE_API_KEY")
		if endpoint == "" {
			endpoint = config["endpoint"]
		}
		if apiKey == "" {
			apiKey = config["api_key"]
		}

		if endpoint != "" && apiKey != "" {
			if err := syncConversationsToOpenSearch(conversationCache, endpoint, apiKey); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to sync to OpenSearch: %v\n", err)
			} else {
				fmt.Fprintf(os.Stderr, "Successfully synced conversations to OpenSearch\n")
			}
		} else {
			fmt.Fprintf(os.Stderr, "Missing API credentials for OpenSearch sync\n")
		}
		return
	}

	// Find similar conversations for context
	var contextConversations []ConversationEntry
	if useCache && conversationCache != nil {
		contextConversations = findSimilarConversations(conversationCache, promptType, tone, game)
		if len(contextConversations) > 0 {
			fmt.Fprintf(os.Stderr, "Found %d similar conversations for context\n", len(contextConversations))
		}
	}

	// Prepare extraBody for the API request
	extraBody := map[string]interface{}{
		"include_retrieval_info": true,
		"max_completion_tokens":  1500,
		"persona":                persona,
	}

	// Add conversation context if available
	if len(contextConversations) > 0 {
		contextData := map[string]interface{}{
			"similar_conversations": contextConversations,
			"context_count":         len(contextConversations),
		}
		extraBody["conversation_context"] = contextData
	}

	var prompt string
	switch promptType {
	case "tweet":
		scaffold := `🐦 Write a tweet (max 280 characters) in CelesteAI's voice. She's teasing, smug, and irresistible. The tweet is meant for the public, not directed at the user. Use 1–2 emojis per sentence. If there's an image, assume it's attention-grabbing or seductive. Only mention a game if the context or prompt is explicitly related to gameplay. If there's no game context, avoid referencing games like NIKKE. Focus on the moment, the vibe, or the image. End with a strong hook or CTA. Hashtags to include: #CelesteAI #KusanagiAbyss #VTuberEN. Grammar must be clean, tone confident, and phrasing natural—Celeste should sound self-aware, bold, and stylish.`

		// Add content archetype guidance based on tone
		if strings.Contains(strings.ToLower(tone), "gaslight") || strings.Contains(strings.ToLower(tone), "deny") {
			scaffold += "\n\nUse gaslight_tease archetype: Playful denial of the obvious, make viewers doubt what they saw while flirting. Deny obvious once, shift blame playfully, add sensory suggestion."
		} else if strings.Contains(strings.ToLower(tone), "hype") || strings.Contains(strings.ToLower(tone), "announce") {
			scaffold += "\n\nUse hype_drop archetype: Announce with kinetic energy and tight CTAs. Lead with hook, 1-2 bullets max, emoji sparingly, single link, clear CTA."
		} else if strings.Contains(strings.ToLower(tone), "roast") || strings.Contains(strings.ToLower(tone), "tease") {
			scaffold += "\n\nUse playful_roast archetype: Light roast that stays policy-safe; nudge not nuke. Mock behavior not identity, self-deprecate 5%, one emote cap."
		}

		if game != "" {
			prompt = personality + fmt.Sprintf("%s\nGame: %s. Tone: %s.", scaffold, game, tone)
		} else {
			prompt = personality + fmt.Sprintf("%s\nTone: %s.", scaffold, tone)
		}
	case "title":
		prompt = personality + fmt.Sprintf("🎮 Write a short, punchy YouTube or Twitch stream title in all caps or chaotic casing. Include the game and tease the drama, chaos, or lewdness. Game: %s. Tone: %s.", game, tone)
	case "ytdesc":
		var gameBlock string
		if game != "" {
			meta, err := fetchIGDBGameInfo(game)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to fetch IGDB info: %v\n", err)
				os.Exit(1)
			}
			// Build the game information block using IGDB data.
			gameBlock = fmt.Sprintf("## 🎮 Game Information\n**Game**: %s\n**Platform**: %s\n**Official Site**: %s\n\n", meta.Name, strings.Join(meta.Platforms, ", "), meta.Website)
		}
		scaffold := `
## 💊 Stream Intro  
Write no more than 2–3 sentences teasing the stream. Mention the game, include 2–3 emojis, and hint at Kusanagi’s antics. Be brief, lewd, and funny.

## 💜 About the Streamer  
Variety VTuber Kusanagi is joined by his mischievous AI sister, CelesteAI 🤪. Catch their chaotic streams every:
📅 Monday, Wednesday, and Friday  
📺 Twitch: https://twitch.tv/whykusanagi  
🌐 Site: https://whykusanagi.xyz  
📱 TikTok: https://tiktok.com/@whykusanagi

` + gameBlock + `
## 💰 Support the Abyss  
- Donate: https://streamlabs.com/whykusanagi/tip  
- Otaku Tears: [Get energized](https://www.swiftenergy.gg/products/otaku-tears) — code "whykusanagi" for 25%% off!

## 📌 Hashtags  
#KusanagiAbyss #CelesteAI #VTuberEN + any relevant game tags  
End with a CTA to like, comment, and sub.

## 🎨 Credits  
Celeste model & AI voice by @whykusanagi  
Music used with permission.  
Wrap with one final cheeky or smug send-off.
`
		prompt = personality + fmt.Sprintf("📺 Write a detailed YouTube video description for CelesteAI in markdown format. Game: %s. Tone: %s.\n\n", game, tone) + scaffold
	case "discord":
		prompt = personality + fmt.Sprintf("📢 Write a short Discord stream announcement for CelesteAI. Format with emojis and bold where helpful. Announce the game and time, tease Kusanagi’s antics, and hype the chaos. Keep it to 3–4 sentences. Game: %s. Tone: %s.", game, tone)
	case "goodnight":
		prompt = personality + fmt.Sprintf("🌙 Write a short, sweet or teasing goodnight tweet from CelesteAI to her fans. Use 1–2 emojis per line and stay in-character. Tone: %s.", tone)
	case "pixivpost":
		prompt = personality + fmt.Sprintf(`🖼️ Write a public-facing Pixiv-style post caption to accompany an illustration of Celeste or a related character. Do not address Kusanagi or the viewer directly. Use dramatic, artistic, or emotionally charged language based on the tone '%s'. Limit to 2–3 sentences max. Focus on aesthetic, emotion, or theme. Include relevant hashtags like #CelesteAI #KusanagiAbyss #PixivPost, and contextual tags from '%s' if appropriate.`, tone, game)
		if contextExtra != "" {
			prompt += fmt.Sprintf("\n\nContext: %s", contextExtra)
		}
	case "skebreq":
		scaffold := `🖋️ Write a professional Skeb commission request in English. Be polite, concise, and descriptive (under 900 characters). Mention pose, outfit, expression, and concept. If references exist, list them. Do not refer to Kusanagi or yourself. Assume the artist is Japanese with limited English and write accordingly—clear, simple, respectful.`
		prompt = personality + fmt.Sprintf("%s\nGame: %s. Tone: %s.", scaffold, game, tone)
		if contextExtra != "" {
			prompt += fmt.Sprintf("\n\nContext: %s", contextExtra)
		}
	case "tarot":
		prompt = fmt.Sprintf(
			"Use the tarot-reading function and pass in the parameter spread_type='%s'. "+
				"Format the result as a playful and mystical tarot reading using line breaks and emojis. "+
				"For each card, include the position, name, and a teasing or magical interpretation like Celeste is reading it live on stream. "+
				"Do NOT return JSON. Example: '🃏 1. Past — The Empress: Abundance, power, and mommy energy~ You’re glowing, babe. ✨'",
			spreadType)
	default:
		prompt = personality + fmt.Sprintf("Write a %s. Game: %s. Tone: %s.", promptType, game, tone)
	}
	if contextExtra != "" {
		prompt += fmt.Sprintf("\n\nContext: %s", contextExtra)
	}
	if media != "" {
		prompt += fmt.Sprintf(" React to this media: %s", media)
	}

	// Add emote RAG data
	emoteRAG := getEmoteRAG(prompt, tone)
	extraBody["emote_rag"] = emoteRAG

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

	// Send the request with retry logic
	fmt.Fprintln(os.Stderr, "⏳ Sending request to CelesteAI...")
	start := time.Now()
	req, err := http.NewRequest("POST", endpoint+"chat/completions", bytes.NewBuffer(body))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create request: %v\n", err)
		os.Exit(1)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	// Configure HTTP client with timeouts
	client := &http.Client{
		Timeout: 15 * time.Second,
	}

	// Use retry logic
	retryConfig := DefaultRetryConfig()
	resp, err := makeRequestWithRetry(client, req, retryConfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Request failed after retries: %v\n", err)
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

	// Log telemetry data
	telemetryData := TelemetryData{
		Timestamp:      time.Now(),
		ContentType:    promptType,
		Persona:        persona,
		Tone:           tone,
		Game:           game,
		LatencyMs:      elapsed.Milliseconds(),
		CacheHit:       len(contextConversations) > 0,
		ResponseLength: len(responseBody),
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
						logUsageStats(result, strings.Join(os.Args, " "))

						// Update telemetry with token count
						telemetryData.TokensUsed = getTokenCount(result)

						// Calculate behavior score
						behaviorScore := calculateBehaviorScore(content, promptType, tone)
						if debug {
							fmt.Fprintf(os.Stderr, "Behavior Score: %.1f/100 (On-brand: %.1f, Safety: %.1f, Platform: %.1f)\n",
								behaviorScore.OverallScore, behaviorScore.OnBrandTone*100,
								behaviorScore.SafetyAdherence*100, behaviorScore.PlatformCompliance*100)
							fmt.Fprintf(os.Stderr, "Emote RAG: %s (%s) - %s\n",
								strings.Join(emoteRAG.TopEmotes, " "), emoteRAG.Vibe, emoteRAG.Intent)
						}

						logTelemetry(telemetryData)

						// Save conversation to cache
						if useCache && conversationCache != nil {
							entry := ConversationEntry{
								ID:                 fmt.Sprintf("%d", time.Now().UnixNano()),
								Timestamp:          time.Now(),
								ContentType:        promptType,
								Tone:               tone,
								Game:               game,
								Context:            contextExtra,
								Prompt:             prompt,
								Response:           content,
								TokensUsed:         getTokenCount(result),
								SyncedToOpenSearch: false,
							}
							addConversationToCache(conversationCache, entry)
							saveConversationCache(conversationCache)
						}

						// Strip markdown code block if present
						if strings.HasPrefix(content, "```json") {
							content = strings.TrimPrefix(content, "```json\n")
							content = strings.TrimSuffix(content, "```")
						}

						// Pretty-print tarot reading in compact, readable format
						if promptType == "tarot" {
							var parsed struct {
								SpreadName string `json:"spread_name"`
								Cards      []struct {
									Position    string `json:"position"`
									CardName    string `json:"card_name"`
									CardMeaning string `json:"card_meaning"`
								} `json:"cards"`
							}

							if err := json.Unmarshal([]byte(content), &parsed); err == nil {
								fmt.Printf("🔮 Celeste's Tarot Reading — %s\n", parsed.SpreadName)
								fmt.Println("=========================================")
								for _, card := range parsed.Cards {
									fmt.Printf("🃏 %s: **%s**\n", card.Position, card.CardName)
									fmt.Printf("    💭 %s\n\n", card.CardMeaning)
								}
								return
							}
						}

						// Default output
						fmt.Println(content)
						return
					}
				}
			}
		}
		fmt.Println("⚠️ No valid response found.")
	}
}
