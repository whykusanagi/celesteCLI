// Package skills provides the skill registry and execution system.
// This file contains built-in skill implementations.
package skills

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// RegisterBuiltinSkills registers all built-in skills with the registry.
func RegisterBuiltinSkills(registry *Registry, configLoader ConfigLoader) {
	// Register skill definitions
	registry.RegisterSkill(TarotSkill())
	registry.RegisterSkill(NSFWSkill())
	registry.RegisterSkill(ContentSkill())
	registry.RegisterSkill(ImageSkill())

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
}

// ConfigLoader provides access to configuration values.
type ConfigLoader interface {
	GetTarotConfig() (TarotConfig, error)
	GetVeniceConfig() (VeniceConfig, error)
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

	req.Header.Set("Authorization", "Basic "+config.AuthToken)
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

