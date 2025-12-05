// Package config provides configuration management for Celeste CLI.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/whykusanagi/celesteCLI/cmd/Celeste/skills"
)

// Config holds all configuration for Celeste CLI.
type Config struct {
	// API settings
	APIKey  string `json:"api_key"`
	BaseURL string `json:"base_url"`
	Model   string `json:"model"`
	Timeout int    `json:"timeout"` // seconds

	// Persona settings
	SkipPersonaPrompt bool `json:"skip_persona_prompt"`

	// Streaming settings
	SimulateTyping bool `json:"simulate_typing"`
	TypingSpeed    int  `json:"typing_speed"` // chars per second

	// Venice.ai settings (for NSFW mode)
	VeniceAPIKey     string `json:"venice_api_key,omitempty"`
	VeniceBaseURL    string `json:"venice_base_url,omitempty"`
	VeniceModel      string `json:"venice_model,omitempty"`       // Chat model (venice-uncensored)
	VeniceImageModel string `json:"venice_image_model,omitempty"` // Image model (lustify-sdxl)

	// Tarot settings
	TarotFunctionURL string `json:"tarot_function_url,omitempty"`
	TarotAuthToken   string `json:"tarot_auth_token,omitempty"`

	// Twitter settings
	TwitterBearerToken       string `json:"twitter_bearer_token,omitempty"`
	TwitterAPIKey            string `json:"twitter_api_key,omitempty"`
	TwitterAPISecret         string `json:"twitter_api_secret,omitempty"`
	TwitterAccessToken       string `json:"twitter_access_token,omitempty"`
	TwitterAccessTokenSecret string `json:"twitter_access_token_secret,omitempty"`

	// Weather settings
	WeatherDefaultZipCode string `json:"weather_default_zip_code,omitempty"`

	// Twitch settings
	TwitchClientID        string `json:"twitch_client_id,omitempty"`
	TwitchClientSecret    string `json:"twitch_client_secret,omitempty"`
	TwitchDefaultStreamer string `json:"twitch_default_streamer,omitempty"`

	// YouTube settings
	YouTubeAPIKey         string `json:"youtube_api_key,omitempty"`
	YouTubeDefaultChannel string `json:"youtube_default_channel,omitempty"`
}

// DefaultConfig returns a config with default values.
func DefaultConfig() *Config {
	return &Config{
		BaseURL:           "https://api.openai.com/v1",
		Model:             "gpt-4o-mini",
		Timeout:           60,
		SkipPersonaPrompt: false,
		SimulateTyping:    true,
		TypingSpeed:       40,
		VeniceBaseURL:     "https://api.venice.ai/api/v1",
		VeniceModel:       "venice-uncensored",
	}
}

// Paths returns the configuration directory and file paths.
func Paths() (configDir, configFile, secretsFile, skillsFile string) {
	homeDir, _ := os.UserHomeDir()
	configDir = filepath.Join(homeDir, ".celeste")
	configFile = filepath.Join(configDir, "config.json")
	secretsFile = filepath.Join(configDir, "secrets.json")
	skillsFile = filepath.Join(configDir, "skills.json")
	return
}

// NamedConfigPath returns the path for a named config file.
// If name is empty, returns the default config path.
func NamedConfigPath(name string) string {
	homeDir, _ := os.UserHomeDir()
	configDir := filepath.Join(homeDir, ".celeste")
	if name == "" {
		return filepath.Join(configDir, "config.json")
	}
	return filepath.Join(configDir, fmt.Sprintf("config.%s.json", name))
}

// LoadSkillsConfig loads skill-specific configuration from skills.json.
func LoadSkillsConfig() (*Config, error) {
	_, _, _, skillsFile := Paths()

	skillsConfig := &Config{}

	// Load skills.json if it exists
	if data, err := os.ReadFile(skillsFile); err == nil {
		if err := json.Unmarshal(data, skillsConfig); err != nil {
			return nil, fmt.Errorf("failed to parse skills config: %w", err)
		}
	}

	return skillsConfig, nil
}

