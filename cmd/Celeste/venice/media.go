// Package venice provides Venice.ai API integration for media generation.
// This handles image, video, upscaling, and image-to-video generation
// that cannot work through function calling (uncensored models).
package venice

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Config holds Venice.ai API configuration.
type Config struct {
	APIKey  string
	BaseURL string
	Model   string // For image generation: fluently-xl, pixart-a, etc.
}

// MediaRequest represents a media generation request.
type MediaRequest struct {
	Type   string // "image", "video", "upscale", "image-to-video"
	Prompt string
	Params map[string]interface{}
}

// MediaResponse represents the response from media generation.
type MediaResponse struct {
	Success   bool   `json:"success"`
	URL       string `json:"url,omitempty"`
	Path      string `json:"path,omitempty"`
	Error     string `json:"error,omitempty"`
	MediaType string `json:"media_type"`
}

// GenerateImage generates an image using Venice.ai.
func GenerateImage(config Config, prompt string, params map[string]interface{}) (*MediaResponse, error) {
	url := config.BaseURL + "/images/generations"

	// Default parameters
	model := "fluently-xl" // Default NSFW-capable model
	if m, ok := params["model"].(string); ok {
		model = m
	}

	width := 1024
	if w, ok := params["width"].(int); ok {
		width = w
	}

	height := 1024
	if h, ok := params["height"].(int); ok {
		height = h
	}

	steps := 30
	if s, ok := params["steps"].(int); ok {
		steps = s
	}

	// Build request payload
	payload := map[string]interface{}{
		"model":  model,
		"prompt": prompt,
		"width":  width,
		"height": height,
		"steps":  steps,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to encode request: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+config.APIKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != 200 {
		return &MediaResponse{
			Success:   false,
			Error:     fmt.Sprintf("API error (status %d): %s", resp.StatusCode, string(body)),
			MediaType: "image",
		}, nil
	}

	// Parse response
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Extract image URL or base64
	if data, ok := result["data"].([]interface{}); ok && len(data) > 0 {
		if img, ok := data[0].(map[string]interface{}); ok {
			// Check for URL
			if imageURL, ok := img["url"].(string); ok {
				return &MediaResponse{
					Success:   true,
					URL:       imageURL,
					MediaType: "image",
				}, nil
			}
			// Check for base64
			if b64, ok := img["b64_json"].(string); ok {
				// Save to file
				path, err := saveBase64Image(b64, "image")
				if err != nil {
					return nil, fmt.Errorf("failed to save image: %w", err)
				}
				return &MediaResponse{
					Success:   true,
					Path:      path,
					MediaType: "image",
				}, nil
			}
		}
	}

	return &MediaResponse{
		Success:   false,
		Error:     "No image data in response",
		MediaType: "image",
	}, nil
}

// UpscaleImage upscales an image using Venice.ai.
func UpscaleImage(config Config, imagePath string, params map[string]interface{}) (*MediaResponse, error) {
	url := config.BaseURL + "/images/upscale"

	// Read image file
	imageData, err := os.ReadFile(imagePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read image: %w", err)
	}

	// Convert to base64
	imageBase64 := base64.StdEncoding.EncodeToString(imageData)

	// Default parameters
	scale := 2
	if s, ok := params["scale"].(int); ok {
		scale = s
	}

	creativity := 0.5
	if c, ok := params["creativity"].(float64); ok {
		creativity = c
	}

	// Build request payload
	payload := map[string]interface{}{
		"image":             imageBase64,
		"scale":             scale,
		"enhance":           true,
		"enhanceCreativity": creativity,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to encode request: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+config.APIKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != 200 {
		return &MediaResponse{
			Success:   false,
			Error:     fmt.Sprintf("API error (status %d): %s", resp.StatusCode, string(body)),
			MediaType: "upscale",
		}, nil
	}

	// Parse response
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Extract upscaled image
	if imageURL, ok := result["url"].(string); ok {
		return &MediaResponse{
			Success:   true,
			URL:       imageURL,
			MediaType: "upscale",
		}, nil
	}

	if b64, ok := result["image"].(string); ok {
		path, err := saveBase64Image(b64, "upscale")
		if err != nil {
			return nil, fmt.Errorf("failed to save upscaled image: %w", err)
		}
		return &MediaResponse{
			Success:   true,
			Path:      path,
			MediaType: "upscale",
		}, nil
	}

	return &MediaResponse{
		Success:   false,
		Error:     "No image data in response",
		MediaType: "upscale",
	}, nil
}

// GenerateVideo generates a video using Venice.ai.
func GenerateVideo(config Config, prompt string, params map[string]interface{}) (*MediaResponse, error) {
	url := config.BaseURL + "/videos/generations"

	// Default parameters
	duration := 5
	if d, ok := params["duration"].(int); ok {
		duration = d
	}

	fps := 24
	if f, ok := params["fps"].(int); ok {
		fps = f
	}

	// Build request payload
	payload := map[string]interface{}{
		"prompt":   prompt,
		"duration": duration,
		"fps":      fps,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to encode request: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+config.APIKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 180 * time.Second} // Longer timeout for video
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != 200 {
		return &MediaResponse{
			Success:   false,
			Error:     fmt.Sprintf("API error (status %d): %s", resp.StatusCode, string(body)),
			MediaType: "video",
		}, nil
	}

	// Parse response
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Extract video URL
	if videoURL, ok := result["url"].(string); ok {
		return &MediaResponse{
			Success:   true,
			URL:       videoURL,
			MediaType: "video",
		}, nil
	}

	return &MediaResponse{
		Success:   false,
		Error:     "No video URL in response",
		MediaType: "video",
	}, nil
}

// ImageToVideo converts an image to video using Venice.ai.
func ImageToVideo(config Config, imagePath string, params map[string]interface{}) (*MediaResponse, error) {
	url := config.BaseURL + "/videos/image-to-video"

	// Read image file
	imageData, err := os.ReadFile(imagePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read image: %w", err)
	}

	// Convert to base64
	imageBase64 := base64.StdEncoding.EncodeToString(imageData)

	// Default parameters
	duration := 5
	if d, ok := params["duration"].(int); ok {
		duration = d
	}

	motion := "medium"
	if m, ok := params["motion"].(string); ok {
		motion = m
	}

	// Build request payload
	payload := map[string]interface{}{
		"image":    imageBase64,
		"duration": duration,
		"motion":   motion,
	}

	if prompt, ok := params["prompt"].(string); ok && prompt != "" {
		payload["prompt"] = prompt
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to encode request: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+config.APIKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 180 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != 200 {
		return &MediaResponse{
			Success:   false,
			Error:     fmt.Sprintf("API error (status %d): %s", resp.StatusCode, string(body)),
			MediaType: "image-to-video",
		}, nil
	}

	// Parse response
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Extract video URL
	if videoURL, ok := result["url"].(string); ok {
		return &MediaResponse{
			Success:   true,
			URL:       videoURL,
			MediaType: "image-to-video",
		}, nil
	}

	return &MediaResponse{
		Success:   false,
		Error:     "No video URL in response",
		MediaType: "image-to-video",
	}, nil
}

// saveBase64Image saves a base64 encoded image to disk.
func saveBase64Image(b64 string, prefix string) (string, error) {
	// Decode base64
	imgData, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64: %w", err)
	}

	// Create output directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	outputDir := filepath.Join(homeDir, ".celeste", "media")
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", err
	}

	// Generate filename
	timestamp := time.Now().Unix()
	filename := fmt.Sprintf("%s_%d.png", prefix, timestamp)
	outputPath := filepath.Join(outputDir, filename)

	// Write file
	if err := os.WriteFile(outputPath, imgData, 0644); err != nil {
		return "", err
	}

	return outputPath, nil
}

// ParseMediaCommand parses a message for media generation commands.
// Returns (type, prompt, params, isMediaCommand)
func ParseMediaCommand(message string) (string, string, map[string]interface{}, bool) {
	message = strings.TrimSpace(message)

	// Check for media prefixes
	prefixes := map[string]string{
		"image:":          "image",
		"video:":          "video",
		"upscale:":        "upscale",
		"image-to-video:": "image-to-video",
		"i2v:":            "image-to-video",
	}

	for prefix, mediaType := range prefixes {
		if strings.HasPrefix(strings.ToLower(message), prefix) {
			// Extract prompt/path after prefix
			content := strings.TrimSpace(message[len(prefix):])
			params := make(map[string]interface{})

			// For upscale and i2v, first word is file path
			if mediaType == "upscale" || mediaType == "image-to-video" {
				parts := strings.SplitN(content, " ", 2)
				if len(parts) > 0 {
					params["path"] = parts[0]
					if len(parts) > 1 {
						content = parts[1] // Rest is additional prompt/params
					} else {
						content = ""
					}
				}
			}

			return mediaType, content, params, true
		}
	}

	return "", "", nil, false
}
