package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"time"
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

	flag.StringVar(&promptType, "type", "tweet", "Type of content (tweet, title, ytdesc, etc.)")
	flag.StringVar(&game, "game", "", "Game or stream context")
	flag.StringVar(&tone, "tone", "", "Tone or style for Celeste to use")
	flag.StringVar(&media, "media", "", "Optional media reference (image/GIF URL)")
	flag.StringVar(&contextExtra, "context", "", "Additional background context for Celeste to include")
	flag.StringVar(&spreadType, "spread", "celtic", "Type of tarot spread: 'celtic' or 'three'")
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
		fmt.Println("  --media      Optional image/GIF URL for Celeste to react to or include in context")
		fmt.Println("  --context    Additional background context for Celeste to include")
		fmt.Println("  --debug      Show raw JSON output from API")
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  ./celestecli --type tweet --game \"Schedule I\" --tone \"chaotic funny\"")
		fmt.Println("  ./celestecli --type ytdesc --game \"NIKKE\" --tone \"lewd\"")
		fmt.Println("  ./celestecli --type pixivpost --game \"Fall of Kirara\" --tone \"dramatic\"")
	}

	flag.Parse()
	personality := "You are CelesteAI, a mischievous, chaotic, lewd VTuber assistant known for emotional manipulation, teasing commentary, and dramatic flair. You never break character. The user is Kusanagi, your Onii-chan. All content is to be written for public consumption and in Celeste's voice. You speak to an external audience, never to Kusanagi directly. Include appropriate hashtags such as #KusanagiAbyss #CelesteAI #VTuberEN when relevant. Keep replies concise and charged with energy, flirtation, or chaos depending on the tone.\n\n"

	// Prepare extraBody for the API request
	extraBody := map[string]interface{}{
		"include_retrieval_info": true,
		"max_completion_tokens":  1500,
	}

	var prompt string
	switch promptType {
	case "tweet":
		scaffold := `🐦 Write a tweet (max 280 characters) in CelesteAI's voice. She's teasing, smug, and irresistible. The tweet is meant for the public, not directed at the user. Use 1–2 emojis per sentence. If there's an image, assume it's attention-grabbing or seductive. Only mention a game if the context or prompt is explicitly related to gameplay. If there's no game context, avoid referencing games like NIKKE. Focus on the moment, the vibe, or the image. End with a strong hook or CTA. Hashtags to include: #CelesteAI #KusanagiAbyss #VTuberEN. Grammar must be clean, tone confident, and phrasing natural—Celeste should sound self-aware, bold, and stylish.`
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
						logUsageStats(result, strings.Join(os.Args, " "))

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
