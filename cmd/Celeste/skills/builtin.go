// Package skills provides the skill registry and execution system.
// This file contains built-in skill implementations.
package skills

import (
	"bytes"
	"crypto/md5"
	"crypto/rand"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/skip2/go-qrcode"
)

// RegisterBuiltinSkills registers all built-in skills with the registry.
func RegisterBuiltinSkills(registry *Registry, configLoader ConfigLoader) {
	// Register skill definitions
	registry.RegisterSkill(TarotSkill())
	registry.RegisterSkill(NSFWSkill())
	registry.RegisterSkill(ContentSkill())
	registry.RegisterSkill(ImageSkill())
	registry.RegisterSkill(WeatherSkill())
	registry.RegisterSkill(UnitConverterSkill())
	registry.RegisterSkill(TimezoneConverterSkill())
	registry.RegisterSkill(HashGeneratorSkill())
	registry.RegisterSkill(Base64EncodeSkill())
	registry.RegisterSkill(Base64DecodeSkill())
	registry.RegisterSkill(UUIDGeneratorSkill())
	registry.RegisterSkill(PasswordGeneratorSkill())
	registry.RegisterSkill(CurrencyConverterSkill())
	registry.RegisterSkill(QRCodeGeneratorSkill())
	registry.RegisterSkill(TwitchLiveCheckSkill())
	registry.RegisterSkill(YouTubeVideosSkill())
	registry.RegisterSkill(SetReminderSkill())
	registry.RegisterSkill(ListRemindersSkill())
	registry.RegisterSkill(SaveNoteSkill())
	registry.RegisterSkill(GetNoteSkill())
	registry.RegisterSkill(ListNotesSkill())

	// Register handlers
	registry.RegisterHandler("tarot_reading", func(args map[string]interface{}) (interface{}, error) {
		return TarotHandler(args, configLoader)
	})
	registry.RegisterHandler("nsfw_mode", func(args map[string]interface{}) (interface{}, error) {
		return NSFWHandler(args, configLoader)
	})
	registry.RegisterHandler("generate_content", func(args map[string]interface{}) (interface{}, error) {
		return ContentHandler(args)
	})
	registry.RegisterHandler("generate_image", func(args map[string]interface{}) (interface{}, error) {
		return ImageHandler(args, configLoader)
	})
	registry.RegisterHandler("get_weather", func(args map[string]interface{}) (interface{}, error) {
		return WeatherHandler(args, configLoader)
	})
	registry.RegisterHandler("convert_units", func(args map[string]interface{}) (interface{}, error) {
		return UnitConverterHandler(args)
	})
	registry.RegisterHandler("convert_timezone", func(args map[string]interface{}) (interface{}, error) {
		return TimezoneConverterHandler(args)
	})
	registry.RegisterHandler("generate_hash", func(args map[string]interface{}) (interface{}, error) {
		return HashGeneratorHandler(args)
	})
	registry.RegisterHandler("base64_encode", func(args map[string]interface{}) (interface{}, error) {
		return Base64EncodeHandler(args)
	})
	registry.RegisterHandler("base64_decode", func(args map[string]interface{}) (interface{}, error) {
		return Base64DecodeHandler(args)
	})
	registry.RegisterHandler("generate_uuid", func(args map[string]interface{}) (interface{}, error) {
		return UUIDGeneratorHandler(args)
	})
	registry.RegisterHandler("generate_password", func(args map[string]interface{}) (interface{}, error) {
		return PasswordGeneratorHandler(args)
	})
	registry.RegisterHandler("convert_currency", func(args map[string]interface{}) (interface{}, error) {
		return CurrencyConverterHandler(args)
	})
	registry.RegisterHandler("generate_qr_code", func(args map[string]interface{}) (interface{}, error) {
		return QRCodeGeneratorHandler(args)
	})
	registry.RegisterHandler("check_twitch_live", func(args map[string]interface{}) (interface{}, error) {
		return TwitchLiveCheckHandler(args, configLoader)
	})
	registry.RegisterHandler("get_youtube_videos", func(args map[string]interface{}) (interface{}, error) {
		return YouTubeVideosHandler(args, configLoader)
	})
	registry.RegisterHandler("set_reminder", func(args map[string]interface{}) (interface{}, error) {
		return SetReminderHandler(args)
	})
	registry.RegisterHandler("list_reminders", func(args map[string]interface{}) (interface{}, error) {
		return ListRemindersHandler(args)
	})
	registry.RegisterHandler("save_note", func(args map[string]interface{}) (interface{}, error) {
		return SaveNoteHandler(args)
	})
	registry.RegisterHandler("get_note", func(args map[string]interface{}) (interface{}, error) {
		return GetNoteHandler(args)
	})
	registry.RegisterHandler("list_notes", func(args map[string]interface{}) (interface{}, error) {
		return ListNotesHandler(args)
	})
}

// ConfigLoader provides access to configuration values.
type ConfigLoader interface {
	GetTarotConfig() (TarotConfig, error)
	GetVeniceConfig() (VeniceConfig, error)
	GetWeatherConfig() (WeatherConfig, error)
	GetTwitchConfig() (TwitchConfig, error)
	GetYouTubeConfig() (YouTubeConfig, error)
}

// TarotConfig holds tarot function configuration.
type TarotConfig struct {
	FunctionURL string
	AuthToken   string
}

// VeniceConfig holds Venice.ai configuration.
type VeniceConfig struct {
	APIKey   string
	BaseURL  string
	Model    string
	Upscaler string
}

// WeatherConfig holds weather skill configuration.
type WeatherConfig struct {
	DefaultZipCode string
}

// TwitchConfig holds Twitch API configuration.
type TwitchConfig struct {
	ClientID       string
	DefaultStreamer string
}

// YouTubeConfig holds YouTube API configuration.
type YouTubeConfig struct {
	APIKey         string
	DefaultChannel string
}

// --- Skill Definitions ---

// TarotSkill returns the tarot reading skill definition.
func TarotSkill() Skill {
	return Skill{
		Name:        "tarot_reading",
		Description: "Generate a tarot card reading using either a three-card spread (past/present/future) or a celtic cross spread",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"spread_type": map[string]interface{}{
					"type":        "string",
					"enum":        []string{"three", "celtic"},
					"description": "Type of spread: 'three' for 3-card past/present/future, 'celtic' for 10-card celtic cross",
				},
				"question": map[string]interface{}{
					"type":        "string",
					"description": "Optional question to focus the reading on",
				},
			},
			"required": []string{"spread_type"},
		},
	}
}