// SaveSkillsConfig saves skill-specific configuration to skills.json.
func SaveSkillsConfig(skillsConfig *Config) error {
	_, _, _, skillsFile := Paths()

	// Create skills config with only skill-related fields
	skillsOnly := &Config{
		VeniceAPIKey:             skillsConfig.VeniceAPIKey,
		VeniceBaseURL:            skillsConfig.VeniceBaseURL,
		VeniceModel:              skillsConfig.VeniceModel,
		TarotFunctionURL:         skillsConfig.TarotFunctionURL,
		TarotAuthToken:           skillsConfig.TarotAuthToken,
		TwitterBearerToken:       skillsConfig.TwitterBearerToken,
		TwitterAPIKey:            skillsConfig.TwitterAPIKey,
		TwitterAPISecret:         skillsConfig.TwitterAPISecret,
		TwitterAccessToken:       skillsConfig.TwitterAccessToken,
		TwitterAccessTokenSecret: skillsConfig.TwitterAccessTokenSecret,
		WeatherDefaultZipCode:    skillsConfig.WeatherDefaultZipCode,
		TwitchClientID:           skillsConfig.TwitchClientID,
		TwitchDefaultStreamer:    skillsConfig.TwitchDefaultStreamer,
		YouTubeAPIKey:            skillsConfig.YouTubeAPIKey,
		YouTubeDefaultChannel:    skillsConfig.YouTubeDefaultChannel,
	}

	data, err := json.MarshalIndent(skillsOnly, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal skills config: %w", err)
	}

	return os.WriteFile(skillsFile, data, 0600) // Restrictive permissions for secrets
}

// LoadNamed loads configuration from a named config file.
// If name is empty, loads the default config.
func LoadNamed(name string) (*Config, error) {
	if name == "" {
		return Load()
	}

	config := DefaultConfig()
	configPath := NamedConfigPath(name)

	// Load named config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("config '%s' not found at %s: %w", name, configPath, err)
	}

	if err := json.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse config '%s': %w", name, err)
	}

	// Load shared skills.json (for all skill configurations)
	if skillsConfig, err := LoadSkillsConfig(); err == nil {
		// Merge skill configs (skills.json takes precedence if set)
		if skillsConfig.VeniceAPIKey != "" {
			config.VeniceAPIKey = skillsConfig.VeniceAPIKey
		}
		if skillsConfig.VeniceBaseURL != "" {
			config.VeniceBaseURL = skillsConfig.VeniceBaseURL
		}
		if skillsConfig.VeniceModel != "" {
			config.VeniceModel = skillsConfig.VeniceModel
		}
		if skillsConfig.TarotFunctionURL != "" {
			config.TarotFunctionURL = skillsConfig.TarotFunctionURL
		}
		if skillsConfig.TarotAuthToken != "" {
			config.TarotAuthToken = skillsConfig.TarotAuthToken
		}
		if skillsConfig.TwitterBearerToken != "" {
			config.TwitterBearerToken = skillsConfig.TwitterBearerToken
		}
		if skillsConfig.TwitterAPIKey != "" {
			config.TwitterAPIKey = skillsConfig.TwitterAPIKey
		}
		if skillsConfig.TwitterAPISecret != "" {
			config.TwitterAPISecret = skillsConfig.TwitterAPISecret
		}
		if skillsConfig.TwitterAccessToken != "" {
			config.TwitterAccessToken = skillsConfig.TwitterAccessToken
		}
		if skillsConfig.TwitterAccessTokenSecret != "" {
			config.TwitterAccessTokenSecret = skillsConfig.TwitterAccessTokenSecret
		}
		if skillsConfig.WeatherDefaultZipCode != "" {
			config.WeatherDefaultZipCode = skillsConfig.WeatherDefaultZipCode
		}
		if skillsConfig.TwitchClientID != "" {
			config.TwitchClientID = skillsConfig.TwitchClientID
		}
		if skillsConfig.TwitchClientSecret != "" {
			config.TwitchClientSecret = skillsConfig.TwitchClientSecret
		}
		if skillsConfig.TwitchDefaultStreamer != "" {
			config.TwitchDefaultStreamer = skillsConfig.TwitchDefaultStreamer
		}
		if skillsConfig.YouTubeAPIKey != "" {
			config.YouTubeAPIKey = skillsConfig.YouTubeAPIKey
		}
		if skillsConfig.YouTubeDefaultChannel != "" {
			config.YouTubeDefaultChannel = skillsConfig.YouTubeDefaultChannel
		}
	}

	// Override with environment variables
	if apiKey := os.Getenv("CELESTE_API_KEY"); apiKey != "" {
		config.APIKey = apiKey
	}
	if endpoint := os.Getenv("CELESTE_API_ENDPOINT"); endpoint != "" {
		config.BaseURL = endpoint
	}
	if veniceKey := os.Getenv("VENICE_API_KEY"); veniceKey != "" {
		config.VeniceAPIKey = veniceKey
	}
	if tarotToken := os.Getenv("TAROT_AUTH_TOKEN"); tarotToken != "" {
		config.TarotAuthToken = tarotToken
	}

	return config, nil
}

