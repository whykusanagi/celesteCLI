package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// ScaffoldingConfig holds the scaffolding configuration
type ScaffoldingConfig struct {
	Formats          map[string]Format          `json:"formats"`
	ToneExamples     map[string]string          `json:"tone_examples"`
	Platforms        map[string]Platform         `json:"platforms"`
	Personas         map[string]Persona          `json:"personas"`
	ScaffoldTemplates map[string]string         `json:"scaffold_templates"`
}

// Format defines a format configuration
type Format struct {
	MaxLength int      `json:"max_length"`
	Scaffold  string   `json:"scaffold"`
	Platforms []string `json:"platforms"`
}

// Platform defines platform-specific settings
type Platform struct {
	Hashtags     []string `json:"hashtags,omitempty"`
	Formatting   string   `json:"formatting,omitempty"`
	EmojiUsage   string   `json:"emoji_usage"`
	Instructions string   `json:"instructions"`
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
		Formats: map[string]Format{
			"short": {
				MaxLength: 280,
				Scaffold:  "Write a short post in CelesteAI's voice. She's teasing, smug, and irresistible. {platform_instructions} {topic_instruction} {request_instruction} {tone_instruction}",
				Platforms: []string{"twitter", "tiktok"},
			},
			"long": {
				MaxLength: 5000,
				Scaffold:  "Write a detailed description in CelesteAI's voice. She's teasing, smug, and irresistible. {platform_instructions} {topic_instruction} {request_instruction} {tone_instruction}",
				Platforms: []string{"youtube"},
			},
			"general": {
				MaxLength: 2000,
				Scaffold:  "Write content in CelesteAI's voice. She's teasing, smug, and irresistible. {topic_instruction} {request_instruction} {tone_instruction}",
				Platforms: []string{"general"},
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
				Hashtags:     []string{"#CelesteAI", "#KusanagiAbyss", "#VTuberEN"},
				EmojiUsage:   "1-2 per sentence",
				Instructions: "Use 1-2 emojis per sentence. End with hashtags: #CelesteAI #KusanagiAbyss #VTuberEN.",
			},
			"tiktok": {
				Hashtags:     []string{"#CelesteAI", "#VTuber", "#Anime"},
				EmojiUsage:   "1-2 per sentence",
				Instructions: "Use compact line breaks, include 1-2 emojis per sentence, and end with relevant hashtags.",
			},
			"youtube": {
				Formatting:   "markdown",
				EmojiUsage:   "1-2 per paragraph",
				Instructions: "Use markdown formatting. Include timestamps, links to website/socials/products. Use 1-2 emojis per paragraph.",
			},
			"discord": {
				Formatting:   "markdown",
				EmojiUsage:   "1-2 per sentence",
				Instructions: "Use markdown formatting. Use 1-2 emojis per sentence. Make it hype and engaging.",
			},
		},
		ScaffoldTemplates: map[string]string{
			"topic_instruction":   "Topic: {topic}",
			"request_instruction": "Specific instructions: {request}",
			"tone_instruction":    "Tone: {tone}",
			"context_instruction": "Additional context: {context}",
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

// getScaffoldPrompt generates a scaffold prompt based on format, platform, topic, request, tone, and context
func getScaffoldPrompt(format, platform, topic, request, tone, context string, config *ScaffoldingConfig) string {
	// Get format config (default to "short" if not found)
	formatConfig, exists := config.Formats[format]
	if !exists {
		formatConfig = config.Formats["short"]
	}

	scaffold := formatConfig.Scaffold

	// Get platform instructions
	platformInstructions := ""
	if platform != "" {
		if platformConfig, ok := config.Platforms[platform]; ok {
			platformInstructions = platformConfig.Instructions
		}
	}

	// Build topic instruction
	topicInstruction := ""
	if topic != "" {
		topicTemplate := config.ScaffoldTemplates["topic_instruction"]
		if topicTemplate == "" {
			topicTemplate = "Topic: {topic}"
		}
		topicInstruction = strings.ReplaceAll(topicTemplate, "{topic}", topic)
	}

	// Build request instruction
	requestInstruction := ""
	if request != "" {
		requestTemplate := config.ScaffoldTemplates["request_instruction"]
		if requestTemplate == "" {
			requestTemplate = "Specific instructions: {request}"
		}
		requestInstruction = strings.ReplaceAll(requestTemplate, "{request}", request)
	}

	// Build tone instruction
	toneInstruction := ""
	if tone != "" {
		toneTemplate := config.ScaffoldTemplates["tone_instruction"]
		if toneTemplate == "" {
			toneTemplate = "Tone: {tone}"
		}
		toneDesc := config.ToneExamples[tone]
		if toneDesc != "" {
			toneInstruction = strings.ReplaceAll(toneTemplate, "{tone}", toneDesc)
		} else {
			toneInstruction = strings.ReplaceAll(toneTemplate, "{tone}", tone)
		}
	}

	// Build context instruction
	contextInstruction := ""
	if context != "" {
		contextTemplate := config.ScaffoldTemplates["context_instruction"]
		if contextTemplate == "" {
			contextTemplate = "Additional context: {context}"
		}
		contextInstruction = strings.ReplaceAll(contextTemplate, "{context}", context)
	}

	// Replace template variables
	scaffold = strings.ReplaceAll(scaffold, "{platform_instructions}", platformInstructions)
	scaffold = strings.ReplaceAll(scaffold, "{topic_instruction}", topicInstruction)
	scaffold = strings.ReplaceAll(scaffold, "{request_instruction}", requestInstruction)
	scaffold = strings.ReplaceAll(scaffold, "{tone_instruction}", toneInstruction)
	scaffold = strings.ReplaceAll(scaffold, "{context_instruction}", contextInstruction)

	// Clean up extra whitespace (multiple spaces, empty lines)
	scaffold = strings.TrimSpace(scaffold)
	scaffold = strings.ReplaceAll(scaffold, "  ", " ")
	scaffold = strings.ReplaceAll(scaffold, "\n\n\n", "\n\n")

	return scaffold
}