// NSFWSkill returns the NSFW mode skill definition.
func NSFWSkill() Skill {
	return Skill{
		Name:        "nsfw_mode",
		Description: "Enable NSFW mode using Venice.ai for uncensored content generation. Changes UI theme and uses Venice.ai endpoint.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"enable": map[string]interface{}{
					"type":        "boolean",
					"description": "Enable (true) or disable (false) NSFW mode",
				},
			},
			"required": []string{"enable"},
		},
	}
}

// ContentSkill returns the content generation skill definition.
func ContentSkill() Skill {
	return Skill{
		Name:        "generate_content",
		Description: "Generate content for various platforms (Twitter, TikTok, YouTube, Discord) with specific format and tone",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"platform": map[string]interface{}{
					"type":        "string",
					"enum":        []string{"twitter", "tiktok", "youtube", "discord"},
					"description": "Target platform for the content",
				},
				"format": map[string]interface{}{
					"type":        "string",
					"enum":        []string{"short", "long", "general"},
					"description": "Content length: short (280 chars), long (5000 chars), general (flexible)",
				},
				"tone": map[string]interface{}{
					"type":        "string",
					"description": "Tone/style for the content (e.g., teasing, cute, dramatic, funny)",
				},
				"topic": map[string]interface{}{
					"type":        "string",
					"description": "Topic or subject for the content",
				},
			},
			"required": []string{"platform", "topic"},
		},
	}
}

// ImageSkill returns the image generation skill definition.
func ImageSkill() Skill {
	return Skill{
		Name:        "generate_image",
		Description: "Generate an image using Venice.ai based on a text prompt",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"prompt": map[string]interface{}{
					"type":        "string",
					"description": "Text description of the image to generate",
				},
				"style": map[string]interface{}{
					"type":        "string",
					"description": "Optional style modifier (e.g., anime, realistic, painting)",
				},
			},
			"required": []string{"prompt"},
		},
	}
}

// WeatherSkill returns the weather forecast skill definition.
func WeatherSkill() Skill {
	return Skill{
		Name:        "get_weather",
		Description: "Get current weather and forecast for a location. Uses default zip code if not specified. User can provide zip code in the prompt to override default.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"zip_code": map[string]interface{}{
					"type":        "string",
					"description": "Optional zip code (5 digits). If not provided, uses default zip code from configuration. User can specify zip code in their message to override default.",
				},
				"days": map[string]interface{}{
					"type":        "integer",
					"description": "Number of days for forecast (1-3, default: 1 for current weather)",
				},
			},
			"required": []string{},
		},
	}
}

// UnitConverterSkill returns the unit converter skill definition.
func UnitConverterSkill() Skill {
	return Skill{
		Name:        "convert_units",
		Description: "Convert between different units of measurement (length, weight, temperature, volume)",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"value": map[string]interface{}{
					"type":        "number",
					"description": "The numeric value to convert",
				},
				"from_unit": map[string]interface{}{
					"type":        "string",
					"description": "Source unit (e.g., 'm', 'km', 'ft', 'mi', 'kg', 'lb', 'celsius', 'fahrenheit', 'liter', 'gallon')",
				},
				"to_unit": map[string]interface{}{
					"type":        "string",
					"description": "Target unit (e.g., 'm', 'km', 'ft', 'mi', 'kg', 'lb', 'celsius', 'fahrenheit', 'liter', 'gallon')",
				},
			},
			"required": []string{"value", "from_unit", "to_unit"},
		},
	}
}

// TimezoneConverterSkill returns the timezone converter skill definition.
func TimezoneConverterSkill() Skill {
	return Skill{
		Name:        "convert_timezone",
		Description: "Convert time between different timezones",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"time": map[string]interface{}{
					"type":        "string",
					"description": "Time to convert (format: 'HH:MM' or 'HH:MM:SS', defaults to current time if not provided)",
				},
				"from_timezone": map[string]interface{}{
					"type":        "string",
					"description": "Source timezone (e.g., 'America/New_York', 'UTC', 'Asia/Tokyo')",
				},
				"to_timezone": map[string]interface{}{
					"type":        "string",
					"description": "Target timezone (e.g., 'America/New_York', 'UTC', 'Asia/Tokyo')",
				},
				"date": map[string]interface{}{
					"type":        "string",
					"description": "Optional date (format: 'YYYY-MM-DD', defaults to today if not provided)",
				},
			},
			"required": []string{"from_timezone", "to_timezone"},
		},
	}
}

// HashGeneratorSkill returns the hash generator skill definition.
func HashGeneratorSkill() Skill {
	return Skill{
		Name:        "generate_hash",
		Description: "Generate cryptographic hash (MD5, SHA256, SHA512) for a given string",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"text": map[string]interface{}{
					"type":        "string",
					"description": "Text to hash",
				},
				"algorithm": map[string]interface{}{
					"type":        "string",
					"enum":        []string{"md5", "sha256", "sha512"},
					"description": "Hash algorithm to use",
				},
			},
			"required": []string{"text", "algorithm"},
		},
	}
}

// Base64EncodeSkill returns the base64 encoder skill definition.
func Base64EncodeSkill() Skill {
	return Skill{
		Name:        "base64_encode",
		Description: "Encode a string to base64",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"text": map[string]interface{}{
					"type":        "string",
					"description": "Text to encode",
				},
			},
			"required": []string{"text"},
		},
	}
}

// Base64DecodeSkill returns the base64 decoder skill definition.
func Base64DecodeSkill() Skill {
	return Skill{
		Name:        "base64_decode",
		Description: "Decode a base64 string",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"encoded": map[string]interface{}{
					"type":        "string",
					"description": "Base64 encoded string to decode",
				},
			},
			"required": []string{"encoded"},
		},
	}
}

// UUIDGeneratorSkill returns the UUID generator skill definition.
func UUIDGeneratorSkill() Skill {
	return Skill{
		Name:        "generate_uuid",
		Description: "Generate a random UUID (v4)",
		Parameters: map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
			"required":   []string{},
		},
	}
}

// PasswordGeneratorSkill returns the password generator skill definition.
func PasswordGeneratorSkill() Skill {
	return Skill{
		Name:        "generate_password",
		Description: "Generate a secure random password",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"length": map[string]interface{}{
					"type":        "integer",
					"description": "Password length (default: 16, min: 8, max: 128)",
				},
				"include_symbols": map[string]interface{}{
					"type":        "boolean",
					"description": "Include special symbols (default: true)",
				},
				"include_numbers": map[string]interface{}{
					"type":        "boolean",
					"description": "Include numbers (default: true)",
				},
			},
			"required": []string{},
		},
	}
}