// ListConfigs returns all available config names.
func ListConfigs() ([]string, error) {
	configDir, _, _, _ := Paths()

	entries, err := os.ReadDir(configDir)
	if err != nil {
		return nil, err
	}

	var configs []string
	for _, entry := range entries {
		name := entry.Name()
		if name == "config.json" {
			configs = append(configs, "default")
		} else if len(name) > 12 && name[:7] == "config." && name[len(name)-5:] == ".json" {
			// Extract name from config.<name>.json
			configName := name[7 : len(name)-5]
			configs = append(configs, configName)
		}
	}

	return configs, nil
}

// Load loads configuration from file and environment.
func Load() (*Config, error) {
	config := DefaultConfig()
	configDir, configFile, secretsFile, _ := Paths()

	// Ensure config directory exists
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	// Load main config file
	if data, err := os.ReadFile(configFile); err == nil {
		if err := json.Unmarshal(data, config); err != nil {
			return nil, fmt.Errorf("failed to parse config: %w", err)
		}
	}

	// Load secrets file (for API keys - backward compatibility)
	if data, err := os.ReadFile(secretsFile); err == nil {
		var secrets Config
		if err := json.Unmarshal(data, &secrets); err == nil {
			if secrets.APIKey != "" {
				config.APIKey = secrets.APIKey
			}
		}
	}

	// Load skills.json (shared across all configs)
	if skillsConfig, err := LoadSkillsConfig(); err == nil {
		// Merge skill configs
		if skillsConfig.VeniceAPIKey != "" {
			config.VeniceAPIKey = skillsConfig.VeniceAPIKey
		}
		if skillsConfig.VeniceBaseURL != "" {
			config.VeniceBaseURL = skillsConfig.VeniceBaseURL
		}
		if skillsConfig.VeniceModel != "" {
			config.VeniceModel = skillsConfig.VeniceModel
		}
		if skillsConfig.TarotFunctionURL != "" {
			config.TarotFunctionURL = skillsConfig.TarotFunctionURL
		}
		if skillsConfig.TarotAuthToken != "" {
			config.TarotAuthToken = skillsConfig.TarotAuthToken
		}
		if skillsConfig.TwitterBearerToken != "" {
			config.TwitterBearerToken = skillsConfig.TwitterBearerToken
		}
		if skillsConfig.TwitterAPIKey != "" {
			config.TwitterAPIKey = skillsConfig.TwitterAPIKey
		}
		if skillsConfig.TwitterAPISecret != "" {
			config.TwitterAPISecret = skillsConfig.TwitterAPISecret
		}
		if skillsConfig.TwitterAccessToken != "" {
			config.TwitterAccessToken = skillsConfig.TwitterAccessToken
		}
		if skillsConfig.TwitterAccessTokenSecret != "" {
			config.TwitterAccessTokenSecret = skillsConfig.TwitterAccessTokenSecret
		}
		if skillsConfig.WeatherDefaultZipCode != "" {
			config.WeatherDefaultZipCode = skillsConfig.WeatherDefaultZipCode
		}
		if skillsConfig.TwitchClientID != "" {
			config.TwitchClientID = skillsConfig.TwitchClientID
		}
		if skillsConfig.TwitchClientSecret != "" {
			config.TwitchClientSecret = skillsConfig.TwitchClientSecret
		}
		if skillsConfig.TwitchDefaultStreamer != "" {
			config.TwitchDefaultStreamer = skillsConfig.TwitchDefaultStreamer
		}
		if skillsConfig.YouTubeAPIKey != "" {
			config.YouTubeAPIKey = skillsConfig.YouTubeAPIKey
		}
		if skillsConfig.YouTubeDefaultChannel != "" {
			config.YouTubeDefaultChannel = skillsConfig.YouTubeDefaultChannel
		}
	}

	// Override with environment variables
	if apiKey := os.Getenv("CELESTE_API_KEY"); apiKey != "" {
		config.APIKey = apiKey
	}
	if endpoint := os.Getenv("CELESTE_API_ENDPOINT"); endpoint != "" {
		config.BaseURL = endpoint
	}
	if veniceKey := os.Getenv("VENICE_API_KEY"); veniceKey != "" {
		config.VeniceAPIKey = veniceKey
	}
	if tarotToken := os.Getenv("TAROT_AUTH_TOKEN"); tarotToken != "" {
		config.TarotAuthToken = tarotToken
	}

	return config, nil
}

