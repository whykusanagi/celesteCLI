package main

import (
	"encoding/json"
	"fmt"
	"os"
)

// ScaffoldingConfig holds the scaffolding configuration
type ScaffoldingConfig struct {
	ContentTypes map[string]ContentType `json:"content_types"`
	ToneExamples map[string]string      `json:"tone_examples"`
	Platforms    map[string]Platform    `json:"platforms"`
	Personas     map[string]Persona     `json:"personas"`
}

// ContentType defines a content type configuration
type ContentType struct {
	Description string `json:"description"`
	Scaffold    string `json:"scaffold"`
	MaxLength   int    `json:"max_length"`
	Platform    string `json:"platform"`
}

// Platform defines platform-specific settings
type Platform struct {
	MaxLength  int      `json:"max_length"`
	Hashtags   []string `json:"hashtags,omitempty"`
	Formatting string   `json:"formatting,omitempty"`
	EmojiUsage string   `json:"emoji_usage"`
}

// Persona defines persona configuration
type Persona struct {
	Description string   `json:"description"`
	Traits      []string `json:"traits"`
}

// loadScaffoldingConfig loads the scaffolding configuration from JSON file
func loadScaffoldingConfig() (*ScaffoldingConfig, error) {
	// Try to load from scaffolding.json in the same directory as the binary
	configPath := "scaffolding.json"
	if _, err := os.Stat(configPath); err != nil {
		// If not found, try to load from embedded data or use defaults
		return getDefaultScaffoldingConfig(), nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read scaffolding.json: %v", err)
	}

	var config ScaffoldingConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse scaffolding.json: %v", err)
	}

	return &config, nil
}

// getDefaultScaffoldingConfig returns default scaffolding configuration
func getDefaultScaffoldingConfig() *ScaffoldingConfig {
	return &ScaffoldingConfig{
		ContentTypes: map[string]ContentType{
			"tweet": {
				Description: "Write a post for X/Twitter",
				Scaffold:    "🐦 Write a Twitter post in CelesteAI's voice. She's teasing, smug, and irresistible. Use 1–2 emojis per sentence. End with a strong hook or CTA. Hashtags: #CelesteAI #KusanagiAbyss #VTuberEN.",
				MaxLength:   280,
				Platform:    "twitter",
			},
			"title": {
				Description: "YouTube or Twitch stream title",
				Scaffold:    "📺 Write a punchy, chaotic stream title in CelesteAI's voice. She's teasing, smug, and irresistible. Keep it under 140 characters. Use 1–2 emojis. Make it hype and engaging.",
				MaxLength:   140,
				Platform:    "streaming",
			},
		},
		ToneExamples: map[string]string{
			"lewd":       "suggestive and teasing",
			"explicit":   "direct and uncensored",
			"teasing":    "playful and mischievous",
			"chaotic":    "wild and unpredictable",
			"cute":       "sweet and endearing",
			"official":   "professional and formal",
			"dramatic":   "intense and emotional",
			"parody":     "humorous and satirical",
			"funny":      "comedy and entertainment",
			"suggestive": "hinting and playful",
			"adult":      "mature and sophisticated",
			"sweet":      "gentle and caring",
			"snarky":     "sarcastic and witty",
			"playful":    "fun and lighthearted",
			"hype":       "energetic and exciting",
		},
		Platforms: map[string]Platform{
			"twitter": {
				MaxLength:  280,
				Hashtags:   []string{"#CelesteAI", "#KusanagiAbyss", "#VTuberEN"},
				EmojiUsage: "1-2 per sentence",
			},
			"tiktok": {
				MaxLength:  2200,
				Hashtags:   []string{"#CelesteAI", "#VTuber", "#Anime"},
				EmojiUsage: "1-2 per sentence",
			},
			"youtube": {
				MaxLength:  5000,
				Formatting: "markdown",
				EmojiUsage: "1-2 per paragraph",
			},
			"discord": {
				MaxLength:  2000,
				Formatting: "markdown",
				EmojiUsage: "1-2 per sentence",
			},
			"streaming": {
				MaxLength:  140,
				EmojiUsage: "1-2 total",
			},
		},
		Personas: map[string]Persona{
			"celeste_stream": {
				Description: "Default streaming persona",
				Traits:      []string{"teasing", "smug", "mischievous", "playful"},
			},
			"celeste_ad_read": {
				Description: "Advertisement reading persona",
				Traits:      []string{"wink-and-nudge", "promotional", "engaging"},
			},
			"celeste_moderation_warning": {
				Description: "Moderation warning persona",
				Traits:      []string{"authoritative", "clear", "firm but fair"},
			},
		},
	}
}

// getScaffoldPrompt generates a scaffold prompt based on content type and configuration
func getScaffoldPrompt(contentType, game, tone string, config *ScaffoldingConfig) string {
	contentTypeConfig, exists := config.ContentTypes[contentType]
	if !exists {
		// Fallback to default tweet scaffold
		contentTypeConfig = config.ContentTypes["tweet"]
	}

	scaffold := contentTypeConfig.Scaffold

	if game != "" {
		scaffold += fmt.Sprintf("\nGame: %s.", game)
	}

	if tone != "" {
		scaffold += fmt.Sprintf(" Tone: %s.", tone)
	}

	return scaffold
}