// CurrencyConverterSkill returns the currency converter skill definition.
func CurrencyConverterSkill() Skill {
	return Skill{
		Name:        "convert_currency",
		Description: "Convert between different currencies using current exchange rates",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"amount": map[string]interface{}{
					"type":        "number",
					"description": "Amount to convert",
				},
				"from_currency": map[string]interface{}{
					"type":        "string",
					"description": "Source currency code (e.g., 'USD', 'EUR', 'JPY', 'GBP')",
				},
				"to_currency": map[string]interface{}{
					"type":        "string",
					"description": "Target currency code (e.g., 'USD', 'EUR', 'JPY', 'GBP')",
				},
			},
			"required": []string{"amount", "from_currency", "to_currency"},
		},
	}
}

// QRCodeGeneratorSkill returns the QR code generator skill definition.
func QRCodeGeneratorSkill() Skill {
	return Skill{
		Name:        "generate_qr_code",
		Description: "Generate a QR code from text or URL",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"text": map[string]interface{}{
					"type":        "string",
					"description": "Text or URL to encode in QR code",
				},
				"size": map[string]interface{}{
					"type":        "integer",
					"description": "QR code size in pixels (default: 256, min: 64, max: 1024)",
				},
			},
			"required": []string{"text"},
		},
	}
}

// TwitchLiveCheckSkill returns the Twitch live check skill definition.
func TwitchLiveCheckSkill() Skill {
	return Skill{
		Name:        "check_twitch_live",
		Description: "Check if a Twitch streamer is currently live. Uses default streamer if not specified. User can provide streamer name in prompt to override default.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"streamer": map[string]interface{}{
					"type":        "string",
					"description": "Optional Twitch streamer username. If not provided, uses default streamer from configuration. User can specify streamer name in their message to override default.",
				},
			},
			"required": []string{},
		},
	}
}

// YouTubeVideosSkill returns the YouTube recent videos skill definition.
func YouTubeVideosSkill() Skill {
	return Skill{
		Name:        "get_youtube_videos",
		Description: "Get recent videos from a YouTube channel. Uses default channel if not specified. User can provide channel name/ID in prompt to override default.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"channel": map[string]interface{}{
					"type":        "string",
					"description": "Optional YouTube channel username or channel ID. If not provided, uses default channel from configuration. User can specify channel in their message to override default.",
				},
				"max_results": map[string]interface{}{
					"type":        "integer",
					"description": "Maximum number of videos to return (default: 5, min: 1, max: 50)",
				},
			},
			"required": []string{},
		},
	}
}

// SetReminderSkill returns the set reminder skill definition.
func SetReminderSkill() Skill {
	return Skill{
		Name:        "set_reminder",
		Description: "Set a reminder with a specific time and message",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"message": map[string]interface{}{
					"type":        "string",
					"description": "Reminder message",
				},
				"time": map[string]interface{}{
					"type":        "string",
					"description": "Time for reminder (format: 'YYYY-MM-DD HH:MM' or 'HH:MM' for today, or relative like 'in 1 hour', 'tomorrow at 3pm')",
				},
			},
			"required": []string{"message", "time"},
		},
	}
}

// ListRemindersSkill returns the list reminders skill definition.
func ListRemindersSkill() Skill {
	return Skill{
		Name:        "list_reminders",
		Description: "List all active reminders",
		Parameters: map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
			"required":   []string{},
		},
	}
}

// SaveNoteSkill returns the save note skill definition.
func SaveNoteSkill() Skill {
	return Skill{
		Name:        "save_note",
		Description: "Save a note with an optional title",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"title": map[string]interface{}{
					"type":        "string",
					"description": "Optional note title",
				},
				"content": map[string]interface{}{
					"type":        "string",
					"description": "Note content",
				},
			},
			"required": []string{"content"},
		},
	}
}

// GetNoteSkill returns the get note skill definition.
func GetNoteSkill() Skill {
	return Skill{
		Name:        "get_note",
		Description: "Retrieve a note by title",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"title": map[string]interface{}{
					"type":        "string",
					"description": "Note title to retrieve",
				},
			},
			"required": []string{"title"},
		},
	}
}

// ListNotesSkill returns the list notes skill definition.
func ListNotesSkill() Skill {
	return Skill{
		Name:        "list_notes",
		Description: "List all saved notes",
		Parameters: map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
			"required":   []string{},
		},
	}
}

// --- Skill Handlers ---