// Save saves configuration to file.
func Save(config *Config) error {
	_, configFile, _, _ := Paths()

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	return os.WriteFile(configFile, data, 0644)
}

// SaveSecrets saves API key to secrets file (backward compatibility).
func SaveSecrets(config *Config) error {
	_, _, secretsFile, _ := Paths()

	secrets := &Config{
		APIKey: config.APIKey, // Only API key in secrets.json now
	}

	data, err := json.MarshalIndent(secrets, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal secrets: %w", err)
	}

	return os.WriteFile(secretsFile, data, 0600) // More restrictive permissions for secrets
}

// ConfigLoader implements skills.ConfigLoader interface.
type ConfigLoader struct {
	config *Config
}

// NewConfigLoader creates a new config loader.
func NewConfigLoader(config *Config) *ConfigLoader {
	return &ConfigLoader{config: config}
}

// GetTarotConfig returns tarot configuration.
func (l *ConfigLoader) GetTarotConfig() (skills.TarotConfig, error) {
	if l.config.TarotAuthToken == "" {
		return skills.TarotConfig{}, fmt.Errorf("tarot auth token not configured")
	}

	url := l.config.TarotFunctionURL
	if url == "" {
		url = "https://faas-nyc1-2ef2e6cc.doserverless.co/api/v1/namespaces/fn-30b193db-d334-4dab-b5cd-ab49067f88cc/actions/tarot/logic?blocking=true&result=true"
	}

	return skills.TarotConfig{
		FunctionURL: url,
		AuthToken:   l.config.TarotAuthToken,
	}, nil
}

// GetVeniceConfig returns Venice.ai configuration.
func (l *ConfigLoader) GetVeniceConfig() (skills.VeniceConfig, error) {
	if l.config.VeniceAPIKey == "" {
		return skills.VeniceConfig{}, fmt.Errorf("Venice.ai API key not configured")
	}

	baseURL := l.config.VeniceBaseURL
	if baseURL == "" {
		baseURL = "https://api.venice.ai/api/v1"
	}

	model := l.config.VeniceModel
	if model == "" {
		model = "venice-uncensored"
	}

	imageModel := l.config.VeniceImageModel
	if imageModel == "" {
		imageModel = "lustify-sdxl" // Default NSFW image generation model
	}

	return skills.VeniceConfig{
		APIKey:     l.config.VeniceAPIKey,
		BaseURL:    baseURL,
		Model:      model,
		ImageModel: imageModel,
		Upscaler:   "upscaler",
	}, nil
}

// GetWeatherConfig returns weather skill configuration.
func (l *ConfigLoader) GetWeatherConfig() (skills.WeatherConfig, error) {
	return skills.WeatherConfig{
		DefaultZipCode: l.config.WeatherDefaultZipCode,
	}, nil
}

// GetTwitchConfig returns Twitch API configuration.
func (l *ConfigLoader) GetTwitchConfig() (skills.TwitchConfig, error) {
	if l.config.TwitchClientID == "" {
		return skills.TwitchConfig{}, fmt.Errorf("Twitch Client ID not configured")
	}

	defaultStreamer := l.config.TwitchDefaultStreamer
	if defaultStreamer == "" {
		defaultStreamer = "whykusanagi"
	}

	return skills.TwitchConfig{
		ClientID:        l.config.TwitchClientID,
		ClientSecret:    l.config.TwitchClientSecret,
		DefaultStreamer: defaultStreamer,
	}, nil
}

// GetYouTubeConfig returns YouTube API configuration.
func (l *ConfigLoader) GetYouTubeConfig() (skills.YouTubeConfig, error) {
	if l.config.YouTubeAPIKey == "" {
		return skills.YouTubeConfig{}, fmt.Errorf("YouTube API key not configured")
	}

	defaultChannel := l.config.YouTubeDefaultChannel
	if defaultChannel == "" {
		defaultChannel = "whykusanagi"
	}

	return skills.YouTubeConfig{
		APIKey:         l.config.YouTubeAPIKey,
		DefaultChannel: defaultChannel,
	}, nil
}

// GetTimeout returns the configured timeout as a duration.
func (c *Config) GetTimeout() time.Duration {
	if c.Timeout <= 0 {
		return 60 * time.Second
	}
	return time.Duration(c.Timeout) * time.Second
}