// TarotHandler executes a tarot reading.
func TarotHandler(args map[string]interface{}, configLoader ConfigLoader) (interface{}, error) {
	config, err := configLoader.GetTarotConfig()
	if err != nil {
		return nil, fmt.Errorf("tarot configuration error: %w", err)
	}

	spreadType := "three"
	if st, ok := args["spread_type"].(string); ok {
		spreadType = st
	}

	question := ""
	if q, ok := args["question"].(string); ok {
		question = q
	}

	// Make request to tarot function
	requestBody := map[string]interface{}{
		"spread_type": spreadType,
	}
	if question != "" {
		requestBody["question"] = question
	}

	reqBodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to encode request: %w", err)
	}

	req, err := http.NewRequest("POST", config.FunctionURL, bytes.NewBuffer(reqBodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set Authorization header - token may already include "Basic " prefix
	authToken := config.AuthToken
	if !strings.HasPrefix(authToken, "Basic ") {
		authToken = "Basic " + authToken
	}
	req.Header.Set("Authorization", authToken)
	req.Header.Set("Content-Type", "application/json")

	// Create client with timeout and logging
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Make request with timeout
	startTime := time.Now()
	resp, err := client.Do(req)
	elapsed := time.Since(startTime)
	
	if err != nil {
		return nil, fmt.Errorf("tarot request failed after %v: %w", elapsed, err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response after %v: %w", elapsed, err)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("tarot API error (status %d) after %v: %s", resp.StatusCode, elapsed, string(responseBody))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(responseBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response after %v: %w", elapsed, err)
	}

	return result, nil
}

// NSFWHandler toggles NSFW mode.
func NSFWHandler(args map[string]interface{}, configLoader ConfigLoader) (interface{}, error) {
	enable, ok := args["enable"].(bool)
	if !ok {
		return nil, fmt.Errorf("enable must be a boolean")
	}

	if enable {
		config, err := configLoader.GetVeniceConfig()
		if err != nil {
			return map[string]interface{}{
				"success":       false,
				"error":         "NSFW mode requires Venice.ai API key",
				"requires_setup": true,
			}, nil
		}

		return map[string]interface{}{
			"success": true,
			"enabled": true,
			"message": "NSFW mode enabled",
			"config": map[string]interface{}{
				"baseUrl": config.BaseURL,
				"model":   config.Model,
			},
		}, nil
	}

	return map[string]interface{}{
		"success": true,
		"enabled": false,
		"message": "NSFW mode disabled",
	}, nil
}

// ContentHandler generates content for platforms.
func ContentHandler(args map[string]interface{}) (interface{}, error) {
	platform := "twitter"
	if p, ok := args["platform"].(string); ok {
		platform = p
	}

	format := "short"
	if f, ok := args["format"].(string); ok {
		format = f
	}

	tone := "teasing"
	if t, ok := args["tone"].(string); ok {
		tone = t
	}

	topic := ""
	if t, ok := args["topic"].(string); ok {
		topic = t
	}

	// Build content generation prompt
	prompt := fmt.Sprintf(`Generate %s content for %s.
Topic: %s
Tone: %s
Format: %s length`, platform, platform, topic, tone, format)

	return map[string]interface{}{
		"success":  true,
		"prompt":   prompt,
		"platform": platform,
		"format":   format,
		"tone":     tone,
		"topic":    topic,
		"message":  "Content parameters configured. Ready for generation.",
	}, nil
}

// ImageHandler generates an image using Venice.ai.
func ImageHandler(args map[string]interface{}, configLoader ConfigLoader) (interface{}, error) {
	config, err := configLoader.GetVeniceConfig()
	if err != nil {
		return nil, fmt.Errorf("Venice.ai configuration error: %w", err)
	}

	prompt, ok := args["prompt"].(string)
	if !ok || prompt == "" {
		return nil, fmt.Errorf("prompt is required")
	}

	style := ""
	if s, ok := args["style"].(string); ok {
		style = s
		prompt = fmt.Sprintf("%s, %s style", prompt, style)
	}

	// Make request to Venice.ai
	requestBody := map[string]interface{}{
		"model":  "fluently-xl",
		"prompt": prompt,
		"width":  1024,
		"height": 1024,
		"steps":  30,
	}

	reqBodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to encode request: %w", err)
	}

	req, err := http.NewRequest("POST", config.BaseURL+"/images/generations", bytes.NewBuffer(reqBodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+config.APIKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("image request failed: %w", err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("Venice.ai API error (status %d): %s", resp.StatusCode, string(responseBody))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(responseBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Extract image URL or base64 data
	if data, ok := result["data"].([]interface{}); ok && len(data) > 0 {
		if img, ok := data[0].(map[string]interface{}); ok {
			if url, ok := img["url"].(string); ok {
				return map[string]interface{}{
					"success":   true,
					"image_url": url,
					"prompt":    prompt,
				}, nil
			}
			if b64, ok := img["b64_json"].(string); ok {
				// Save base64 image to file
				filename := fmt.Sprintf("celeste_image_%d.png", time.Now().Unix())
				homeDir, _ := os.UserHomeDir()
				filepath := filepath.Join(homeDir, ".celeste", "images", filename)

				// Ensure directory exists
				os.MkdirAll(filepath[:len(filepath)-len(filename)-1], 0755)

				// Decode and save
				imgData, err := base64.StdEncoding.DecodeString(b64)
				if err == nil {
					os.WriteFile(filepath, imgData, 0644)
					return map[string]interface{}{
						"success":    true,
						"image_path": filepath,
						"prompt":     prompt,
					}, nil
				}
			}
		}
	}

	return result, nil
}

// WeatherHandler gets weather forecast for a location.
func WeatherHandler(args map[string]interface{}, configLoader ConfigLoader) (interface{}, error) {
	config, err := configLoader.GetWeatherConfig()
	if err != nil {
		// If config not available, use empty config (will require zip in args)
		config = WeatherConfig{}
	}

	// Get zip code from args or use default
	zipCode := ""
	if z, ok := args["zip_code"].(string); ok && z != "" {
		zipCode = z
	} else if config.DefaultZipCode != "" {
		zipCode = config.DefaultZipCode
	}

	// If still no zip code, return helpful error that LLM can interpret
	if zipCode == "" {
		return map[string]interface{}{
			"error":   "zip_code_required",
			"message": "A zip code is required for weather lookup. Please provide a 5-digit zip code in your request, or set a default using: celeste config --set-weather-zip <zip>",
			"hint":    "You can ask the user for their zip code or location",
		}, nil
	}

	// Validate zip code format (5 digits)
	if len(zipCode) != 5 {
		return nil, fmt.Errorf("zip code must be 5 digits")
	}
	for _, c := range zipCode {
		if c < '0' || c > '9' {
			return nil, fmt.Errorf("zip code must contain only digits")
		}
	}

	// Get forecast days (default 1 for current weather)
	days := 1
	if d, ok := args["days"].(float64); ok {
		days = int(d)
		if days < 1 {
			days = 1
		}
		if days > 3 {
			days = 3
		}
	}

	// Use wttr.in API (free, no key required)
	// Format: https://wttr.in/{zip}?format=j1 for JSON
	url := fmt.Sprintf("https://wttr.in/%s?format=j1", zipCode)
	if days > 1 {
		url = fmt.Sprintf("https://wttr.in/%s?format=j1&days=%d", zipCode, days)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("weather request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("weather API error (status %d): %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to parse weather response: %w", err)
	}

	// Add zip code to result for reference
	result["zip_code"] = zipCode
	result["requested_days"] = days

	return result, nil
}

// UnitConverterHandler converts between different units.
func UnitConverterHandler(args map[string]interface{}) (interface{}, error) {
	value, ok := args["value"].(float64)
	if !ok {
		return nil, fmt.Errorf("value must be a number")
	}

	fromUnit, ok := args["from_unit"].(string)
	if !ok || fromUnit == "" {
		return nil, fmt.Errorf("from_unit is required")
	}

	toUnit, ok := args["to_unit"].(string)
	if !ok || toUnit == "" {
		return nil, fmt.Errorf("to_unit is required")
	}

	// Convert to lowercase for comparison
	fromUnit = strings.ToLower(fromUnit)
	toUnit = strings.ToLower(toUnit)

	// Length conversions (meters as base)
	lengthConversions := map[string]float64{
		"m":  1.0,
		"km": 1000.0,
		"cm": 0.01,
		"mm": 0.001,
		"ft": 0.3048,
		"in": 0.0254,
		"yd": 0.9144,
		"mi": 1609.34,
	}

	// Weight conversions (kilograms as base)
	weightConversions := map[string]float64{
		"kg": 1.0,
		"g":  0.001,
		"mg": 0.000001,
		"lb": 0.453592,
		"oz": 0.0283495,
	}

	// Volume conversions (liters as base)
	volumeConversions := map[string]float64{
		"l":      1.0,
		"liter":  1.0,
		"ml":     0.001,
		"gallon": 3.78541,
		"quart":  0.946353,
		"pint":   0.473176,
		"cup":    0.236588,
		"fl oz":  0.0295735,
	}

	var result float64
	var category string

	// Check if it's a length conversion
	if fromVal, fromOk := lengthConversions[fromUnit]; fromOk {
		if toVal, toOk := lengthConversions[toUnit]; toOk {
			result = (value * fromVal) / toVal
			category = "length"
		} else {
			return nil, fmt.Errorf("invalid to_unit for length conversion")
		}
	} else if fromVal, fromOk := weightConversions[fromUnit]; fromOk {
		// Check if it's a weight conversion
		if toVal, toOk := weightConversions[toUnit]; toOk {
			result = (value * fromVal) / toVal
			category = "weight"
		} else {
			return nil, fmt.Errorf("invalid to_unit for weight conversion")
		}
	} else if fromVal, fromOk := volumeConversions[fromUnit]; fromOk {
		// Check if it's a volume conversion
		if toVal, toOk := volumeConversions[toUnit]; toOk {
			result = (value * fromVal) / toVal
			category = "volume"
		} else {
			return nil, fmt.Errorf("invalid to_unit for volume conversion")
		}
	} else if strings.Contains(fromUnit, "celsius") || strings.Contains(fromUnit, "fahrenheit") {
		// Temperature conversion
		category = "temperature"
		var celsius float64

		if strings.Contains(fromUnit, "fahrenheit") {
			celsius = (value - 32) * 5 / 9
		} else {
			celsius = value
		}

		if strings.Contains(toUnit, "fahrenheit") {
			result = celsius*9/5 + 32
		} else {
			result = celsius
		}
	} else {
		return nil, fmt.Errorf("unsupported unit conversion from %s to %s", fromUnit, toUnit)
	}

	return map[string]interface{}{
		"value":      result,
		"from_value": value,
		"from_unit":  fromUnit,
		"to_unit":    toUnit,
		"category":   category,
	}, nil
}

// TimezoneConverterHandler converts time between timezones.
func TimezoneConverterHandler(args map[string]interface{}) (interface{}, error) {
	fromTZ, ok := args["from_timezone"].(string)
	if !ok || fromTZ == "" {
		return nil, fmt.Errorf("from_timezone is required")
	}

	toTZ, ok := args["to_timezone"].(string)
	if !ok || toTZ == "" {
		return nil, fmt.Errorf("to_timezone is required")
	}

	// Load timezone locations
	fromLoc, err := time.LoadLocation(fromTZ)
	if err != nil {
		return nil, fmt.Errorf("invalid from_timezone: %w", err)
	}

	toLoc, err := time.LoadLocation(toTZ)
	if err != nil {
		return nil, fmt.Errorf("invalid to_timezone: %w", err)
	}

	// Parse time if provided, otherwise use current time
	var t time.Time
	if timeStr, ok := args["time"].(string); ok && timeStr != "" {
		// Parse time string (HH:MM or HH:MM:SS)
		var dateStr string
		if date, ok := args["date"].(string); ok && date != "" {
			dateStr = date
		} else {
			dateStr = time.Now().In(fromLoc).Format("2006-01-02")
		}

		timeLayout := "2006-01-02 15:04:05"
		if !strings.Contains(timeStr, ":") {
			return nil, fmt.Errorf("invalid time format, expected HH:MM or HH:MM:SS")
		}
		if len(strings.Split(timeStr, ":")[0]) == 1 {
			timeStr = "0" + timeStr
		}
		if len(strings.Split(timeStr, ":")) == 2 {
			timeStr = timeStr + ":00"
		}

		fullTimeStr := dateStr + " " + timeStr
		t, err = time.ParseInLocation(timeLayout, fullTimeStr, fromLoc)
		if err != nil {
			return nil, fmt.Errorf("invalid time format: %w", err)
		}
	} else {
		// Use current time in source timezone
		if date, ok := args["date"].(string); ok && date != "" {
			dateStr := date + " 00:00:00"
			timeLayout := "2006-01-02 15:04:05"
			t, err = time.ParseInLocation(timeLayout, dateStr, fromLoc)
			if err != nil {
				return nil, fmt.Errorf("invalid date format: %w", err)
			}
		} else {
			t = time.Now().In(fromLoc)
		}
	}

	// Convert to target timezone
	converted := t.In(toLoc)

	return map[string]interface{}{
		"original_time":    t.Format("2006-01-02 15:04:05 MST"),
		"converted_time":   converted.Format("2006-01-02 15:04:05 MST"),
		"from_timezone":    fromTZ,
		"to_timezone":      toTZ,
		"original_utc":     t.UTC().Format("2006-01-02 15:04:05 UTC"),
		"converted_utc":     converted.UTC().Format("2006-01-02 15:04:05 UTC"),
		"timezone_offset":  converted.Format("-07:00"),
	}, nil
}

// HashGeneratorHandler generates a hash for the given text.
func HashGeneratorHandler(args map[string]interface{}) (interface{}, error) {
	text, ok := args["text"].(string)
	if !ok || text == "" {
		return nil, fmt.Errorf("text is required")
	}

	algorithm, ok := args["algorithm"].(string)
	if !ok || algorithm == "" {
		return nil, fmt.Errorf("algorithm is required (md5, sha256, sha512)")
	}

	algorithm = strings.ToLower(algorithm)
	var hash string

	switch algorithm {
	case "md5":
		h := md5.Sum([]byte(text))
		hash = hex.EncodeToString(h[:])
	case "sha256":
		h := sha256.Sum256([]byte(text))
		hash = hex.EncodeToString(h[:])
	case "sha512":
		h := sha512.Sum512([]byte(text))
		hash = hex.EncodeToString(h[:])
	default:
		return nil, fmt.Errorf("unsupported algorithm: %s (supported: md5, sha256, sha512)", algorithm)
	}

	return map[string]interface{}{
		"text":      text,
		"algorithm": algorithm,
		"hash":      hash,
	}, nil
}

// Base64EncodeHandler encodes text to base64.
func Base64EncodeHandler(args map[string]interface{}) (interface{}, error) {
	text, ok := args["text"].(string)
	if !ok || text == "" {
		return nil, fmt.Errorf("text is required")
	}

	encoded := base64.StdEncoding.EncodeToString([]byte(text))

	return map[string]interface{}{
		"original": text,
		"encoded":  encoded,
	}, nil
}

// Base64DecodeHandler decodes a base64 string.
func Base64DecodeHandler(args map[string]interface{}) (interface{}, error) {
	encoded, ok := args["encoded"].(string)
	if !ok || encoded == "" {
		return nil, fmt.Errorf("encoded is required")
	}

	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("invalid base64 string: %w", err)
	}

	return map[string]interface{}{
		"encoded":  encoded,
		"decoded":  string(decoded),
	}, nil
}

// UUIDGeneratorHandler generates a UUID v4.
func UUIDGeneratorHandler(args map[string]interface{}) (interface{}, error) {
	id := uuid.New()

	return map[string]interface{}{
		"uuid": id.String(),
	}, nil
}

// PasswordGeneratorHandler generates a secure random password.
func PasswordGeneratorHandler(args map[string]interface{}) (interface{}, error) {
	length := 16
	if l, ok := args["length"].(float64); ok {
		length = int(l)
		if length < 8 {
			length = 8
		}
		if length > 128 {
			length = 128
		}
	}

	includeSymbols := true
	if s, ok := args["include_symbols"].(bool); ok {
		includeSymbols = s
	}

	includeNumbers := true
	if n, ok := args["include_numbers"].(bool); ok {
		includeNumbers = n
	}

	// Character sets
	lowercase := "abcdefghijklmnopqrstuvwxyz"
	uppercase := "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	numbers := "0123456789"
	symbols := "!@#$%^&*()_+-=[]{}|;:,.<>?"

	charset := lowercase + uppercase
	if includeNumbers {
		charset += numbers
	}
	if includeSymbols {
		charset += symbols
	}

	password := make([]byte, length)
	for i := range password {
		b := make([]byte, 1)
		rand.Read(b)
		password[i] = charset[int(b[0])%len(charset)]
	}

	return map[string]interface{}{
		"password":        string(password),
		"length":         length,
		"include_symbols": includeSymbols,
		"include_numbers": includeNumbers,
	}, nil
}

// CurrencyConverterHandler converts between currencies using exchangerate-api.com.
func CurrencyConverterHandler(args map[string]interface{}) (interface{}, error) {
	amount, ok := args["amount"].(float64)
	if !ok {
		return nil, fmt.Errorf("amount must be a number")
	}

	fromCurrency, ok := args["from_currency"].(string)
	if !ok || fromCurrency == "" {
		return nil, fmt.Errorf("from_currency is required")
	}

	toCurrency, ok := args["to_currency"].(string)
	if !ok || toCurrency == "" {
		return nil, fmt.Errorf("to_currency is required")
	}

	// Normalize currency codes to uppercase
	fromCurrency = strings.ToUpper(fromCurrency)
	toCurrency = strings.ToUpper(toCurrency)

	if fromCurrency == toCurrency {
		return map[string]interface{}{
			"amount":        amount,
			"from_currency": fromCurrency,
			"to_currency":   toCurrency,
			"converted":     amount,
			"rate":          1.0,
		}, nil
	}

	// Use exchangerate-api.com free tier (no key required for basic)
	// First get rates for the base currency
	url := fmt.Sprintf("https://api.exchangerate-api.com/v6/latest/%s", fromCurrency)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("currency API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("currency API error (status %d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		Rates map[string]float64 `json:"rates"`
		Base  string             `json:"base"`
		Date  string             `json:"date"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to parse currency response: %w", err)
	}

	rate, ok := result.Rates[toCurrency]
	if !ok {
		return nil, fmt.Errorf("currency %s not found in exchange rates", toCurrency)
	}

	converted := amount * rate

	return map[string]interface{}{
		"amount":        amount,
		"from_currency": fromCurrency,
		"to_currency":   toCurrency,
		"converted":     converted,
		"rate":          rate,
		"date":          result.Date,
	}, nil
}

// QRCodeGeneratorHandler generates a QR code from text.
func QRCodeGeneratorHandler(args map[string]interface{}) (interface{}, error) {
	text, ok := args["text"].(string)
	if !ok || text == "" {
		return nil, fmt.Errorf("text is required")
	}

	size := 256
	if s, ok := args["size"].(float64); ok {
		size = int(s)
		if size < 64 {
			size = 64
		}
		if size > 1024 {
			size = 1024
		}
	}

	// Generate QR code
	qr, err := qrcode.New(text, qrcode.Medium)
	if err != nil {
		return nil, fmt.Errorf("failed to generate QR code: %w", err)
	}

	// Get QR code as PNG bytes
	pngData, err := qr.PNG(size)
	if err != nil {
		return nil, fmt.Errorf("failed to encode QR code as PNG: %w", err)
	}

	// Save to file
	homeDir, _ := os.UserHomeDir()
	qrDir := filepath.Join(homeDir, ".celeste", "qr_codes")
	os.MkdirAll(qrDir, 0755)

	filename := fmt.Sprintf("qr_%d.png", time.Now().Unix())
	filepath := filepath.Join(qrDir, filename)

	if err := os.WriteFile(filepath, pngData, 0644); err != nil {
		return nil, fmt.Errorf("failed to save QR code: %w", err)
	}

	return map[string]interface{}{
		"text":     text,
		"size":     size,
		"filepath": filepath,
		"success":  true,
	}, nil
}

// TwitchLiveCheckHandler checks if a Twitch streamer is live.
func TwitchLiveCheckHandler(args map[string]interface{}, configLoader ConfigLoader) (interface{}, error) {
	config, err := configLoader.GetTwitchConfig()
	if err != nil {
		return nil, fmt.Errorf("Twitch configuration error: %w", err)
	}

	// Get streamer from args or use default
	streamer := ""
	if s, ok := args["streamer"].(string); ok && s != "" {
		streamer = s
	} else {
		streamer = config.DefaultStreamer
	}

	if streamer == "" {
		return nil, fmt.Errorf("streamer required. Set default with 'celeste config --set-twitch-streamer <name>' or provide in your message")
	}

	// Use Twitch Helix API
	url := fmt.Sprintf("https://api.twitch.tv/helix/streams?user_login=%s", streamer)

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Client-ID", config.ClientID)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Twitch API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Twitch API error (status %d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		Data []struct {
			ID           string    `json:"id"`
			UserID       string    `json:"user_id"`
			UserLogin    string    `json:"user_login"`
			UserName     string    `json:"user_name"`
			GameID       string    `json:"game_id"`
			GameName     string    `json:"game_name"`
			Type         string    `json:"type"`
			Title        string    `json:"title"`
			ViewerCount  int       `json:"viewer_count"`
			StartedAt    time.Time `json:"started_at"`
			Language     string    `json:"language"`
			ThumbnailURL string    `json:"thumbnail_url"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to parse Twitch response: %w", err)
	}

	isLive := len(result.Data) > 0

	response := map[string]interface{}{
		"streamer": streamer,
		"is_live":  isLive,
	}

	if isLive {
		stream := result.Data[0]
		response["title"] = stream.Title
		response["game"] = stream.GameName
		response["viewer_count"] = stream.ViewerCount
		response["started_at"] = stream.StartedAt.Format(time.RFC3339)
		response["language"] = stream.Language
		response["thumbnail_url"] = stream.ThumbnailURL
		response["stream_url"] = fmt.Sprintf("https://www.twitch.tv/%s", stream.UserLogin)
	}

	return response, nil
}

// YouTubeVideosHandler gets recent videos from a YouTube channel.
func YouTubeVideosHandler(args map[string]interface{}, configLoader ConfigLoader) (interface{}, error) {
	config, err := configLoader.GetYouTubeConfig()
	if err != nil {
		return nil, fmt.Errorf("YouTube configuration error: %w", err)
	}

	// Get channel from args or use default
	channel := ""
	if c, ok := args["channel"].(string); ok && c != "" {
		channel = c
	} else {
		channel = config.DefaultChannel
	}

	if channel == "" {
		return nil, fmt.Errorf("channel required. Set default with 'celeste config --set-youtube-channel <name>' or provide in your message")
	}

	maxResults := 5
	if m, ok := args["max_results"].(float64); ok {
		maxResults = int(m)
		if maxResults < 1 {
			maxResults = 1
		}
		if maxResults > 50 {
			maxResults = 50
		}
	}

	// First, try to get channel ID if channel is a username
	// YouTube Data API v3 requires channel ID for search
	// We'll try to search by username first, then use channel ID
	channelID := channel

	// If it doesn't look like a channel ID (starts with UC), try to resolve it
	if !strings.HasPrefix(channel, "UC") && len(channel) != 24 {
		// Try to get channel ID from username
		searchURL := fmt.Sprintf("https://www.googleapis.com/youtube/v3/search?part=snippet&q=%s&type=channel&maxResults=1&key=%s", channel, config.APIKey)
		
		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Get(searchURL)
		if err == nil && resp.StatusCode == 200 {
			var searchResult struct {
				Items []struct {
					ID struct {
						ChannelID string `json:"channelId"`
					} `json:"id"`
				} `json:"items"`
			}
			if json.NewDecoder(resp.Body).Decode(&searchResult) == nil && len(searchResult.Items) > 0 {
				channelID = searchResult.Items[0].ID.ChannelID
			}
			resp.Body.Close()
		}
	}

	// Get recent videos
	url := fmt.Sprintf("https://www.googleapis.com/youtube/v3/search?part=snippet&channelId=%s&order=date&type=video&maxResults=%d&key=%s", channelID, maxResults, config.APIKey)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("YouTube API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("YouTube API error (status %d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		Items []struct {
			ID struct {
				VideoID string `json:"videoId"`
			} `json:"id"`
			Snippet struct {
				Title       string    `json:"title"`
				Description string    `json:"description"`
				PublishedAt time.Time `json:"publishedAt"`
				Thumbnails  struct {
					Default struct {
						URL string `json:"url"`
					} `json:"default"`
				} `json:"thumbnails"`
				ChannelTitle string `json:"channelTitle"`
			} `json:"snippet"`
		} `json:"items"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to parse YouTube response: %w", err)
	}

	videos := make([]map[string]interface{}, 0, len(result.Items))
	for _, item := range result.Items {
		videos = append(videos, map[string]interface{}{
			"video_id":    item.ID.VideoID,
			"title":       item.Snippet.Title,
			"description": item.Snippet.Description,
			"published_at": item.Snippet.PublishedAt.Format(time.RFC3339),
			"thumbnail_url": item.Snippet.Thumbnails.Default.URL,
			"channel_title": item.Snippet.ChannelTitle,
			"url": fmt.Sprintf("https://www.youtube.com/watch?v=%s", item.ID.VideoID),
		})
	}

	return map[string]interface{}{
		"channel":    channel,
		"channel_id": channelID,
		"count":      len(videos),
		"videos":     videos,
	}, nil
}

// Reminder represents a reminder entry.
type Reminder struct {
	ID      string    `json:"id"`
	Message string    `json:"message"`
	Time    time.Time `json:"time"`
	Created time.Time `json:"created"`
}

// Note represents a note entry.
type Note struct {
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	Created   time.Time `json:"created"`
	Updated   time.Time `json:"updated"`
}

// getRemindersPath returns the path to reminders.json.
func getRemindersPath() string {
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".celeste", "reminders.json")
}

// getNotesPath returns the path to notes.json.
func getNotesPath() string {
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".celeste", "notes.json")
}

// SetReminderHandler sets a reminder.
func SetReminderHandler(args map[string]interface{}) (interface{}, error) {
	message, ok := args["message"].(string)
	if !ok || message == "" {
		return nil, fmt.Errorf("message is required")
	}

	timeStr, ok := args["time"].(string)
	if !ok || timeStr == "" {
		return nil, fmt.Errorf("time is required")
	}

	// Parse time string
	var reminderTime time.Time
	var err error

	// Try relative time first (e.g., "in 1 hour", "tomorrow at 3pm")
	now := time.Now()
	if strings.HasPrefix(timeStr, "in ") {
		// Simple relative parsing - could be enhanced
		return nil, fmt.Errorf("relative time parsing not yet implemented, please use format 'YYYY-MM-DD HH:MM' or 'HH:MM'")
	}

	// Try full datetime format
	if len(timeStr) > 10 {
		reminderTime, err = time.Parse("2006-01-02 15:04", timeStr)
		if err != nil {
			reminderTime, err = time.Parse("2006-01-02 15:04:05", timeStr)
		}
	} else {
		// Just time, use today
		timeLayout := "15:04"
		if len(strings.Split(timeStr, ":")) == 3 {
			timeLayout = "15:04:05"
		}
		parsedTime, err := time.Parse(timeLayout, timeStr)
		if err != nil {
			return nil, fmt.Errorf("invalid time format: %w", err)
		}
		reminderTime = time.Date(now.Year(), now.Month(), now.Day(), parsedTime.Hour(), parsedTime.Minute(), parsedTime.Second(), 0, now.Location())
		if reminderTime.Before(now) {
			// If time has passed today, set for tomorrow
			reminderTime = reminderTime.Add(24 * time.Hour)
		}
	}

	if err != nil {
		return nil, fmt.Errorf("failed to parse time: %w", err)
	}

	// Load existing reminders
	remindersPath := getRemindersPath()
	var reminders []Reminder
	if data, err := os.ReadFile(remindersPath); err == nil {
		json.Unmarshal(data, &reminders)
	}

	// Add new reminder
	reminder := Reminder{
		ID:      uuid.New().String(),
		Message: message,
		Time:    reminderTime,
		Created: now,
	}
	reminders = append(reminders, reminder)

	// Save reminders
	os.MkdirAll(filepath.Dir(remindersPath), 0755)
	data, err := json.MarshalIndent(reminders, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal reminders: %w", err)
	}
	if err := os.WriteFile(remindersPath, data, 0644); err != nil {
		return nil, fmt.Errorf("failed to save reminders: %w", err)
	}

	return map[string]interface{}{
		"id":      reminder.ID,
		"message": message,
		"time":    reminderTime.Format(time.RFC3339),
		"success": true,
	}, nil
}

// ListRemindersHandler lists all reminders.
func ListRemindersHandler(args map[string]interface{}) (interface{}, error) {
	remindersPath := getRemindersPath()
	var reminders []Reminder
	if data, err := os.ReadFile(remindersPath); err == nil {
		json.Unmarshal(data, &reminders)
	}

	// Filter active reminders (future only)
	now := time.Now()
	activeReminders := make([]map[string]interface{}, 0)
	for _, r := range reminders {
		if r.Time.After(now) {
			activeReminders = append(activeReminders, map[string]interface{}{
				"id":      r.ID,
				"message": r.Message,
				"time":    r.Time.Format(time.RFC3339),
				"created": r.Created.Format(time.RFC3339),
			})
		}
	}

	return map[string]interface{}{
		"count":     len(activeReminders),
		"reminders": activeReminders,
	}, nil
}

// SaveNoteHandler saves a note.
func SaveNoteHandler(args map[string]interface{}) (interface{}, error) {
	content, ok := args["content"].(string)
	if !ok || content == "" {
		return nil, fmt.Errorf("content is required")
	}

	title := ""
	if t, ok := args["title"].(string); ok {
		title = t
	} else {
		// Generate title from first line of content
		lines := strings.Split(content, "\n")
		title = strings.TrimSpace(lines[0])
		if len(title) > 50 {
			title = title[:50] + "..."
		}
	}

	// Load existing notes
	notesPath := getNotesPath()
	var notes map[string]Note
	if data, err := os.ReadFile(notesPath); err == nil {
		json.Unmarshal(data, &notes)
	} else {
		notes = make(map[string]Note)
	}

	// Save or update note
	now := time.Now()
	if existing, exists := notes[title]; exists {
		existing.Content = content
		existing.Updated = now
		notes[title] = existing
	} else {
		notes[title] = Note{
			Title:   title,
			Content: content,
			Created: now,
			Updated: now,
		}
	}

	// Save notes
	os.MkdirAll(filepath.Dir(notesPath), 0755)
	data, err := json.MarshalIndent(notes, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal notes: %w", err)
	}
	if err := os.WriteFile(notesPath, data, 0644); err != nil {
		return nil, fmt.Errorf("failed to save notes: %w", err)
	}

	return map[string]interface{}{
		"title":   title,
		"success": true,
	}, nil
}

// GetNoteHandler retrieves a note.
func GetNoteHandler(args map[string]interface{}) (interface{}, error) {
	title, ok := args["title"].(string)
	if !ok || title == "" {
		return nil, fmt.Errorf("title is required")
	}

	// Load notes
	notesPath := getNotesPath()
	var notes map[string]Note
	if data, err := os.ReadFile(notesPath); err != nil {
		return nil, fmt.Errorf("note not found: %s", title)
	} else {
		if err := json.Unmarshal(data, &notes); err != nil {
			return nil, fmt.Errorf("failed to parse notes: %w", err)
		}
	}

	note, exists := notes[title]
	if !exists {
		return nil, fmt.Errorf("note not found: %s", title)
	}

	return map[string]interface{}{
		"title":   note.Title,
		"content": note.Content,
		"created": note.Created.Format(time.RFC3339),
		"updated": note.Updated.Format(time.RFC3339),
	}, nil
}

// ListNotesHandler lists all notes.
func ListNotesHandler(args map[string]interface{}) (interface{}, error) {
	notesPath := getNotesPath()
	var notes map[string]Note
	if data, err := os.ReadFile(notesPath); err == nil {
		json.Unmarshal(data, &notes)
	} else {
		notes = make(map[string]Note)
	}

	noteList := make([]map[string]interface{}, 0, len(notes))
	for _, note := range notes {
		noteList = append(noteList, map[string]interface{}{
			"title":   note.Title,
			"created": note.Created.Format(time.RFC3339),
			"updated": note.Updated.Format(time.RFC3339),
		})
	}

	return map[string]interface{}{
		"count": len(noteList),
		"notes": noteList,
	}, nil
}

// CreateDefaultSkillFiles creates default skill JSON files in ~/.celeste/skills/
func CreateDefaultSkillFiles() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	skillsDir := filepath.Join(homeDir, ".celeste", "skills")
	if err := os.MkdirAll(skillsDir, 0755); err != nil {
		return err
	}

	skills := []Skill{
		TarotSkill(),
		NSFWSkill(),
		ContentSkill(),
		ImageSkill(),
	}

	for _, skill := range skills {
		data, err := json.MarshalIndent(skill, "", "  ")
		if err != nil {
			continue
		}

		path := filepath.Join(skillsDir, skill.Name+".json")
		if _, err := os.Stat(path); os.IsNotExist(err) {
			os.WriteFile(path, data, 0644)
		}
	}

	return nil
}

